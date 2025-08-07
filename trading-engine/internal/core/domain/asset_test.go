package domain

import (
	"testing"

	"github.com/trading-engine/pkg/types"
)

func TestAssetBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder func() *AssetBuilder
		wantErr bool
	}{
		{
			name: "valid stock asset",
			builder: func() *AssetBuilder {
				return NewAssetBuilder().
					Symbol("AAPL").
					Name("Apple Inc.").
					Type(AssetTypeStock).
					Exchange("NASDAQ").
					Currency("USD")
			},
			wantErr: false,
		},
		{
			name: "valid crypto asset",
			builder: func() *AssetBuilder {
				return NewAssetBuilder().
					Symbol("BTCUSD").
					Name("Bitcoin").
					Type(AssetTypeCrypto).
					Exchange("BINANCE").
					Currency("USD")
			},
			wantErr: false,
		},
		{
			name: "empty symbol",
			builder: func() *AssetBuilder {
				return NewAssetBuilder().
					Symbol("").
					Name("Test Asset").
					Type(AssetTypeStock)
			},
			wantErr: true,
		},
		{
			name: "empty name",
			builder: func() *AssetBuilder {
				return NewAssetBuilder().
					Symbol("TEST").
					Name("").
					Type(AssetTypeStock)
			},
			wantErr: true,
		},
		{
			name: "unknown asset type",
			builder: func() *AssetBuilder {
				return NewAssetBuilder().
					Symbol("TEST").
					Name("Test Asset").
					Type(AssetTypeUnknown)
			},
			wantErr: true,
		},
		{
			name: "invalid currency",
			builder: func() *AssetBuilder {
				return NewAssetBuilder().
					Symbol("TEST").
					Name("Test Asset").
					Type(AssetTypeStock).
					Currency("INVALID")
			},
			wantErr: true,
		},
		{
			name: "negative min quantity",
			builder: func() *AssetBuilder {
				negativeQty, _ := types.NewDecimal("-1")
				return NewAssetBuilder().
					Symbol("TEST").
					Name("Test Asset").
					Type(AssetTypeStock).
					MinQuantity(negativeQty)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := tt.builder().Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("AssetBuilder.Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && asset == nil {
				t.Errorf("AssetBuilder.Build() returned nil asset")
			}
		})
	}
}

func TestValidateSymbol(t *testing.T) {
	tests := []struct {
		name    string
		symbol  string
		wantErr bool
	}{
		{"valid simple symbol", "AAPL", false},
		{"valid crypto pair", "BTCUSD", false},
		{"valid with separator", "BTC-USD", false},
		{"valid with dot", "BRK.A", false},
		{"valid with slash", "EUR/USD", false},
		{"empty symbol", "", true},
		{"whitespace only", "   ", true},
		{"invalid characters", "BTC@USD", true},
		{"too long symbol", "VERYLONGSYMBOLNAME123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSymbol(tt.symbol)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSymbol() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		wantErr  bool
	}{
		{"valid USD", "USD", false},
		{"valid EUR", "EUR", false},
		{"valid lowercase", "usd", false},
		{"empty currency", "", true},
		{"too short", "US", true},
		{"too long", "USDT", true},
		{"with numbers", "US1", true},
		{"with symbols", "US$", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCurrency(tt.currency)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCurrency() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDefaultPrecision(t *testing.T) {
	tests := []struct {
		name      string
		assetType AssetType
		want      int
	}{
		{"stock precision", AssetTypeStock, 2},
		{"crypto precision", AssetTypeCrypto, 8},
		{"forex precision", AssetTypeForex, 5},
		{"unknown precision", AssetTypeUnknown, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDefaultPrecision(tt.assetType)
			if got != tt.want {
				t.Errorf("getDefaultPrecision() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAssetID(t *testing.T) {
	tests := []struct {
		name     string
		asset    Asset
		expected string
	}{
		{
			name: "with exchange",
			asset: Asset{
				Symbol:   "AAPL",
				Exchange: "NASDAQ",
			},
			expected: "NASDAQ:AAPL",
		},
		{
			name: "without exchange",
			asset: Asset{
				Symbol: "BTCUSD",
			},
			expected: "BTCUSD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.asset.ID()
			if got != tt.expected {
				t.Errorf("Asset.ID() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAssetValidate(t *testing.T) {
	validAsset := Asset{
		Symbol:    "AAPL",
		Name:      "Apple Inc.",
		AssetType: AssetTypeStock,
		Currency:  "USD",
		Precision: 2,
	}

	tests := []struct {
		name    string
		asset   Asset
		wantErr bool
	}{
		{"valid asset", validAsset, false},
		{
			"invalid symbol",
			Asset{
				Symbol:    "",
				Name:      "Test",
				AssetType: AssetTypeStock,
			},
			true,
		},
		{
			"invalid name",
			Asset{
				Symbol:    "TEST",
				Name:      "",
				AssetType: AssetTypeStock,
			},
			true,
		},
		{
			"invalid asset type",
			Asset{
				Symbol:    "TEST",
				Name:      "Test",
				AssetType: AssetTypeUnknown,
			},
			true,
		},
		{
			"invalid precision",
			Asset{
				Symbol:    "TEST",
				Name:      "Test",
				AssetType: AssetTypeStock,
				Precision: -1,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.asset.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Asset.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssetIsValidQuantity(t *testing.T) {
	minQty, _ := types.NewDecimal("1")
	maxQty, _ := types.NewDecimal("1000")
	
	asset := Asset{
		MinQuantity: minQty,
		MaxQuantity: maxQty,
	}

	tests := []struct {
		name     string
		quantity string
		want     bool
	}{
		{"valid quantity", "10", true},
		{"minimum quantity", "1", true},
		{"maximum quantity", "1000", true},
		{"below minimum", "0.5", false},
		{"above maximum", "1001", false},
		{"negative quantity", "-1", false},
		{"zero quantity", "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qty, _ := types.NewDecimal(tt.quantity)
			got := asset.IsValidQuantity(qty)
			if got != tt.want {
				t.Errorf("Asset.IsValidQuantity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAssetRoundPrice(t *testing.T) {
	tickSize, _ := types.NewDecimal("0.01")
	asset := Asset{TickSize: tickSize}

	tests := []struct {
		name  string
		price string
		want  string
	}{
		{"exact tick", "10.01", "10.01"},
		{"round down", "10.014", "10.01"},
		{"round up", "10.016", "10.02"},
		{"no tick size", "10.123456", "10.123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, _ := types.NewDecimal(tt.price)
			testAsset := asset
			if tt.name == "no tick size" {
				testAsset.TickSize = types.Zero()
			}
			
			got := testAsset.RoundPrice(price)
			if got.String() != tt.want {
				t.Errorf("Asset.RoundPrice() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

func BenchmarkAssetValidation(b *testing.B) {
	asset := Asset{
		Symbol:    "AAPL",
		Name:      "Apple Inc.",
		AssetType: AssetTypeStock,
		Currency:  "USD",
		Precision: 2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = asset.Validate()
	}
}

func BenchmarkAssetBuilder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewAssetBuilder().
			Symbol("AAPL").
			Name("Apple Inc.").
			Type(AssetTypeStock).
			Exchange("NASDAQ").
			Currency("USD").
			Build()
	}
}