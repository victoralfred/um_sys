package execution

import (
	"testing"
	"time"

	"github.com/trading-engine/pkg/types"
)

func TestExecutionType_String(t *testing.T) {
	tests := []struct {
		name     string
		execType ExecutionType
		expected string
	}{
		{
			name:     "Immediate execution",
			execType: ExecutionTypeImmediate,
			expected: "IMMEDIATE",
		},
		{
			name:     "TWAP execution",
			execType: ExecutionTypeTWAP,
			expected: "TWAP",
		},
		{
			name:     "VWAP execution",
			execType: ExecutionTypeVWAP,
			expected: "VWAP",
		},
		{
			name:     "POV execution",
			execType: ExecutionTypePOV,
			expected: "POV",
		},
		{
			name:     "Iceberg execution",
			execType: ExecutionTypeIceberg,
			expected: "ICEBERG",
		},
		{
			name:     "Sniper execution",
			execType: ExecutionTypeSniper,
			expected: "SNIPER",
		},
		{
			name:     "Maker execution",
			execType: ExecutionTypeMaker,
			expected: "MAKER",
		},
		{
			name:     "Unknown execution type",
			execType: ExecutionType(999),
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.execType.String()
			if result != tt.expected {
				t.Errorf("ExecutionType.String() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestExecutionStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   ExecutionStatus
		expected string
	}{
		{
			name:     "Pending status",
			status:   ExecutionStatusPending,
			expected: "PENDING",
		},
		{
			name:     "Accepted status",
			status:   ExecutionStatusAccepted,
			expected: "ACCEPTED",
		},
		{
			name:     "Rejected status",
			status:   ExecutionStatusRejected,
			expected: "REJECTED",
		},
		{
			name:     "Partially filled status",
			status:   ExecutionStatusPartiallyFilled,
			expected: "PARTIALLY_FILLED",
		},
		{
			name:     "Filled status",
			status:   ExecutionStatusFilled,
			expected: "FILLED",
		},
		{
			name:     "Cancelled status",
			status:   ExecutionStatusCancelled,
			expected: "CANCELLED",
		},
		{
			name:     "Expired status",
			status:   ExecutionStatusExpired,
			expected: "EXPIRED",
		},
		{
			name:     "Error status",
			status:   ExecutionStatusError,
			expected: "ERROR",
		},
		{
			name:     "Unknown status",
			status:   ExecutionStatus(999),
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("ExecutionStatus.String() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestExecutionRequest_Validation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		request ExecutionRequest
		valid   bool
	}{
		{
			name: "Valid immediate execution request",
			request: ExecutionRequest{
				MaxSlippageBps:  types.NewDecimalFromFloat(10.0),
				ExecutionType:   ExecutionTypeImmediate,
				PostOnly:        false,
				ReduceOnly:      false,
				ClientRequestID: "req-123",
				Timestamp:       now,
			},
			valid: true,
		},
		{
			name: "Valid TWAP execution request",
			request: ExecutionRequest{
				MaxSlippageBps:  types.NewDecimalFromFloat(5.0),
				ExecutionType:   ExecutionTypeTWAP,
				PostOnly:        false,
				ReduceOnly:      false,
				ClientRequestID: "req-124",
				Timestamp:       now,
			},
			valid: true,
		},
		{
			name: "Post-only execution request",
			request: ExecutionRequest{
				MaxSlippageBps:  types.NewDecimalFromFloat(0.0),
				ExecutionType:   ExecutionTypeMaker,
				PostOnly:        true,
				ReduceOnly:      false,
				ClientRequestID: "req-125",
				Timestamp:       now,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate client request ID
			if tt.request.ClientRequestID == "" && tt.valid {
				t.Error("Valid execution request should have ClientRequestID")
			}

			// Validate timestamp
			if tt.request.Timestamp.IsZero() && tt.valid {
				t.Error("Valid execution request should have Timestamp")
			}

			// Validate max slippage
			if tt.request.MaxSlippageBps.IsNegative() && tt.valid {
				t.Error("Valid execution request should not have negative max slippage")
			}

			// Validate post-only constraint
			if tt.request.PostOnly && tt.request.ExecutionType == ExecutionTypeImmediate {
				// This could be considered invalid in some contexts
				t.Log("Post-only immediate execution may not be valid in all contexts")
			}
		})
	}
}

func TestExecutionResponse_Validation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		response ExecutionResponse
		valid    bool
	}{
		{
			name: "Valid accepted response",
			response: ExecutionResponse{
				RequestID:     "req-123",
				OrderID:       "order-456",
				Status:        ExecutionStatusAccepted,
				Message:       "Order accepted for execution",
				Timestamp:     now,
				LatencyMicros: 1500,
			},
			valid: true,
		},
		{
			name: "Valid filled response with estimated fill",
			response: ExecutionResponse{
				RequestID: "req-124",
				OrderID:   "order-457",
				Status:    ExecutionStatusFilled,
				Message:   "Order completely filled",
				EstimatedFill: &EstimatedFill{
					EstimatedPrice:    types.NewDecimalFromFloat(100.25),
					EstimatedQuantity: types.NewDecimalFromFloat(100.0),
					EstimatedFee:      types.NewDecimalFromFloat(0.25),
					EstimatedSlippage: types.NewDecimalFromFloat(0.05),
					Confidence:        0.95,
				},
				Timestamp:     now,
				LatencyMicros: 850,
			},
			valid: true,
		},
		{
			name: "Valid rejected response",
			response: ExecutionResponse{
				RequestID:     "req-125",
				OrderID:       "order-458",
				Status:        ExecutionStatusRejected,
				Message:       "Order rejected - insufficient funds",
				Timestamp:     now,
				LatencyMicros: 500,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate required fields
			if tt.response.RequestID == "" && tt.valid {
				t.Error("Valid execution response should have RequestID")
			}

			if tt.response.OrderID == "" && tt.valid {
				t.Error("Valid execution response should have OrderID")
			}

			// Validate timestamp
			if tt.response.Timestamp.IsZero() && tt.valid {
				t.Error("Valid execution response should have Timestamp")
			}

			// Validate latency
			if tt.response.LatencyMicros < 0 && tt.valid {
				t.Error("Valid execution response should not have negative latency")
			}

			// Validate estimated fill if present
			if tt.response.EstimatedFill != nil {
				fill := tt.response.EstimatedFill

				if fill.EstimatedPrice.IsNegative() && tt.valid {
					t.Error("Valid estimated fill should not have negative price")
				}

				if fill.EstimatedQuantity.IsNegative() && tt.valid {
					t.Error("Valid estimated fill should not have negative quantity")
				}

				if fill.Confidence < 0.0 || fill.Confidence > 1.0 {
					t.Error("Valid estimated fill confidence should be between 0.0 and 1.0")
				}
			}
		})
	}
}

func TestOrderBook_Validation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		orderBook OrderBook
		valid     bool
	}{
		{
			name: "Valid order book",
			orderBook: OrderBook{
				Symbol:    "AAPL",
				Timestamp: now,
				Bids: []BookLevel{
					{
						Price:    types.NewDecimalFromFloat(150.25),
						Quantity: types.NewDecimalFromFloat(100.0),
						Orders:   5,
					},
					{
						Price:    types.NewDecimalFromFloat(150.20),
						Quantity: types.NewDecimalFromFloat(200.0),
						Orders:   8,
					},
				},
				Asks: []BookLevel{
					{
						Price:    types.NewDecimalFromFloat(150.30),
						Quantity: types.NewDecimalFromFloat(150.0),
						Orders:   6,
					},
					{
						Price:    types.NewDecimalFromFloat(150.35),
						Quantity: types.NewDecimalFromFloat(300.0),
						Orders:   12,
					},
				},
				LastTrade: &LastTrade{
					Price:     types.NewDecimalFromFloat(150.27),
					Quantity:  types.NewDecimalFromFloat(50.0),
					Timestamp: now.Add(-time.Second),
					Side:      "BUY",
				},
			},
			valid: true,
		},
		{
			name: "Invalid order book - crossed market",
			orderBook: OrderBook{
				Symbol:    "AAPL",
				Timestamp: now,
				Bids: []BookLevel{
					{
						Price:    types.NewDecimalFromFloat(150.35), // Higher than ask
						Quantity: types.NewDecimalFromFloat(100.0),
						Orders:   5,
					},
				},
				Asks: []BookLevel{
					{
						Price:    types.NewDecimalFromFloat(150.30), // Lower than bid
						Quantity: types.NewDecimalFromFloat(150.0),
						Orders:   6,
					},
				},
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate symbol
			if tt.orderBook.Symbol == "" && tt.valid {
				t.Error("Valid order book should have Symbol")
			}

			// Validate timestamp
			if tt.orderBook.Timestamp.IsZero() && tt.valid {
				t.Error("Valid order book should have Timestamp")
			}

			// Validate bid prices are descending
			for i := 1; i < len(tt.orderBook.Bids); i++ {
				if tt.orderBook.Bids[i-1].Price.Cmp(tt.orderBook.Bids[i].Price) < 0 && tt.valid {
					t.Error("Valid order book should have descending bid prices")
				}
			}

			// Validate ask prices are ascending
			for i := 1; i < len(tt.orderBook.Asks); i++ {
				if tt.orderBook.Asks[i-1].Price.Cmp(tt.orderBook.Asks[i].Price) > 0 && tt.valid {
					t.Error("Valid order book should have ascending ask prices")
				}
			}

			// Check for crossed market
			if len(tt.orderBook.Bids) > 0 && len(tt.orderBook.Asks) > 0 {
				bestBid := tt.orderBook.Bids[0].Price
				bestAsk := tt.orderBook.Asks[0].Price

				if bestBid.Cmp(bestAsk) >= 0 && tt.valid {
					t.Error("Valid order book should not have crossed market (bid >= ask)")
				}
			}
		})
	}
}

func TestEngineStatus_HealthCheck(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		status  EngineStatus
		healthy bool
	}{
		{
			name: "Healthy engine status",
			status: EngineStatus{
				IsRunning:       true,
				Uptime:          time.Hour * 2,
				ProcessedOrders: 1000,
				ActiveOrders:    25,
				AverageLatency:  time.Millisecond * 2,
				ErrorRate:       0.01,             // 1% error rate
				MemoryUsage:     50 * 1024 * 1024, // 50MB
				CPUUsage:        0.15,             // 15% CPU
				LastHealthCheck: now.Add(-time.Minute),
				ConnectedVenues: []string{"NYSE", "NASDAQ"},
			},
			healthy: true,
		},
		{
			name: "Unhealthy engine - high error rate",
			status: EngineStatus{
				IsRunning:       true,
				Uptime:          time.Hour,
				ProcessedOrders: 1000,
				ActiveOrders:    25,
				AverageLatency:  time.Millisecond * 2,
				ErrorRate:       0.15, // 15% error rate - too high
				MemoryUsage:     50 * 1024 * 1024,
				CPUUsage:        0.15,
				LastHealthCheck: now.Add(-time.Minute),
				ConnectedVenues: []string{"NYSE", "NASDAQ"},
			},
			healthy: false,
		},
		{
			name: "Unhealthy engine - high latency",
			status: EngineStatus{
				IsRunning:       true,
				Uptime:          time.Hour,
				ProcessedOrders: 1000,
				ActiveOrders:    25,
				AverageLatency:  time.Millisecond * 50, // 50ms - too high
				ErrorRate:       0.01,
				MemoryUsage:     50 * 1024 * 1024,
				CPUUsage:        0.15,
				LastHealthCheck: now.Add(-time.Minute),
				ConnectedVenues: []string{"NYSE", "NASDAQ"},
			},
			healthy: false,
		},
		{
			name: "Unhealthy engine - not running",
			status: EngineStatus{
				IsRunning:       false,
				Uptime:          time.Hour,
				ProcessedOrders: 1000,
				ActiveOrders:    0,
				AverageLatency:  time.Millisecond * 2,
				ErrorRate:       0.01,
				MemoryUsage:     50 * 1024 * 1024,
				CPUUsage:        0.15,
				LastHealthCheck: now.Add(-time.Minute),
				ConnectedVenues: []string{},
			},
			healthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic health checks
			isHealthy := tt.status.IsRunning &&
				tt.status.ErrorRate < 0.05 && // Less than 5% error rate
				tt.status.AverageLatency < time.Millisecond*10 && // Less than 10ms average latency
				len(tt.status.ConnectedVenues) > 0 && // At least one venue connected
				time.Since(tt.status.LastHealthCheck) < time.Minute*5 // Recent health check

			if isHealthy != tt.healthy {
				t.Errorf("Engine health check result = %v, expected %v", isHealthy, tt.healthy)
			}
		})
	}
}

func TestAlgoParams_Validation(t *testing.T) {
	now := time.Now()
	duration := time.Hour

	tests := []struct {
		name   string
		params AlgoParams
		valid  bool
	}{
		{
			name: "Valid TWAP parameters",
			params: AlgoParams{
				TWAPDuration:  &duration,
				TWAPSliceSize: &[]types.Decimal{types.NewDecimalFromFloat(100.0)}[0],
				MaxSlippage:   &[]types.Decimal{types.NewDecimalFromFloat(5.0)}[0],
				Urgency:       &[]float64{0.5}[0],
			},
			valid: true,
		},
		{
			name: "Valid VWAP parameters",
			params: AlgoParams{
				VWAPEndTime:       &[]time.Time{now.Add(time.Hour)}[0],
				VWAPParticipation: &[]float64{0.2}[0], // 20% participation rate
				MaxSlippage:       &[]types.Decimal{types.NewDecimalFromFloat(3.0)}[0],
			},
			valid: true,
		},
		{
			name: "Valid POV parameters",
			params: AlgoParams{
				POVRate:    &[]float64{0.15}[0], // 15% of volume
				POVMaxSize: &[]types.Decimal{types.NewDecimalFromFloat(1000.0)}[0],
				StartTime:  &now,
				EndTime:    &[]time.Time{now.Add(time.Hour)}[0],
			},
			valid: true,
		},
		{
			name: "Invalid parameters - negative urgency",
			params: AlgoParams{
				TWAPDuration: &duration,
				Urgency:      &[]float64{-0.5}[0], // Invalid - should be 0.0-1.0
			},
			valid: false,
		},
		{
			name: "Invalid parameters - end time before start time",
			params: AlgoParams{
				StartTime: &now,
				EndTime:   &[]time.Time{now.Add(-time.Hour)}[0], // Invalid - end before start
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate urgency range
			if tt.params.Urgency != nil {
				if (*tt.params.Urgency < 0.0 || *tt.params.Urgency > 1.0) && tt.valid {
					t.Error("Valid algo params should have urgency between 0.0 and 1.0")
				}
			}

			// Validate time range
			if tt.params.StartTime != nil && tt.params.EndTime != nil {
				if tt.params.EndTime.Before(*tt.params.StartTime) && tt.valid {
					t.Error("Valid algo params should have end time after start time")
				}
			}

			// Validate participation rates
			if tt.params.VWAPParticipation != nil {
				if (*tt.params.VWAPParticipation <= 0.0 || *tt.params.VWAPParticipation > 1.0) && tt.valid {
					t.Error("Valid VWAP participation should be between 0.0 and 1.0")
				}
			}

			if tt.params.POVRate != nil {
				if (*tt.params.POVRate <= 0.0 || *tt.params.POVRate > 1.0) && tt.valid {
					t.Error("Valid POV rate should be between 0.0 and 1.0")
				}
			}

			// Validate sizes are positive
			if tt.params.TWAPSliceSize != nil {
				if tt.params.TWAPSliceSize.IsNegative() && tt.valid {
					t.Error("Valid TWAP slice size should be positive")
				}
			}

			if tt.params.POVMaxSize != nil {
				if tt.params.POVMaxSize.IsNegative() && tt.valid {
					t.Error("Valid POV max size should be positive")
				}
			}
		})
	}
}
