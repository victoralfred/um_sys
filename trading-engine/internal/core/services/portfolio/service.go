package portfolio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
	"github.com/trading-engine/pkg/types"
)

// ExecutionService interface to avoid circular imports
type ExecutionService = interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SubmitOrder(ctx context.Context, order *domain.Order) (*ports.ExecutionResult, error)
	GetOrderStatus(ctx context.Context, orderID string) (*domain.Order, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetMetrics() ports.ExecutionMetrics
}

// Service implements the PortfolioManager interface
type Service struct {
	repository       ports.PortfolioRepository
	executionService ExecutionService
	riskManager      ports.RiskManager

	// In-memory cache for active portfolios
	portfolioCache map[string]*domain.Portfolio
	cacheMutex     sync.RWMutex

	// Event handlers
	onPositionUpdate func(*domain.Portfolio, *domain.Position)

	// Configuration
	config ServiceConfig
}

// ServiceConfig contains configuration for the portfolio service
type ServiceConfig struct {
	EnableCache           bool          `json:"enable_cache"`
	CacheSize             int           `json:"cache_size"`
	SnapshotInterval      time.Duration `json:"snapshot_interval"`
	MetricsUpdateInterval time.Duration `json:"metrics_update_interval"`

	// Risk limits
	DefaultRiskLimits ports.RiskLimits `json:"default_risk_limits"`

	// Performance tracking
	TrackDailyReturns bool `json:"track_daily_returns"`
	TrackDrawdowns    bool `json:"track_drawdowns"`
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() ServiceConfig {
	maxDrawdown, _ := types.NewDecimal("0.20")      // 20%
	maxPosition, _ := types.NewDecimal("0.10")      // 10%
	maxConcentration, _ := types.NewDecimal("0.30") // 30%
	varLimit, _ := types.NewDecimal("0.05")         // 5%
	maxLeverage, _ := types.NewDecimal("2.0")       // 2x

	return ServiceConfig{
		EnableCache:           true,
		CacheSize:             1000,
		SnapshotInterval:      time.Hour,
		MetricsUpdateInterval: time.Minute * 15,

		DefaultRiskLimits: ports.RiskLimits{
			MaxDrawdownPercent:   maxDrawdown,
			MaxPositionWeight:    maxPosition,
			MaxConcentrationRisk: maxConcentration,
			VaR95Limit:           varLimit,
			MaxLeverage:          maxLeverage,
		},

		TrackDailyReturns: true,
		TrackDrawdowns:    true,
	}
}

// NewService creates a new portfolio management service
func NewService(
	repository ports.PortfolioRepository,
	executionService ExecutionService,
	riskManager ports.RiskManager,
	config ServiceConfig,
) *Service {
	return &Service{
		repository:       repository,
		executionService: executionService,
		riskManager:      riskManager,
		portfolioCache:   make(map[string]*domain.Portfolio),
		config:           config,
	}
}

// CreatePortfolio creates a new portfolio
func (s *Service) CreatePortfolio(ctx context.Context, req *ports.CreatePortfolioRequest) (*domain.Portfolio, error) {
	// Generate unique ID
	portfolioID := fmt.Sprintf("portfolio_%d", time.Now().UnixNano())

	// Create portfolio domain object
	portfolio, err := domain.NewPortfolio(portfolioID, req.Name, req.InitialCapital)
	if err != nil {
		return nil, fmt.Errorf("failed to create portfolio: %w", err)
	}

	// Save to repository
	if err := s.repository.Save(ctx, portfolio); err != nil {
		return nil, fmt.Errorf("failed to save portfolio: %w", err)
	}

	// Add to cache if enabled
	if s.config.EnableCache {
		s.addToCache(portfolio)
	}

	return portfolio, nil
}

// GetPortfolio retrieves a portfolio by ID
func (s *Service) GetPortfolio(ctx context.Context, portfolioID string) (*domain.Portfolio, error) {
	// Check cache first if enabled
	if s.config.EnableCache {
		if portfolio := s.getFromCache(portfolioID); portfolio != nil {
			return portfolio, nil
		}
	}

	// Load from repository
	portfolio, err := s.repository.FindByID(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to find portfolio %s: %w", portfolioID, err)
	}

	// Add to cache if enabled
	if s.config.EnableCache && portfolio != nil {
		s.addToCache(portfolio)
	}

	return portfolio, nil
}

// UpdatePortfolio updates an existing portfolio
func (s *Service) UpdatePortfolio(ctx context.Context, portfolio *domain.Portfolio) error {
	portfolio.UpdatedAt = time.Now()

	if err := s.repository.Save(ctx, portfolio); err != nil {
		return fmt.Errorf("failed to update portfolio: %w", err)
	}

	// Update cache if enabled
	if s.config.EnableCache {
		s.addToCache(portfolio)
	}

	return nil
}

// ClosePortfolio closes a portfolio
func (s *Service) ClosePortfolio(ctx context.Context, portfolioID string) error {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return err
	}

	// Check if all positions are closed
	for _, position := range portfolio.Positions {
		if !position.Quantity.IsZero() {
			return fmt.Errorf("cannot close portfolio with open positions")
		}
	}

	// Update status
	now := time.Now()
	portfolio.Status = domain.PortfolioStatusClosed
	portfolio.ClosedAt = &now
	portfolio.UpdatedAt = now

	if err := s.UpdatePortfolio(ctx, portfolio); err != nil {
		return err
	}

	// Remove from cache
	if s.config.EnableCache {
		s.removeFromCache(portfolioID)
	}

	return nil
}

// ListPortfolios returns a list of portfolios matching the filter
func (s *Service) ListPortfolios(ctx context.Context, filter *ports.PortfolioFilter) ([]*domain.Portfolio, error) {
	return s.repository.FindAll(ctx, filter)
}

// AddPosition adds a new position to a portfolio
func (s *Service) AddPosition(ctx context.Context, portfolioID string, position *domain.Position) error {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return err
	}

	if err := portfolio.AddPosition(position); err != nil {
		return err
	}

	// Save position
	if err := s.repository.SavePosition(ctx, portfolioID, position); err != nil {
		return fmt.Errorf("failed to save position: %w", err)
	}

	// Update portfolio
	if err := s.UpdatePortfolio(ctx, portfolio); err != nil {
		return err
	}

	// Trigger callback if set
	if s.onPositionUpdate != nil {
		s.onPositionUpdate(portfolio, position)
	}

	return nil
}

// UpdatePosition updates an existing position
func (s *Service) UpdatePosition(ctx context.Context, portfolioID string, position *domain.Position) error {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return err
	}

	// Update position in portfolio maps
	portfolio.Positions[position.ID] = position
	portfolio.AssetPositions[position.Asset.Symbol] = position

	// Save position
	if err := s.repository.SavePosition(ctx, portfolioID, position); err != nil {
		return fmt.Errorf("failed to save position: %w", err)
	}

	// Update portfolio metrics
	if err := s.updatePortfolioMetrics(ctx, portfolio); err != nil {
		return fmt.Errorf("failed to update portfolio metrics: %w", err)
	}

	// Update portfolio
	if err := s.UpdatePortfolio(ctx, portfolio); err != nil {
		return err
	}

	// Trigger callback if set
	if s.onPositionUpdate != nil {
		s.onPositionUpdate(portfolio, position)
	}

	return nil
}

// ClosePosition closes a position in a portfolio
func (s *Service) ClosePosition(ctx context.Context, portfolioID string, positionID string) error {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return err
	}

	position, exists := portfolio.Positions[positionID]
	if !exists {
		return fmt.Errorf("position %s not found in portfolio %s", positionID, portfolioID)
	}

	// Mark position as closed
	now := time.Now()
	position.Status = domain.PositionStatusClosed
	position.ClosedAt = &now
	position.UpdatedAt = now

	// Remove from asset positions map
	delete(portfolio.AssetPositions, position.Asset.Symbol)

	return s.UpdatePosition(ctx, portfolioID, position)
}

// GetPosition retrieves a position by symbol
func (s *Service) GetPosition(ctx context.Context, portfolioID string, symbol string) (*domain.Position, error) {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	position, exists := portfolio.AssetPositions[symbol]
	if !exists {
		return nil, fmt.Errorf("position for symbol %s not found", symbol)
	}

	return position, nil
}

// OnOrderExecuted handles execution results from the execution engine
func (s *Service) OnOrderExecuted(ctx context.Context, execution *ports.ExecutionResult) error {
	// Find the portfolio that owns this order
	// In a real implementation, you'd store the portfolio ID with the order
	// For now, we'll assume it's passed in the execution result
	portfolioID := execution.PortfolioID
	if portfolioID == "" {
		return fmt.Errorf("portfolio ID not provided in execution result")
	}

	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return err
	}

	// Convert execution to position update
	return s.processExecution(ctx, portfolio, execution)
}

// OnOrderFilled handles order fills
func (s *Service) OnOrderFilled(ctx context.Context, fill *ports.OrderFill) error {
	portfolio, err := s.GetPortfolio(ctx, fill.PortfolioID)
	if err != nil {
		return err
	}

	return s.processFill(ctx, portfolio, fill)
}

// CalculateMetrics recalculates portfolio metrics
func (s *Service) CalculateMetrics(ctx context.Context, portfolioID string) (*domain.PortfolioMetrics, error) {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	if err := s.updatePortfolioMetrics(ctx, portfolio); err != nil {
		return nil, err
	}

	return &portfolio.Metrics, nil
}

// GetPerformanceReport generates a comprehensive performance report
func (s *Service) GetPerformanceReport(ctx context.Context, portfolioID string, from, to time.Time) (*ports.PerformanceReport, error) {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	// Get historical snapshots
	snapshots, err := s.repository.GetSnapshots(ctx, portfolioID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio snapshots: %w", err)
	}

	return s.generatePerformanceReport(portfolio, snapshots, from, to), nil
}

// ValidateOrder validates an order against portfolio risk limits
func (s *Service) ValidateOrder(ctx context.Context, portfolioID string, order *domain.Order) error {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return err
	}

	// Check cash availability for buy orders
	if order.Side == domain.OrderSideBuy {
		orderValue := order.Quantity.Mul(order.Price)
		if portfolio.CashBalance.Cmp(orderValue) < 0 {
			return fmt.Errorf("insufficient cash: required %v, available %v",
				orderValue, portfolio.CashBalance)
		}
	}

	// Check position size limits
	return s.validatePositionLimits(portfolio, order)
}

// CheckRiskLimits performs comprehensive risk limit checks
func (s *Service) CheckRiskLimits(ctx context.Context, portfolioID string) (*ports.RiskCheck, error) {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	riskCheck := &ports.RiskCheck{
		Passed:            true,
		Violations:        []ports.RiskViolation{},
		CurrentLimits:     s.config.DefaultRiskLimits,
		PortfolioValue:    portfolio.Metrics.TotalValue,
		MaxDrawdown:       portfolio.Metrics.MaxDrawdown,
		ConcentrationRisk: portfolio.Metrics.ConcentrationRisk,
		CheckedAt:         time.Now(),
	}

	// Check drawdown limits (ensure MaxDrawdownPercent is not nil)
	if !portfolio.Metrics.MaxDrawdownPercent.IsZero() && portfolio.Metrics.MaxDrawdownPercent.Cmp(s.config.DefaultRiskLimits.MaxDrawdownPercent) > 0 {
		riskCheck.Passed = false
		riskCheck.Violations = append(riskCheck.Violations, ports.RiskViolation{
			Type:        ports.RiskViolationMaxDrawdown,
			Description: "Maximum drawdown exceeded",
			Current:     portfolio.Metrics.MaxDrawdownPercent,
			Limit:       s.config.DefaultRiskLimits.MaxDrawdownPercent,
			Severity:    ports.ViolationSeverityHigh,
		})
	}

	// Check concentration risk (ensure ConcentrationRisk is not nil)
	if !portfolio.Metrics.ConcentrationRisk.IsZero() && portfolio.Metrics.ConcentrationRisk.Cmp(s.config.DefaultRiskLimits.MaxConcentrationRisk) > 0 {
		riskCheck.Passed = false
		riskCheck.Violations = append(riskCheck.Violations, ports.RiskViolation{
			Type:        ports.RiskViolationConcentration,
			Description: "Portfolio concentration risk exceeded",
			Current:     portfolio.Metrics.ConcentrationRisk,
			Limit:       s.config.DefaultRiskLimits.MaxConcentrationRisk,
			Severity:    ports.ViolationSeverityMedium,
		})
	}

	return riskCheck, nil
}

// UpdateCashBalance updates the cash balance of a portfolio
func (s *Service) UpdateCashBalance(ctx context.Context, portfolioID string, amount types.Decimal, reason string) error {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return err
	}

	newBalance := portfolio.CashBalance.Add(amount)
	if newBalance.IsNegative() {
		return fmt.Errorf("insufficient cash: current balance %v, requested change %v",
			portfolio.CashBalance, amount)
	}

	portfolio.CashBalance = newBalance
	portfolio.UpdatedAt = time.Now()

	return s.UpdatePortfolio(ctx, portfolio)
}

// GetCashBalance returns the current cash balance
func (s *Service) GetCashBalance(ctx context.Context, portfolioID string) (types.Decimal, error) {
	portfolio, err := s.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return types.Decimal{}, err
	}

	return portfolio.CashBalance, nil
}

// Cache management methods
func (s *Service) addToCache(portfolio *domain.Portfolio) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Simple LRU eviction if cache is full
	if len(s.portfolioCache) >= s.config.CacheSize {
		// Remove oldest entry (in a real implementation, use proper LRU)
		for id := range s.portfolioCache {
			delete(s.portfolioCache, id)
			break
		}
	}

	s.portfolioCache[portfolio.ID] = portfolio
}

func (s *Service) getFromCache(portfolioID string) *domain.Portfolio {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	return s.portfolioCache[portfolioID]
}

func (s *Service) removeFromCache(portfolioID string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	delete(s.portfolioCache, portfolioID)
}

// Helper methods
func (s *Service) processExecution(ctx context.Context, portfolio *domain.Portfolio, execution *ports.ExecutionResult) error {
	// This would update positions based on execution results
	// Implementation depends on the specific execution result format
	return nil
}

func (s *Service) processFill(ctx context.Context, portfolio *domain.Portfolio, fill *ports.OrderFill) error {
	// Find or create position for the symbol
	position, exists := portfolio.AssetPositions[fill.Symbol]
	if !exists {
		// Create new position using the domain constructor
		asset := &domain.Asset{
			Symbol:    fill.Symbol,
			Name:      fill.Symbol + " Asset",
			AssetType: domain.AssetTypeStock,
			Currency:  "USD",
			IsActive:  true,
		} // In real implementation, fetch from asset service

		initialTransaction := domain.PositionTransaction{
			ID:        fmt.Sprintf("tx_%d", time.Now().UnixNano()),
			Type:      fill.Side,
			Quantity:  fill.Quantity,
			Price:     fill.Price,
			Fee:       fill.Commission,
			Timestamp: fill.FillTime,
		}

		newPosition, err := domain.NewPosition(
			fmt.Sprintf("pos_%s_%d", fill.Symbol, time.Now().UnixNano()),
			asset,
			initialTransaction,
		)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		position = newPosition
		portfolio.AssetPositions[fill.Symbol] = position
		portfolio.Positions[position.ID] = position
	} else {
		// Update existing position with new transaction
		transaction := domain.PositionTransaction{
			ID:        fmt.Sprintf("tx_%d", time.Now().UnixNano()),
			Type:      fill.Side,
			Quantity:  fill.Quantity,
			Price:     fill.Price,
			Fee:       fill.Commission,
			Timestamp: fill.FillTime,
		}

		// Update position based on fill
		if fill.Side == "BUY" {
			totalCost := position.Quantity.Mul(position.AvgEntryPrice).Add(fill.Quantity.Mul(fill.Price))
			position.Quantity = position.Quantity.Add(fill.Quantity)
			if !position.Quantity.IsZero() {
				position.AvgEntryPrice = totalCost.Div(position.Quantity)
			}
		} else { // SELL
			position.Quantity = position.Quantity.Sub(fill.Quantity)
			if position.Quantity.IsZero() {
				position.Status = domain.PositionStatusClosed
				now := time.Now()
				position.ClosedAt = &now
			}
		}

		// Add transaction to position
		position.Transactions = append(position.Transactions, transaction)
		position.UpdatedAt = time.Now()
	}

	// Update cash balance
	orderValue := fill.Quantity.Mul(fill.Price)
	var cashChange types.Decimal
	if fill.Side == "BUY" {
		cashChange = orderValue.Mul(types.NewDecimalFromFloat(-1)) // Negative for buy
	} else {
		cashChange = orderValue // Positive for sell
	}
	cashChange = cashChange.Sub(fill.Commission) // Subtract commission

	portfolio.CashBalance = portfolio.CashBalance.Add(cashChange)

	// Update position
	return s.UpdatePosition(ctx, portfolio.ID, position)
}

func (s *Service) updatePortfolioMetrics(ctx context.Context, portfolio *domain.Portfolio) error {
	// Initialize all decimal fields to avoid nil pointer issues
	zero := types.NewDecimalFromFloat(0)
	if portfolio.Metrics.MarketValue.String() == "" {
		portfolio.Metrics.MarketValue = zero
	}
	if portfolio.Metrics.TotalCost.String() == "" {
		portfolio.Metrics.TotalCost = zero
	}
	if portfolio.Metrics.RealizedPnL.String() == "" {
		portfolio.Metrics.RealizedPnL = zero
	}
	if portfolio.Metrics.UnrealizedPnL.String() == "" {
		portfolio.Metrics.UnrealizedPnL = zero
	}
	if portfolio.Metrics.TotalPnL.String() == "" {
		portfolio.Metrics.TotalPnL = zero
	}
	if portfolio.Metrics.MaxDrawdown.String() == "" {
		portfolio.Metrics.MaxDrawdown = zero
	}
	if portfolio.Metrics.MaxDrawdownPercent.String() == "" {
		portfolio.Metrics.MaxDrawdownPercent = zero
	}
	if portfolio.Metrics.ConcentrationRisk.String() == "" {
		portfolio.Metrics.ConcentrationRisk = zero
	}
	// Calculate market value
	marketValue := types.NewDecimalFromFloat(0)
	for _, position := range portfolio.Positions {
		if position.Status == domain.PositionStatusOpen {
			// In real implementation, get current market price
			positionValue := position.Quantity.Mul(position.CurrentPrice)
			marketValue = marketValue.Add(positionValue)
		}
	}

	// Update metrics
	portfolio.Metrics.CashBalance = portfolio.CashBalance
	portfolio.Metrics.MarketValue = marketValue
	portfolio.Metrics.TotalValue = portfolio.CashBalance.Add(marketValue)

	// Calculate PnL
	totalCost := types.NewDecimalFromFloat(0)
	for _, position := range portfolio.Positions {
		if position.Status == domain.PositionStatusOpen {
			positionCost := position.Quantity.Mul(position.AvgEntryPrice)
			totalCost = totalCost.Add(positionCost)
		}
	}
	portfolio.Metrics.TotalCost = totalCost
	portfolio.Metrics.UnrealizedPnL = marketValue.Sub(totalCost)
	portfolio.Metrics.TotalPnL = portfolio.Metrics.RealizedPnL.Add(portfolio.Metrics.UnrealizedPnL)

	// Calculate return percentage
	if !portfolio.InitialCapital.IsZero() {
		portfolio.Metrics.ReturnPercentage = portfolio.Metrics.TotalPnL.Div(portfolio.InitialCapital).Mul(types.NewDecimalFromFloat(100))
	}

	// Update peak value and drawdown (initialize if needed)
	if portfolio.Metrics.PeakValue.IsZero() || portfolio.Metrics.TotalValue.Cmp(portfolio.Metrics.PeakValue) > 0 {
		portfolio.Metrics.PeakValue = portfolio.Metrics.TotalValue
	}

	currentDrawdown := portfolio.Metrics.PeakValue.Sub(portfolio.Metrics.TotalValue)
	portfolio.Metrics.CurrentDrawdown = currentDrawdown

	if currentDrawdown.Cmp(portfolio.Metrics.MaxDrawdown) > 0 {
		portfolio.Metrics.MaxDrawdown = currentDrawdown
		if !portfolio.Metrics.PeakValue.IsZero() {
			portfolio.Metrics.MaxDrawdownPercent = currentDrawdown.Div(portfolio.Metrics.PeakValue)
		}
	}

	portfolio.Metrics.UpdatedAt = time.Now()
	return nil
}

func (s *Service) validatePositionLimits(portfolio *domain.Portfolio, order *domain.Order) error {
	// Check maximum position weight
	orderValue := order.Quantity.Mul(order.Price)
	totalValue := portfolio.Metrics.TotalValue

	if !totalValue.IsZero() {
		positionWeight := orderValue.Div(totalValue)
		if positionWeight.Cmp(s.config.DefaultRiskLimits.MaxPositionWeight) > 0 {
			return fmt.Errorf("position weight %v exceeds limit %v",
				positionWeight, s.config.DefaultRiskLimits.MaxPositionWeight)
		}
	}

	return nil
}

func (s *Service) generatePerformanceReport(portfolio *domain.Portfolio, snapshots []*ports.PortfolioSnapshot, from, to time.Time) *ports.PerformanceReport {
	report := &ports.PerformanceReport{
		PortfolioID: portfolio.ID,
		Period: ports.Period{
			From: from,
			To:   to,
		},
		Metrics:     &portfolio.Metrics,
		Returns:     []ports.DailyReturn{},
		Drawdowns:   []ports.DrawdownPeriod{},
		Trades:      []ports.TradeAnalysis{},
		RiskMetrics: ports.PortfolioRiskMetrics{},
		Attribution: ports.PerformanceAttribution{},
		GeneratedAt: time.Now(),
	}

	// Generate daily returns from snapshots
	for i, snapshot := range snapshots {
		if i > 0 {
			prevValue := snapshots[i-1].TotalValue
			currentValue := snapshot.TotalValue

			if !prevValue.IsZero() {
				dailyReturn := currentValue.Sub(prevValue).Div(prevValue)
				report.Returns = append(report.Returns, ports.DailyReturn{
					Date:   snapshot.Timestamp,
					Return: dailyReturn,
					Value:  currentValue,
				})
			}
		}
	}

	// Calculate risk metrics
	report.RiskMetrics.MaxDrawdown = portfolio.Metrics.MaxDrawdownPercent
	// Other risk metrics would be calculated here

	return report
}
