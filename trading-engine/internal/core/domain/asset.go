package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/trading-engine/pkg/types"
)

// AssetType represents the type of financial instrument
type AssetType int

const (
	AssetTypeUnknown AssetType = iota
	AssetTypeStock
	AssetTypeCrypto
	AssetTypeForex
	AssetTypeFuture
	AssetTypeOption
	AssetTypeBond
	AssetTypeETF
	AssetTypeCommodity
)

func (at AssetType) String() string {
	switch at {
	case AssetTypeStock:
		return "STOCK"
	case AssetTypeCrypto:
		return "CRYPTO"
	case AssetTypeForex:
		return "FOREX"
	case AssetTypeFuture:
		return "FUTURE"
	case AssetTypeOption:
		return "OPTION"
	case AssetTypeBond:
		return "BOND"
	case AssetTypeETF:
		return "ETF"
	case AssetTypeCommodity:
		return "COMMODITY"
	default:
		return "UNKNOWN"
	}
}

// Asset represents a tradeable financial instrument
type Asset struct {
	Symbol      string        `json:"symbol"`
	Name        string        `json:"name"`
	AssetType   AssetType     `json:"asset_type"`
	Exchange    string        `json:"exchange"`
	Currency    string        `json:"currency"`
	Precision   int           `json:"precision"`
	MinQuantity types.Decimal `json:"min_quantity"`
	MaxQuantity types.Decimal `json:"max_quantity"`
	TickSize    types.Decimal `json:"tick_size"`
	IsActive    bool          `json:"is_active"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// AssetBuilder provides a fluent interface for creating assets
type AssetBuilder struct {
	asset Asset
	err   error
}

// NewAssetBuilder creates a new asset builder
func NewAssetBuilder() *AssetBuilder {
	return &AssetBuilder{
		asset: Asset{
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Precision:   8, // Default precision
			MinQuantity: types.Zero(),
			MaxQuantity: types.Zero(),
			TickSize:    types.Zero(),
		},
	}
}

// Symbol sets the asset symbol with validation
func (b *AssetBuilder) Symbol(symbol string) *AssetBuilder {
	if b.err != nil {
		return b
	}

	if err := validateSymbol(symbol); err != nil {
		b.err = err
		return b
	}

	b.asset.Symbol = strings.ToUpper(symbol)
	return b
}

// Name sets the asset name
func (b *AssetBuilder) Name(name string) *AssetBuilder {
	if b.err != nil {
		return b
	}

	if strings.TrimSpace(name) == "" {
		b.err = fmt.Errorf("asset name cannot be empty")
		return b
	}

	b.asset.Name = strings.TrimSpace(name)
	return b
}

// Type sets the asset type
func (b *AssetBuilder) Type(assetType AssetType) *AssetBuilder {
	if b.err != nil {
		return b
	}

	if assetType == AssetTypeUnknown {
		b.err = fmt.Errorf("asset type cannot be unknown")
		return b
	}

	b.asset.AssetType = assetType
	// Set default precision based on asset type
	b.asset.Precision = getDefaultPrecision(assetType)
	return b
}

// Exchange sets the exchange
func (b *AssetBuilder) Exchange(exchange string) *AssetBuilder {
	if b.err != nil {
		return b
	}

	b.asset.Exchange = strings.ToUpper(strings.TrimSpace(exchange))
	return b
}

// Currency sets the base currency
func (b *AssetBuilder) Currency(currency string) *AssetBuilder {
	if b.err != nil {
		return b
	}

	if err := validateCurrency(currency); err != nil {
		b.err = err
		return b
	}

	b.asset.Currency = strings.ToUpper(currency)
	return b
}

// Precision sets the decimal precision
func (b *AssetBuilder) Precision(precision int) *AssetBuilder {
	if b.err != nil {
		return b
	}

	if precision < 0 || precision > 18 {
		b.err = fmt.Errorf("precision must be between 0 and 18, got %d", precision)
		return b
	}

	b.asset.Precision = precision
	return b
}

// MinQuantity sets the minimum tradeable quantity
func (b *AssetBuilder) MinQuantity(qty types.Decimal) *AssetBuilder {
	if b.err != nil {
		return b
	}

	if qty.IsNegative() {
		b.err = fmt.Errorf("minimum quantity cannot be negative")
		return b
	}

	b.asset.MinQuantity = qty
	return b
}

// MaxQuantity sets the maximum tradeable quantity
func (b *AssetBuilder) MaxQuantity(qty types.Decimal) *AssetBuilder {
	if b.err != nil {
		return b
	}

	if qty.IsNegative() || qty.IsZero() {
		b.err = fmt.Errorf("maximum quantity must be positive")
		return b
	}

	b.asset.MaxQuantity = qty
	return b
}

// TickSize sets the minimum price movement
func (b *AssetBuilder) TickSize(size types.Decimal) *AssetBuilder {
	if b.err != nil {
		return b
	}

	if !size.IsPositive() {
		b.err = fmt.Errorf("tick size must be positive")
		return b
	}

	b.asset.TickSize = size
	return b
}

// Build creates the asset
func (b *AssetBuilder) Build() (*Asset, error) {
	if b.err != nil {
		return nil, b.err
	}

	// Validate required fields
	if b.asset.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if b.asset.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if b.asset.AssetType == AssetTypeUnknown {
		return nil, fmt.Errorf("asset type is required")
	}

	// Validate quantity ranges
	if !b.asset.MinQuantity.IsZero() && !b.asset.MaxQuantity.IsZero() {
		if b.asset.MinQuantity.Cmp(b.asset.MaxQuantity) > 0 {
			return nil, fmt.Errorf("minimum quantity cannot be greater than maximum quantity")
		}
	}

	return &b.asset, nil
}

// validateSymbol validates asset symbol format
func validateSymbol(symbol string) error {
	if strings.TrimSpace(symbol) == "" {
		return fmt.Errorf("symbol cannot be empty")
	}

	// Basic symbol validation - alphanumeric and common separators
	symbolPattern := regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	if !symbolPattern.MatchString(symbol) {
		return fmt.Errorf("symbol contains invalid characters: %s", symbol)
	}

	if len(symbol) > 20 {
		return fmt.Errorf("symbol too long (max 20 characters): %s", symbol)
	}

	return nil
}

// validateCurrency validates currency code
func validateCurrency(currency string) error {
	currency = strings.TrimSpace(currency)
	if currency == "" {
		return fmt.Errorf("currency cannot be empty")
	}

	// Basic currency validation - 3 letter codes
	if len(currency) != 3 {
		return fmt.Errorf("currency must be 3 characters: %s", currency)
	}

	currencyPattern := regexp.MustCompile(`^[A-Za-z]{3}$`)
	if !currencyPattern.MatchString(currency) {
		return fmt.Errorf("currency must be alphabetic: %s", currency)
	}

	return nil
}

// getDefaultPrecision returns default precision for asset type
func getDefaultPrecision(assetType AssetType) int {
	switch assetType {
	case AssetTypeStock:
		return 2
	case AssetTypeCrypto:
		return 8
	case AssetTypeForex:
		return 5
	case AssetTypeFuture:
		return 2
	case AssetTypeOption:
		return 2
	case AssetTypeBond:
		return 4
	case AssetTypeETF:
		return 2
	case AssetTypeCommodity:
		return 3
	default:
		return 8
	}
}

// ID returns a unique identifier for the asset
func (a *Asset) ID() string {
	if a.Exchange != "" {
		return fmt.Sprintf("%s:%s", a.Exchange, a.Symbol)
	}
	return a.Symbol
}

// Validate performs comprehensive validation of the asset
func (a *Asset) Validate() error {
	if err := validateSymbol(a.Symbol); err != nil {
		return fmt.Errorf("invalid symbol: %w", err)
	}

	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if a.AssetType == AssetTypeUnknown {
		return fmt.Errorf("asset type is required")
	}

	if a.Currency != "" {
		if err := validateCurrency(a.Currency); err != nil {
			return fmt.Errorf("invalid currency: %w", err)
		}
	}

	if a.Precision < 0 || a.Precision > 18 {
		return fmt.Errorf("precision must be between 0 and 18")
	}

	if !a.MinQuantity.IsZero() && a.MinQuantity.IsNegative() {
		return fmt.Errorf("minimum quantity cannot be negative")
	}

	if !a.MaxQuantity.IsZero() && !a.MaxQuantity.IsPositive() {
		return fmt.Errorf("maximum quantity must be positive")
	}

	if !a.MinQuantity.IsZero() && !a.MaxQuantity.IsZero() {
		if a.MinQuantity.Cmp(a.MaxQuantity) > 0 {
			return fmt.Errorf("minimum quantity cannot be greater than maximum quantity")
		}
	}

	if !a.TickSize.IsZero() && !a.TickSize.IsPositive() {
		return fmt.Errorf("tick size must be positive")
	}

	return nil
}

// IsValidQuantity checks if a quantity is valid for this asset
func (a *Asset) IsValidQuantity(quantity types.Decimal) bool {
	if quantity.IsNegative() {
		return false
	}

	if !a.MinQuantity.IsZero() && quantity.Cmp(a.MinQuantity) < 0 {
		return false
	}

	if !a.MaxQuantity.IsZero() && quantity.Cmp(a.MaxQuantity) > 0 {
		return false
	}

	return true
}

// RoundPrice rounds a price to the asset's tick size
func (a *Asset) RoundPrice(price types.Decimal) types.Decimal {
	if a.TickSize.IsZero() {
		return price
	}

	// Round to nearest tick
	ticks := price.Div(a.TickSize)
	roundedTicks := types.NewDecimalFromInt(int64(ticks.Float64() + 0.5))
	return roundedTicks.Mul(a.TickSize)
}
