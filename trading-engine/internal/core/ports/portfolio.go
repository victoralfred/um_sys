package ports

import (
	"context"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/pkg/types"
)

// PortfolioManager defines the interface for portfolio management operations
type PortfolioManager interface {
	// Portfolio lifecycle operations
	CreatePortfolio(ctx context.Context, req *CreatePortfolioRequest) (*domain.Portfolio, error)
	GetPortfolio(ctx context.Context, portfolioID string) (*domain.Portfolio, error)
	UpdatePortfolio(ctx context.Context, portfolio *domain.Portfolio) error
	ClosePortfolio(ctx context.Context, portfolioID string) error
	ListPortfolios(ctx context.Context, filter *PortfolioFilter) ([]*domain.Portfolio, error)

	// Position management
	AddPosition(ctx context.Context, portfolioID string, position *domain.Position) error
	UpdatePosition(ctx context.Context, portfolioID string, position *domain.Position) error
	ClosePosition(ctx context.Context, portfolioID string, positionID string) error
	GetPosition(ctx context.Context, portfolioID string, symbol string) (*domain.Position, error)

	// Order integration - called by execution engine when orders are executed
	OnOrderExecuted(ctx context.Context, execution *ExecutionResult) error
	OnOrderFilled(ctx context.Context, fill *OrderFill) error
	
	// Portfolio metrics and analysis
	CalculateMetrics(ctx context.Context, portfolioID string) (*domain.PortfolioMetrics, error)
	GetPerformanceReport(ctx context.Context, portfolioID string, from, to time.Time) (*PerformanceReport, error)
	
	// Risk management
	ValidateOrder(ctx context.Context, portfolioID string, order *domain.Order) error
	CheckRiskLimits(ctx context.Context, portfolioID string) (*RiskCheck, error)
	
	// Cash management
	UpdateCashBalance(ctx context.Context, portfolioID string, amount types.Decimal, reason string) error
	GetCashBalance(ctx context.Context, portfolioID string) (types.Decimal, error)
}

// PortfolioRepository defines the interface for portfolio data persistence
type PortfolioRepository interface {
	// Basic CRUD operations
	Save(ctx context.Context, portfolio *domain.Portfolio) error
	FindByID(ctx context.Context, id string) (*domain.Portfolio, error)
	FindAll(ctx context.Context, filter *PortfolioFilter) ([]*domain.Portfolio, error)
	Delete(ctx context.Context, id string) error
	
	// Position operations
	SavePosition(ctx context.Context, portfolioID string, position *domain.Position) error
	FindPositions(ctx context.Context, portfolioID string) ([]*domain.Position, error)
	FindPositionBySymbol(ctx context.Context, portfolioID string, symbol string) (*domain.Position, error)
	
	// Historical data
	SaveSnapshot(ctx context.Context, snapshot *PortfolioSnapshot) error
	GetSnapshots(ctx context.Context, portfolioID string, from, to time.Time) ([]*PortfolioSnapshot, error)
}

// CreatePortfolioRequest contains the parameters for creating a new portfolio
type CreatePortfolioRequest struct {
	Name           string        `json:"name"`
	InitialCapital types.Decimal `json:"initial_capital"`
	Currency       string        `json:"currency"`
	Strategy       string        `json:"strategy,omitempty"`
	RiskProfile    string        `json:"risk_profile,omitempty"`
}

// PortfolioFilter defines filtering criteria for portfolio queries
type PortfolioFilter struct {
	Status     *domain.PortfolioStatus `json:"status,omitempty"`
	Strategy   *string                 `json:"strategy,omitempty"`
	MinCapital *types.Decimal          `json:"min_capital,omitempty"`
	MaxCapital *types.Decimal          `json:"max_capital,omitempty"`
	CreatedAfter  *time.Time           `json:"created_after,omitempty"`
	CreatedBefore *time.Time           `json:"created_before,omitempty"`
	Limit      int                     `json:"limit,omitempty"`
	Offset     int                     `json:"offset,omitempty"`
}

// PerformanceReport contains comprehensive portfolio performance analysis
type PerformanceReport struct {
	PortfolioID   string                   `json:"portfolio_id"`
	Period        Period                   `json:"period"`
	Metrics       *domain.PortfolioMetrics `json:"metrics"`
	Returns       []DailyReturn            `json:"returns"`
	Drawdowns     []DrawdownPeriod         `json:"drawdowns"`
	Trades        []TradeAnalysis          `json:"trades"`
	RiskMetrics   PortfolioRiskMetrics     `json:"risk_metrics"`
	Attribution   PerformanceAttribution   `json:"attribution"`
	GeneratedAt   time.Time                `json:"generated_at"`
}

// Period represents a time period for analysis
type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// DailyReturn represents portfolio return for a single day
type DailyReturn struct {
	Date   time.Time     `json:"date"`
	Return types.Decimal `json:"return"`
	Value  types.Decimal `json:"value"`
}

// DrawdownPeriod represents a drawdown period in the portfolio
type DrawdownPeriod struct {
	Start        time.Time     `json:"start"`
	End          *time.Time    `json:"end,omitempty"`
	PeakValue    types.Decimal `json:"peak_value"`
	TroughValue  types.Decimal `json:"trough_value"`
	Drawdown     types.Decimal `json:"drawdown"`
	DrawdownPct  types.Decimal `json:"drawdown_pct"`
	RecoveryDays *int          `json:"recovery_days,omitempty"`
}

// TradeAnalysis contains analysis of individual trades
type TradeAnalysis struct {
	Symbol       string        `json:"symbol"`
	Side         string        `json:"side"`
	EntryDate    time.Time     `json:"entry_date"`
	ExitDate     *time.Time    `json:"exit_date,omitempty"`
	EntryPrice   types.Decimal `json:"entry_price"`
	ExitPrice    *types.Decimal `json:"exit_price,omitempty"`
	Quantity     types.Decimal `json:"quantity"`
	PnL          types.Decimal `json:"pnl"`
	PnLPercent   types.Decimal `json:"pnl_percent"`
	HoldingDays  int           `json:"holding_days"`
	Fees         types.Decimal `json:"fees"`
}

// PortfolioRiskMetrics contains portfolio risk analysis
type PortfolioRiskMetrics struct {
	Beta               types.Decimal `json:"beta"`
	Alpha              types.Decimal `json:"alpha"`
	Sharpe             types.Decimal `json:"sharpe"`
	Sortino            types.Decimal `json:"sortino"`
	Volatility         types.Decimal `json:"volatility"`
	DownsideDeviation  types.Decimal `json:"downside_deviation"`
	MaxDrawdown        types.Decimal `json:"max_drawdown"`
	VaR95              types.Decimal `json:"var_95"`
	CVaR95             types.Decimal `json:"cvar_95"`
	ConcentrationRisk  types.Decimal `json:"concentration_risk"`
}

// PerformanceAttribution breaks down performance by various factors
type PerformanceAttribution struct {
	AssetClass    map[string]types.Decimal `json:"asset_class"`
	Sector        map[string]types.Decimal `json:"sector"`
	Geography     map[string]types.Decimal `json:"geography"`
	Individual    map[string]types.Decimal `json:"individual"`
}

// RiskCheck contains the result of portfolio risk validation
type RiskCheck struct {
	Passed          bool                    `json:"passed"`
	Violations      []RiskViolation         `json:"violations"`
	CurrentLimits   RiskLimits              `json:"current_limits"`
	PortfolioValue  types.Decimal           `json:"portfolio_value"`
	MaxDrawdown     types.Decimal           `json:"max_drawdown"`
	ConcentrationRisk types.Decimal         `json:"concentration_risk"`
	CheckedAt       time.Time               `json:"checked_at"`
}

// RiskViolation represents a specific risk limit violation
type RiskViolation struct {
	Type        RiskViolationType `json:"type"`
	Description string            `json:"description"`
	Current     types.Decimal     `json:"current"`
	Limit       types.Decimal     `json:"limit"`
	Severity    ViolationSeverity `json:"severity"`
}

// RiskViolationType defines different types of risk violations
type RiskViolationType string

const (
	RiskViolationMaxDrawdown        RiskViolationType = "MAX_DRAWDOWN"
	RiskViolationConcentration      RiskViolationType = "CONCENTRATION"
	RiskViolationPositionSize       RiskViolationType = "POSITION_SIZE"
	RiskViolationVaR                RiskViolationType = "VAR"
	RiskViolationLeverage           RiskViolationType = "LEVERAGE"
)

// ViolationSeverity defines the severity of a risk violation
type ViolationSeverity string

const (
	ViolationSeverityLow      ViolationSeverity = "LOW"
	ViolationSeverityMedium   ViolationSeverity = "MEDIUM"
	ViolationSeverityHigh     ViolationSeverity = "HIGH"
	ViolationSeverityCritical ViolationSeverity = "CRITICAL"
)

// RiskLimits defines the risk limits for a portfolio
type RiskLimits struct {
	MaxDrawdownPercent    types.Decimal `json:"max_drawdown_percent"`
	MaxPositionWeight     types.Decimal `json:"max_position_weight"`
	MaxConcentrationRisk  types.Decimal `json:"max_concentration_risk"`
	VaR95Limit           types.Decimal `json:"var_95_limit"`
	MaxLeverage          types.Decimal `json:"max_leverage"`
}

// PortfolioSnapshot represents a point-in-time snapshot of a portfolio
type PortfolioSnapshot struct {
	ID             string                   `json:"id"`
	PortfolioID    string                   `json:"portfolio_id"`
	Timestamp      time.Time                `json:"timestamp"`
	Metrics        *domain.PortfolioMetrics `json:"metrics"`
	Positions      []*domain.Position       `json:"positions"`
	CashBalance    types.Decimal            `json:"cash_balance"`
	TotalValue     types.Decimal            `json:"total_value"`
}

// OrderFill represents a fill from an executed order
type OrderFill struct {
	OrderID      string        `json:"order_id"`
	PortfolioID  string        `json:"portfolio_id"`
	Symbol       string        `json:"symbol"`
	Side         string        `json:"side"`
	Quantity     types.Decimal `json:"quantity"`
	Price        types.Decimal `json:"price"`
	Commission   types.Decimal `json:"commission"`
	FillTime     time.Time     `json:"fill_time"`
	ExecutionID  string        `json:"execution_id"`
}