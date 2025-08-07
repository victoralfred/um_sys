package portfolio

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
	"github.com/trading-engine/internal/core/services/execution"
	"github.com/trading-engine/pkg/types"
)

// TestPortfolioExecutionIntegration tests the complete integration between portfolio management and execution
func TestPortfolioExecutionIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup components
	portfolioRepo := NewMemoryRepository()
	portfolioService := NewService(portfolioRepo, nil, nil, DefaultServiceConfig())
	executionService := execution.NewOptimizedExecutionService(nil, nil)
	
	// Start execution service
	if err := executionService.Start(ctx); err != nil {
		t.Fatalf("Failed to start execution service: %v", err)
	}

	// Create integration
	integration := NewExecutionIntegration(
		portfolioService,
		executionService,
		DefaultIntegrationConfig(),
	)

	if err := integration.Start(ctx); err != nil {
		t.Fatalf("Failed to start integration: %v", err)
	}
	defer integration.Stop()

	// Create a test portfolio
	initialCapital := types.NewDecimalFromFloat(100000.0)
	portfolioReq := &ports.CreatePortfolioRequest{
		Name:           "Test Portfolio",
		InitialCapital: initialCapital,
		Currency:       "USD",
		Strategy:       "test",
	}

	portfolio, err := portfolioService.CreatePortfolio(ctx, portfolioReq)
	if err != nil {
		t.Fatalf("Failed to create portfolio: %v", err)
	}

	t.Run("BasicOrderSubmissionAndTracking", func(t *testing.T) {
		testBasicOrderSubmissionAndTracking(t, ctx, integration, portfolio.ID)
	})

	t.Run("CashBalanceUpdates", func(t *testing.T) {
		testCashBalanceUpdates(t, ctx, portfolioService, integration, portfolio.ID)
	})

	t.Run("PositionManagement", func(t *testing.T) {
		testPositionManagement(t, ctx, portfolioService, integration, portfolio.ID)
	})

	t.Run("RiskValidation", func(t *testing.T) {
		testRiskValidation(t, ctx, portfolioService, portfolio.ID)
	})

	t.Run("MetricsCalculation", func(t *testing.T) {
		testMetricsCalculation(t, ctx, portfolioService, portfolio.ID)
	})
}

func testBasicOrderSubmissionAndTracking(t *testing.T, ctx context.Context, integration *ExecutionIntegration, portfolioID string) {
	// Create test order (smaller to avoid position limits with 100K portfolio)
	order := createTestOrder("integration-test-1", "AAPL", domain.OrderSideBuy, domain.OrderTypeMarket, 50, 100.0)

	// Submit order with portfolio tracking
	result, err := integration.SubmitOrderWithPortfolio(ctx, portfolioID, order)
	if err != nil {
		t.Fatalf("Failed to submit order: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("Expected non-nil execution result")
	}

	if result.OrderID != order.ID {
		t.Errorf("Expected order ID %s, got %s", order.ID, result.OrderID)
	}

	if result.PortfolioID != portfolioID {
		t.Errorf("Expected portfolio ID %s, got %s", portfolioID, result.PortfolioID)
	}

	// Verify order tracking
	trackedPortfolio, tracked := integration.GetPortfolioForOrder(order.ID)
	if !tracked {
		t.Error("Order should be tracked")
	}

	if trackedPortfolio != portfolioID {
		t.Errorf("Expected tracked portfolio %s, got %s", portfolioID, trackedPortfolio)
	}

	// Verify order appears in tracked orders for portfolio
	trackedOrders := integration.GetTrackedOrders(portfolioID)
	found := false
	for _, orderID := range trackedOrders {
		if orderID == order.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Order should appear in tracked orders for portfolio")
	}

	t.Logf("✓ Basic order submission and tracking successful")
}

func testCashBalanceUpdates(t *testing.T, ctx context.Context, portfolioService *Service, integration *ExecutionIntegration, portfolioID string) {
	// Get initial cash balance
	initialBalance, err := portfolioService.GetCashBalance(ctx, portfolioID)
	if err != nil {
		t.Fatalf("Failed to get initial cash balance: %v", err)
	}

	// Create and process a mock order fill
	fill := &ports.OrderFill{
		OrderID:     "cash-test-order",
		PortfolioID: portfolioID,
		Symbol:      "MSFT",
		Side:        "BUY",
		Quantity:    types.NewDecimalFromFloat(50),
		Price:       types.NewDecimalFromFloat(300.0),
		Commission:  types.NewDecimalFromFloat(5.0),
		FillTime:    time.Now(),
		ExecutionID: "exec-1",
	}

	// Add order tracking first
	integration.orderMutex.Lock()
	integration.orderToPortfolio["cash-test-order"] = portfolioID
	integration.orderMutex.Unlock()
	
	// Process the fill
	if err := integration.OnOrderFilled(ctx, "cash-test-order", fill); err != nil {
		t.Fatalf("Failed to process order fill: %v", err)
	}

	// Allow some time for processing
	time.Sleep(100 * time.Millisecond)

	// Check updated cash balance
	newBalance, err := portfolioService.GetCashBalance(ctx, portfolioID)
	if err != nil {
		t.Fatalf("Failed to get updated cash balance: %v", err)
	}

	// Calculate expected balance change
	orderValue := fill.Quantity.Mul(fill.Price)          // 50 * 300 = 15000
	totalCost := orderValue.Add(fill.Commission)         // 15000 + 5 = 15005
	expectedBalance := initialBalance.Sub(totalCost)     // Should decrease by 15005

	if newBalance.Cmp(expectedBalance) != 0 {
		t.Errorf("Expected balance %v, got %v", expectedBalance, newBalance)
	}

	t.Logf("✓ Cash balance updates working correctly: %v -> %v", initialBalance, newBalance)
}

func testPositionManagement(t *testing.T, ctx context.Context, portfolioService *Service, integration *ExecutionIntegration, portfolioID string) {
	symbol := "GOOGL"
	
	// Create a buy fill
	buyFill := &ports.OrderFill{
		OrderID:     "position-test-buy",
		PortfolioID: portfolioID,
		Symbol:      symbol,
		Side:        "BUY",
		Quantity:    types.NewDecimalFromFloat(100),
		Price:       types.NewDecimalFromFloat(2800.0),
		Commission:  types.NewDecimalFromFloat(10.0),
		FillTime:    time.Now(),
		ExecutionID: "exec-buy",
	}

	// Add order tracking first
	integration.orderMutex.Lock()
	integration.orderToPortfolio["position-test-buy"] = portfolioID
	integration.orderMutex.Unlock()
	
	// Process the buy fill
	if err := integration.OnOrderFilled(ctx, "position-test-buy", buyFill); err != nil {
		t.Fatalf("Failed to process buy fill: %v", err)
	}

	// Allow processing time
	time.Sleep(100 * time.Millisecond)

	// Check position was created
	position, err := portfolioService.GetPosition(ctx, portfolioID, symbol)
	if err != nil {
		t.Fatalf("Failed to get position: %v", err)
	}

	if position.Quantity.Cmp(buyFill.Quantity) != 0 {
		t.Errorf("Expected position quantity %v, got %v", buyFill.Quantity, position.Quantity)
	}

	if position.Status != domain.PositionStatusOpen {
		t.Errorf("Expected position status %s, got %s", domain.PositionStatusOpen, position.Status)
	}

	// Create a partial sell fill
	sellFill := &ports.OrderFill{
		OrderID:     "position-test-sell",
		PortfolioID: portfolioID,
		Symbol:      symbol,
		Side:        "SELL",
		Quantity:    types.NewDecimalFromFloat(30),
		Price:       types.NewDecimalFromFloat(2850.0),
		Commission:  types.NewDecimalFromFloat(8.0),
		FillTime:    time.Now(),
		ExecutionID: "exec-sell",
	}

	// Add order tracking first
	integration.orderMutex.Lock()
	integration.orderToPortfolio["position-test-sell"] = portfolioID
	integration.orderMutex.Unlock()
	
	// Process the sell fill
	if err := integration.OnOrderFilled(ctx, "position-test-sell", sellFill); err != nil {
		t.Fatalf("Failed to process sell fill: %v", err)
	}

	// Allow processing time
	time.Sleep(100 * time.Millisecond)

	// Check position was updated
	position, err = portfolioService.GetPosition(ctx, portfolioID, symbol)
	if err != nil {
		t.Fatalf("Failed to get updated position: %v", err)
	}

	expectedQuantity := types.NewDecimalFromFloat(70) // 100 - 30
	if position.Quantity.Cmp(expectedQuantity) != 0 {
		t.Errorf("Expected position quantity %v, got %v", expectedQuantity, position.Quantity)
	}

	if position.Status != domain.PositionStatusOpen {
		t.Errorf("Position should still be open, got %s", position.Status)
	}

	t.Logf("✓ Position management working correctly")
}

func testRiskValidation(t *testing.T, ctx context.Context, portfolioService *Service, portfolioID string) {
	// Create a very large order that should exceed position limits
	largeOrder := createTestOrder("risk-test-1", "RISK", domain.OrderSideBuy, domain.OrderTypeMarket, 10000, 1000.0)

	// This should fail validation due to position size limits
	err := portfolioService.ValidateOrder(ctx, portfolioID, largeOrder)
	if err == nil {
		t.Error("Expected validation to fail for oversized order")
	}

	// Test insufficient cash scenario
	expensiveOrder := createTestOrder("risk-test-2", "EXPENSIVE", domain.OrderSideBuy, domain.OrderTypeMarket, 1000, 500.0)
	
	err = portfolioService.ValidateOrder(ctx, portfolioID, expensiveOrder)
	if err == nil {
		t.Error("Expected validation to fail for insufficient cash")
	}

	// Check risk limits
	riskCheck, err := portfolioService.CheckRiskLimits(ctx, portfolioID)
	if err != nil {
		t.Fatalf("Failed to check risk limits: %v", err)
	}

	if riskCheck == nil {
		t.Fatal("Expected non-nil risk check")
	}

	t.Logf("✓ Risk validation working correctly - found %d violations", len(riskCheck.Violations))
}

func testMetricsCalculation(t *testing.T, ctx context.Context, portfolioService *Service, portfolioID string) {
	// Calculate metrics
	metrics, err := portfolioService.CalculateMetrics(ctx, portfolioID)
	if err != nil {
		t.Fatalf("Failed to calculate metrics: %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Basic sanity checks
	if metrics.TotalValue.IsNegative() {
		t.Error("Total value should not be negative")
	}

	if metrics.CashBalance.IsNegative() {
		t.Error("Cash balance should not be negative")
	}

	t.Logf("Portfolio metrics:")
	t.Logf("  Total value: %v", metrics.TotalValue)
	t.Logf("  Cash balance: %v", metrics.CashBalance)
	t.Logf("  Market value: %v", metrics.MarketValue)
	t.Logf("  Total PnL: %v", metrics.TotalPnL)
	t.Logf("  Return: %v%%", metrics.ReturnPercentage)

	t.Logf("✓ Metrics calculation working correctly")
}

// Helper function to create test orders (same as in execution tests)
func createTestOrder(id, symbol string, side domain.OrderSide, orderType domain.OrderType, quantity, price float64) *domain.Order {
	qty := types.NewDecimalFromFloat(quantity)
	prc := types.NewDecimalFromFloat(price)
	
	minQty, _ := types.NewDecimal("0.01")
	maxQty, _ := types.NewDecimal("1000000")
	tickSize, _ := types.NewDecimal("0.01")
	
	asset := &domain.Asset{
		Symbol:      symbol,
		Name:        symbol + " Stock",
		AssetType:   domain.AssetTypeStock,
		Exchange:    "NASDAQ",
		Currency:    "USD",
		Precision:   2,
		MinQuantity: minQty,
		MaxQuantity: maxQty,
		TickSize:    tickSize,
		IsActive:    true,
	}
	
	return &domain.Order{
		ID:       id,
		Asset:    asset,
		Type:     orderType,
		Side:     side,
		Status:   domain.OrderStatusPending,
		Quantity: qty,
		Price:    prc,
		TimeInForce: domain.TimeInForceGTC,
	}
}

// TestPortfolioEnabledExecutionService tests the wrapper service
func TestPortfolioEnabledExecutionService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup components
	portfolioRepo := NewMemoryRepository()
	portfolioService := NewService(portfolioRepo, nil, nil, DefaultServiceConfig())
	executionService := execution.NewOptimizedExecutionService(nil, nil)
	
	if err := executionService.Start(ctx); err != nil {
		t.Fatalf("Failed to start execution service: %v", err)
	}

	// Create integration
	integration := NewExecutionIntegration(portfolioService, executionService, DefaultIntegrationConfig())
	if err := integration.Start(ctx); err != nil {
		t.Fatalf("Failed to start integration: %v", err)
	}
	defer integration.Stop()

	// Create portfolio
	initialCapital := types.NewDecimalFromFloat(50000.0)
	portfolioReq := &ports.CreatePortfolioRequest{
		Name:           "Wrapper Test Portfolio",
		InitialCapital: initialCapital,
		Currency:       "USD",
	}

	portfolio, err := portfolioService.CreatePortfolio(ctx, portfolioReq)
	if err != nil {
		t.Fatalf("Failed to create portfolio: %v", err)
	}

	// Create portfolio-enabled execution service
	wrapperService := NewPortfolioEnabledExecutionService(executionService, integration, portfolio.ID)

	// Test order submission through wrapper (smaller order to avoid position limits)
	order := createTestOrder("wrapper-test-1", "TSLA", domain.OrderSideBuy, domain.OrderTypeMarket, 5, 500.0)

	result, err := wrapperService.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to submit order through wrapper: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.PortfolioID != portfolio.ID {
		t.Errorf("Expected portfolio ID %s in result, got %s", portfolio.ID, result.PortfolioID)
	}

	// Test order status retrieval
	_, err = wrapperService.GetOrderStatus(ctx, order.ID)
	if err != nil {
		t.Fatalf("Failed to get order status: %v", err)
	}

	// Test metrics retrieval
	metrics := wrapperService.GetMetrics()
	if metrics.TotalOrdersProcessed == 0 {
		t.Error("Expected some orders to be processed")
	}

	t.Logf("✓ Portfolio-enabled execution service working correctly")
}

// TestPortfolioRepository tests the memory repository implementation
func TestPortfolioRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	t.Run("BasicCRUD", func(t *testing.T) {
		testRepositoryBasicCRUD(t, ctx, repo)
	})

	t.Run("PositionManagement", func(t *testing.T) {
		testRepositoryPositionManagement(t, ctx, repo)
	})

	t.Run("SnapshotManagement", func(t *testing.T) {
		testRepositorySnapshotManagement(t, ctx, repo)
	})

	t.Run("FilteringAndSorting", func(t *testing.T) {
		testRepositoryFilteringAndSorting(t, ctx, repo)
	})
}

func testRepositoryBasicCRUD(t *testing.T, ctx context.Context, repo *MemoryRepository) {
	// Test creation
	initialCapital := types.NewDecimalFromFloat(25000.0)
	portfolio, err := domain.NewPortfolio("test-repo-1", "Repository Test", initialCapital)
	if err != nil {
		t.Fatalf("Failed to create portfolio: %v", err)
	}

	// Save
	if err := repo.Save(ctx, portfolio); err != nil {
		t.Fatalf("Failed to save portfolio: %v", err)
	}

	// Retrieve
	retrieved, err := repo.FindByID(ctx, portfolio.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve portfolio: %v", err)
	}

	if retrieved.ID != portfolio.ID {
		t.Errorf("Expected ID %s, got %s", portfolio.ID, retrieved.ID)
	}

	if retrieved.Name != portfolio.Name {
		t.Errorf("Expected name %s, got %s", portfolio.Name, retrieved.Name)
	}

	// Update
	retrieved.Name = "Updated Name"
	if err := repo.Save(ctx, retrieved); err != nil {
		t.Fatalf("Failed to update portfolio: %v", err)
	}

	// Verify update
	updated, err := repo.FindByID(ctx, portfolio.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated portfolio: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected updated name 'Updated Name', got %s", updated.Name)
	}

	// Delete
	if err := repo.Delete(ctx, portfolio.ID); err != nil {
		t.Fatalf("Failed to delete portfolio: %v", err)
	}

	// Verify deletion
	_, err = repo.FindByID(ctx, portfolio.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted portfolio")
	}

	t.Logf("✓ Repository basic CRUD operations working")
}

func testRepositoryPositionManagement(t *testing.T, ctx context.Context, repo *MemoryRepository) {
	// Create portfolio
	portfolio, _ := domain.NewPortfolio("pos-test", "Position Test", types.NewDecimalFromFloat(10000))
	repo.Save(ctx, portfolio)

	// Create position
	asset := &domain.Asset{Symbol: "AAPL"}
	position := &domain.Position{
		ID:        "pos-1",
		Asset:     asset,
		Status:    domain.PositionStatusOpen,
		Quantity:  types.NewDecimalFromFloat(100),
		OpenedAt: time.Now(),
	}

	// Save position
	if err := repo.SavePosition(ctx, portfolio.ID, position); err != nil {
		t.Fatalf("Failed to save position: %v", err)
	}

	// Find positions
	positions, err := repo.FindPositions(ctx, portfolio.ID)
	if err != nil {
		t.Fatalf("Failed to find positions: %v", err)
	}

	if len(positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(positions))
	}

	// Find position by symbol
	foundPosition, err := repo.FindPositionBySymbol(ctx, portfolio.ID, "AAPL")
	if err != nil {
		t.Fatalf("Failed to find position by symbol: %v", err)
	}

	if foundPosition.ID != position.ID {
		t.Errorf("Expected position ID %s, got %s", position.ID, foundPosition.ID)
	}

	t.Logf("✓ Repository position management working")
}

func testRepositorySnapshotManagement(t *testing.T, ctx context.Context, repo *MemoryRepository) {
	// Create portfolio
	portfolio, _ := domain.NewPortfolio("snapshot-test", "Snapshot Test", types.NewDecimalFromFloat(10000))
	repo.Save(ctx, portfolio)

	// Create snapshots
	now := time.Now()
	for i := 0; i < 5; i++ {
		snapshot := &ports.PortfolioSnapshot{
			ID:          fmt.Sprintf("snap-%d", i),
			PortfolioID: portfolio.ID,
			Timestamp:   now.Add(time.Duration(i) * time.Hour),
			TotalValue:  types.NewDecimalFromFloat(10000 + float64(i*100)),
		}

		if err := repo.SaveSnapshot(ctx, snapshot); err != nil {
			t.Fatalf("Failed to save snapshot %d: %v", i, err)
		}
	}

	// Get snapshots
	from := now
	to := now.Add(6 * time.Hour)
	snapshots, err := repo.GetSnapshots(ctx, portfolio.ID, from, to)
	if err != nil {
		t.Fatalf("Failed to get snapshots: %v", err)
	}

	if len(snapshots) != 5 {
		t.Errorf("Expected 5 snapshots, got %d", len(snapshots))
	}

	// Verify chronological order
	for i := 1; i < len(snapshots); i++ {
		if snapshots[i].Timestamp.Before(snapshots[i-1].Timestamp) {
			t.Error("Snapshots should be in chronological order")
		}
	}

	t.Logf("✓ Repository snapshot management working")
}

func testRepositoryFilteringAndSorting(t *testing.T, ctx context.Context, repo *MemoryRepository) {
	repo.Clear() // Start fresh

	// Create multiple portfolios
	for i := 0; i < 10; i++ {
		capital := types.NewDecimalFromFloat(float64(1000 * (i + 1)))
		portfolio, _ := domain.NewPortfolio(fmt.Sprintf("filter-test-%d", i), fmt.Sprintf("Portfolio %d", i), capital)
		
		// Set different statuses
		if i%2 == 0 {
			portfolio.Status = domain.PortfolioStatusActive
		} else {
			portfolio.Status = domain.PortfolioStatusSuspended
		}
		
		repo.Save(ctx, portfolio)
		time.Sleep(time.Millisecond) // Ensure different creation times
	}

	// Test status filter
	activeFilter := &ports.PortfolioFilter{
		Status: func() *domain.PortfolioStatus { s := domain.PortfolioStatusActive; return &s }(),
	}
	
	activePortfolios, err := repo.FindAll(ctx, activeFilter)
	if err != nil {
		t.Fatalf("Failed to filter by status: %v", err)
	}

	if len(activePortfolios) != 5 {
		t.Errorf("Expected 5 active portfolios, got %d", len(activePortfolios))
	}

	// Test limit and offset
	limitFilter := &ports.PortfolioFilter{
		Limit:  3,
		Offset: 2,
	}

	limitedPortfolios, err := repo.FindAll(ctx, limitFilter)
	if err != nil {
		t.Fatalf("Failed to apply limit and offset: %v", err)
	}

	if len(limitedPortfolios) != 3 {
		t.Errorf("Expected 3 portfolios with limit, got %d", len(limitedPortfolios))
	}

	// Test capital filter
	minCapital := types.NewDecimalFromFloat(5000)
	capitalFilter := &ports.PortfolioFilter{
		MinCapital: &minCapital,
	}

	expensivePortfolios, err := repo.FindAll(ctx, capitalFilter)
	if err != nil {
		t.Fatalf("Failed to filter by capital: %v", err)
	}

	if len(expensivePortfolios) != 6 { // Portfolios 4-9 have capital >= 5000
		t.Errorf("Expected 6 expensive portfolios, got %d", len(expensivePortfolios))
	}

	t.Logf("✓ Repository filtering and sorting working")
}