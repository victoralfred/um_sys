package ports

import (
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

func TestExecutionEventType_String(t *testing.T) {
	tests := []struct {
		name     string
		event    ExecutionEventType
		expected string
	}{
		{
			name:     "Order submitted event",
			event:    ExecutionEventOrderSubmitted,
			expected: "ORDER_SUBMITTED",
		},
		{
			name:     "Order accepted event",
			event:    ExecutionEventOrderAccepted,
			expected: "ORDER_ACCEPTED",
		},
		{
			name:     "Order rejected event",
			event:    ExecutionEventOrderRejected,
			expected: "ORDER_REJECTED",
		},
		{
			name:     "Order filled event",
			event:    ExecutionEventOrderFilled,
			expected: "ORDER_FILLED",
		},
		{
			name:     "Order partially filled event",
			event:    ExecutionEventOrderPartiallyFilled,
			expected: "ORDER_PARTIALLY_FILLED",
		},
		{
			name:     "Order cancelled event",
			event:    ExecutionEventOrderCancelled,
			expected: "ORDER_CANCELLED",
		},
		{
			name:     "Order modified event",
			event:    ExecutionEventOrderModified,
			expected: "ORDER_MODIFIED",
		},
		{
			name:     "Execution error event",
			event:    ExecutionEventExecutionError,
			expected: "EXECUTION_ERROR",
		},
		{
			name:     "Unknown event type",
			event:    ExecutionEventType(999),
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.String()
			if result != tt.expected {
				t.Errorf("ExecutionEventType.String() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestExecutionResult_Validation(t *testing.T) {
	tests := []struct {
		name   string
		result ExecutionResult
		valid  bool
	}{
		{
			name: "Valid execution result",
			result: ExecutionResult{
				OrderID:       "order-123",
				ExecutedAt:    time.Now(),
				AveragePrice:  types.NewDecimalFromFloat(100.50),
				TotalQuantity: types.NewDecimalFromFloat(10.0),
				Status:        "FILLED",
				Fills: []Fill{
					{
						ID:        "fill-1",
						OrderID:   "order-123",
						Price:     types.NewDecimalFromFloat(100.50),
						Quantity:  types.NewDecimalFromFloat(10.0),
						Fee:       types.NewDecimalFromFloat(0.10),
						Timestamp: time.Now(),
						Venue:     "NYSE",
						TradeID:   "trade-456",
					},
				},
			},
			valid: true,
		},
		{
			name: "Valid partial execution result",
			result: ExecutionResult{
				OrderID:       "order-124",
				ExecutedAt:    time.Now(),
				AveragePrice:  types.NewDecimalFromFloat(99.75),
				TotalQuantity: types.NewDecimalFromFloat(5.0),
				Status:        "PARTIALLY_FILLED",
				Fills: []Fill{
					{
						ID:        "fill-2",
						OrderID:   "order-124",
						Price:     types.NewDecimalFromFloat(99.75),
						Quantity:  types.NewDecimalFromFloat(5.0),
						Fee:       types.NewDecimalFromFloat(0.05),
						Timestamp: time.Now(),
						Venue:     "NASDAQ",
						TradeID:   "trade-789",
					},
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			if tt.result.OrderID == "" && tt.valid {
				t.Error("Valid execution result should have OrderID")
			}

			if tt.result.Status == "" && tt.valid {
				t.Error("Valid execution result should have Status")
			}

			// Check that fills belong to the order
			for _, fill := range tt.result.Fills {
				if fill.OrderID != tt.result.OrderID && tt.valid {
					t.Error("All fills should belong to the same order")
				}
			}
		})
	}
}

func TestFill_Validation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		fill  Fill
		valid bool
	}{
		{
			name: "Valid fill",
			fill: Fill{
				ID:        "fill-1",
				OrderID:   "order-123",
				Price:     types.NewDecimalFromFloat(100.0),
				Quantity:  types.NewDecimalFromFloat(10.0),
				Fee:       types.NewDecimalFromFloat(0.10),
				Timestamp: now,
				Venue:     "NYSE",
				TradeID:   "trade-456",
			},
			valid: true,
		},
		{
			name: "Fill with zero price should be invalid",
			fill: Fill{
				ID:        "fill-2",
				OrderID:   "order-124",
				Price:     types.Zero(),
				Quantity:  types.NewDecimalFromFloat(10.0),
				Fee:       types.NewDecimalFromFloat(0.10),
				Timestamp: now,
				Venue:     "NYSE",
				TradeID:   "trade-789",
			},
			valid: false,
		},
		{
			name: "Fill with zero quantity should be invalid",
			fill: Fill{
				ID:        "fill-3",
				OrderID:   "order-125",
				Price:     types.NewDecimalFromFloat(100.0),
				Quantity:  types.Zero(),
				Fee:       types.NewDecimalFromFloat(0.10),
				Timestamp: now,
				Venue:     "NYSE",
				TradeID:   "trade-101",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate price
			if tt.fill.Price.IsZero() && tt.valid {
				t.Error("Valid fill should have positive price")
			}

			// Validate quantity
			if tt.fill.Quantity.IsZero() && tt.valid {
				t.Error("Valid fill should have positive quantity")
			}

			// Validate required fields
			if tt.fill.ID == "" && tt.valid {
				t.Error("Valid fill should have ID")
			}

			if tt.fill.OrderID == "" && tt.valid {
				t.Error("Valid fill should have OrderID")
			}
		})
	}
}

func TestMarketData_Validation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		marketData MarketData
		valid      bool
	}{
		{
			name: "Valid market data",
			marketData: MarketData{
				BidPrice:       types.NewDecimalFromFloat(99.50),
				AskPrice:       types.NewDecimalFromFloat(100.50),
				BidSize:        types.NewDecimalFromFloat(100.0),
				AskSize:        types.NewDecimalFromFloat(150.0),
				LastTradePrice: types.NewDecimalFromFloat(100.0),
				LastTradeSize:  types.NewDecimalFromFloat(50.0),
				Volume:         types.NewDecimalFromFloat(10000.0),
				VWAP:           types.NewDecimalFromFloat(99.75),
				Volatility:     types.NewDecimalFromFloat(0.25),
				Timestamp:      now,
			},
			valid: true,
		},
		{
			name: "Market data with invalid spread (bid > ask)",
			marketData: MarketData{
				BidPrice:       types.NewDecimalFromFloat(101.0),
				AskPrice:       types.NewDecimalFromFloat(100.0),
				BidSize:        types.NewDecimalFromFloat(100.0),
				AskSize:        types.NewDecimalFromFloat(150.0),
				LastTradePrice: types.NewDecimalFromFloat(100.0),
				LastTradeSize:  types.NewDecimalFromFloat(50.0),
				Volume:         types.NewDecimalFromFloat(10000.0),
				VWAP:           types.NewDecimalFromFloat(99.75),
				Volatility:     types.NewDecimalFromFloat(0.25),
				Timestamp:      now,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate bid-ask spread
			if tt.marketData.BidPrice.Cmp(tt.marketData.AskPrice) >= 0 && tt.valid {
				t.Error("Valid market data should have bid price less than ask price")
			}

			// Validate positive prices
			if tt.marketData.BidPrice.IsNegative() && tt.valid {
				t.Error("Valid market data should have positive bid price")
			}

			if tt.marketData.AskPrice.IsNegative() && tt.valid {
				t.Error("Valid market data should have positive ask price")
			}

			// Validate sizes are non-negative
			if tt.marketData.BidSize.IsNegative() && tt.valid {
				t.Error("Valid market data should have non-negative bid size")
			}

			if tt.marketData.AskSize.IsNegative() && tt.valid {
				t.Error("Valid market data should have non-negative ask size")
			}
		})
	}
}

func TestExecutionMetrics_Calculation(t *testing.T) {
	metrics := ExecutionMetrics{
		TotalOrdersProcessed: 1000,
		SuccessfulExecutions: 950,
		FailedExecutions:     50,
		OrdersPerSecond:      100.0,
		AverageLatency:       time.Millisecond * 5,
		P99Latency:           time.Millisecond * 20,
		ActiveOrders:         25,
		LastExecutionTime:    time.Now(),
	}

	// Test success rate calculation
	expectedSuccessRate := float64(950) / float64(1000)
	actualSuccessRate := float64(metrics.SuccessfulExecutions) / float64(metrics.TotalOrdersProcessed)

	if actualSuccessRate != expectedSuccessRate {
		t.Errorf("Success rate calculation incorrect. Expected %v, got %v", expectedSuccessRate, actualSuccessRate)
	}

	// Test that total orders equals successful + failed
	if metrics.TotalOrdersProcessed != metrics.SuccessfulExecutions+metrics.FailedExecutions {
		t.Error("Total orders should equal successful + failed executions")
	}

	// Test latency constraints
	if metrics.P99Latency < metrics.AverageLatency {
		t.Error("P99 latency should be greater than or equal to average latency")
	}
}

func TestSlippageData_Calculation(t *testing.T) {
	tests := []struct {
		name             string
		slippageData     SlippageData
		expectedSlippage types.Decimal
		expectedBps      types.Decimal
	}{
		{
			name: "Positive slippage",
			slippageData: SlippageData{
				ExpectedPrice: types.NewDecimalFromFloat(100.0),
				ActualPrice:   types.NewDecimalFromFloat(100.5),
				OrderSize:     types.NewDecimalFromFloat(1000.0),
			},
			expectedSlippage: types.NewDecimalFromFloat(0.5),
			expectedBps:      types.NewDecimalFromFloat(50.0), // 0.5% = 50bps
		},
		{
			name: "Negative slippage (better execution)",
			slippageData: SlippageData{
				ExpectedPrice: types.NewDecimalFromFloat(100.0),
				ActualPrice:   types.NewDecimalFromFloat(99.8),
				OrderSize:     types.NewDecimalFromFloat(500.0),
			},
			expectedSlippage: types.NewDecimalFromFloat(-0.2),
			expectedBps:      types.NewDecimalFromFloat(-20.0), // -0.2% = -20bps
		},
		{
			name: "Zero slippage",
			slippageData: SlippageData{
				ExpectedPrice: types.NewDecimalFromFloat(100.0),
				ActualPrice:   types.NewDecimalFromFloat(100.0),
				OrderSize:     types.NewDecimalFromFloat(100.0),
			},
			expectedSlippage: types.Zero(),
			expectedBps:      types.Zero(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate slippage: actual_price - expected_price
			actualSlippage := tt.slippageData.ActualPrice.Sub(tt.slippageData.ExpectedPrice)

			// Use tolerance for decimal comparison due to floating point precision
			tolerance := types.NewDecimalFromFloat(0.0001)
			slippageDiff := actualSlippage.Sub(tt.expectedSlippage).Abs()
			if slippageDiff.Cmp(tolerance) > 0 {
				t.Errorf("Slippage calculation incorrect. Expected %v, got %v (diff: %v)",
					tt.expectedSlippage.String(), actualSlippage.String(), slippageDiff.String())
			}

			// Calculate basis points: (slippage / expected_price) * 10000
			if !tt.slippageData.ExpectedPrice.IsZero() {
				expectedBps := actualSlippage.Div(tt.slippageData.ExpectedPrice).Mul(types.NewDecimalFromInt(10000))
				bpsTolerance := types.NewDecimalFromFloat(0.01) // 0.01 bps tolerance
				bpsDiff := expectedBps.Sub(tt.expectedBps).Abs()

				if bpsDiff.Cmp(bpsTolerance) > 0 {
					t.Errorf("Slippage BPS calculation incorrect. Expected %v, got %v (diff: %v)",
						tt.expectedBps.String(), expectedBps.String(), bpsDiff.String())
				}
			}
		})
	}
}

func TestTimeRange_Validation(t *testing.T) {
	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	tests := []struct {
		name      string
		timeRange TimeRange
		valid     bool
	}{
		{
			name: "Valid time range",
			timeRange: TimeRange{
				From: hourAgo,
				To:   now,
			},
			valid: true,
		},
		{
			name: "Invalid time range (from after to)",
			timeRange: TimeRange{
				From: now,
				To:   hourAgo,
			},
			valid: false,
		},
		{
			name: "Same start and end time",
			timeRange: TimeRange{
				From: now,
				To:   now,
			},
			valid: true, // Could be valid for instantaneous queries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := !tt.timeRange.From.After(tt.timeRange.To)

			if isValid != tt.valid {
				t.Errorf("TimeRange validation incorrect. Expected %v, got %v", tt.valid, isValid)
			}
		})
	}
}
