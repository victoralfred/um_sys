package portfolio

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
)


// ExecutionIntegration provides integration between execution engine and portfolio management
type ExecutionIntegration struct {
	portfolioManager ports.PortfolioManager
	executionService ExecutionService
	
	// Order tracking
	orderToPortfolio map[string]string // orderID -> portfolioID
	orderMutex       sync.RWMutex
	
	// Event channels
	executionChan    chan *ports.ExecutionResult
	fillChan         chan *ports.OrderFill
	stopChan         chan struct{}
	
	// Configuration
	config IntegrationConfig
}

// IntegrationConfig contains configuration for the execution integration
type IntegrationConfig struct {
	BufferSize           int  `json:"buffer_size"`
	EnableAsyncProcessing bool `json:"enable_async_processing"`
	LogExecutions        bool `json:"log_executions"`
	ValidateOrders       bool `json:"validate_orders"`
}

// DefaultIntegrationConfig returns default configuration
func DefaultIntegrationConfig() IntegrationConfig {
	return IntegrationConfig{
		BufferSize:           1000,
		EnableAsyncProcessing: true,
		LogExecutions:        true,
		ValidateOrders:       true,
	}
}

// NewExecutionIntegration creates a new execution integration
func NewExecutionIntegration(
	portfolioManager ports.PortfolioManager,
	executionService ExecutionService,
	config IntegrationConfig,
) *ExecutionIntegration {
	return &ExecutionIntegration{
		portfolioManager:  portfolioManager,
		executionService:  executionService,
		orderToPortfolio:  make(map[string]string),
		executionChan:     make(chan *ports.ExecutionResult, config.BufferSize),
		fillChan:          make(chan *ports.OrderFill, config.BufferSize),
		stopChan:          make(chan struct{}),
		config:           config,
	}
}

// Start begins the integration service
func (ei *ExecutionIntegration) Start(ctx context.Context) error {
	if ei.config.EnableAsyncProcessing {
		go ei.processExecutions(ctx)
		go ei.processFills(ctx)
	}
	
	if ei.config.LogExecutions {
		log.Println("Execution integration started")
	}
	
	return nil
}

// Stop shuts down the integration service
func (ei *ExecutionIntegration) Stop() error {
	close(ei.stopChan)
	
	if ei.config.LogExecutions {
		log.Println("Execution integration stopped")
	}
	
	return nil
}

// SubmitOrderWithPortfolio submits an order and tracks it for a specific portfolio
func (ei *ExecutionIntegration) SubmitOrderWithPortfolio(ctx context.Context, portfolioID string, order *domain.Order) (*ports.ExecutionResult, error) {
	// Validate order against portfolio if enabled
	if ei.config.ValidateOrders {
		if err := ei.portfolioManager.ValidateOrder(ctx, portfolioID, order); err != nil {
			return nil, fmt.Errorf("portfolio validation failed: %w", err)
		}
	}
	
	// Track order to portfolio mapping
	ei.orderMutex.Lock()
	ei.orderToPortfolio[order.ID] = portfolioID
	ei.orderMutex.Unlock()
	
	// Submit order to execution service
	result, err := ei.executionService.SubmitOrder(ctx, order)
	if err != nil {
		// Remove tracking on failure
		ei.orderMutex.Lock()
		delete(ei.orderToPortfolio, order.ID)
		ei.orderMutex.Unlock()
		return nil, err
	}
	
	// Set portfolio ID in result for downstream processing
	if result != nil {
		result.PortfolioID = portfolioID
	}
	
	// Process execution synchronously or asynchronously
	if ei.config.EnableAsyncProcessing {
		select {
		case ei.executionChan <- result:
		default:
			// Channel full, process synchronously
			if err := ei.portfolioManager.OnOrderExecuted(ctx, result); err != nil {
				if ei.config.LogExecutions {
					log.Printf("Failed to process execution async: %v", err)
				}
			}
		}
	} else {
		if err := ei.portfolioManager.OnOrderExecuted(ctx, result); err != nil {
			if ei.config.LogExecutions {
				log.Printf("Failed to process execution sync: %v", err)
			}
		}
	}
	
	return result, nil
}

// OnOrderFilled handles order fill notifications
func (ei *ExecutionIntegration) OnOrderFilled(ctx context.Context, orderID string, fill *ports.OrderFill) error {
	// Get portfolio ID for this order
	ei.orderMutex.RLock()
	portfolioID, exists := ei.orderToPortfolio[orderID]
	ei.orderMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("portfolio not found for order %s", orderID)
	}
	
	// Set portfolio ID in fill
	fill.PortfolioID = portfolioID
	fill.OrderID = orderID
	
	// Process fill synchronously or asynchronously
	if ei.config.EnableAsyncProcessing {
		select {
		case ei.fillChan <- fill:
		default:
			// Channel full, process synchronously
			if err := ei.portfolioManager.OnOrderFilled(ctx, fill); err != nil {
				if ei.config.LogExecutions {
					log.Printf("Failed to process fill async: %v", err)
				}
			}
		}
	} else {
		if err := ei.portfolioManager.OnOrderFilled(ctx, fill); err != nil {
			if ei.config.LogExecutions {
				log.Printf("Failed to process fill sync: %v", err)
			}
		}
	}
	
	return nil
}

// OnOrderCompleted handles order completion notifications
func (ei *ExecutionIntegration) OnOrderCompleted(orderID string) {
	// Remove order tracking when completed
	ei.orderMutex.Lock()
	delete(ei.orderToPortfolio, orderID)
	ei.orderMutex.Unlock()
}

// GetPortfolioForOrder returns the portfolio ID associated with an order
func (ei *ExecutionIntegration) GetPortfolioForOrder(orderID string) (string, bool) {
	ei.orderMutex.RLock()
	defer ei.orderMutex.RUnlock()
	
	portfolioID, exists := ei.orderToPortfolio[orderID]
	return portfolioID, exists
}

// GetTrackedOrders returns all tracked orders for a portfolio
func (ei *ExecutionIntegration) GetTrackedOrders(portfolioID string) []string {
	ei.orderMutex.RLock()
	defer ei.orderMutex.RUnlock()
	
	var orders []string
	for orderID, pID := range ei.orderToPortfolio {
		if pID == portfolioID {
			orders = append(orders, orderID)
		}
	}
	
	return orders
}

// processExecutions handles execution results asynchronously
func (ei *ExecutionIntegration) processExecutions(ctx context.Context) {
	for {
		select {
		case <-ei.stopChan:
			return
		case <-ctx.Done():
			return
		case execution := <-ei.executionChan:
			if execution != nil {
				if err := ei.portfolioManager.OnOrderExecuted(ctx, execution); err != nil {
					if ei.config.LogExecutions {
						log.Printf("Failed to process execution for order %s: %v", execution.OrderID, err)
					}
				}
			}
		}
	}
}

// processFills handles order fills asynchronously
func (ei *ExecutionIntegration) processFills(ctx context.Context) {
	for {
		select {
		case <-ei.stopChan:
			return
		case <-ctx.Done():
			return
		case fill := <-ei.fillChan:
			if fill != nil {
				if err := ei.portfolioManager.OnOrderFilled(ctx, fill); err != nil {
					if ei.config.LogExecutions {
						log.Printf("Failed to process fill for order %s: %v", fill.OrderID, err)
					}
				}
			}
		}
	}
}

// PortfolioEnabledExecutionService wraps an execution service to automatically integrate with portfolio management
type PortfolioEnabledExecutionService struct {
	wrapped     ExecutionService
	integration *ExecutionIntegration
	defaultPortfolio string
}

// NewPortfolioEnabledExecutionService creates a wrapper that automatically handles portfolio integration
func NewPortfolioEnabledExecutionService(
	wrapped ExecutionService,
	integration *ExecutionIntegration,
	defaultPortfolio string,
) *PortfolioEnabledExecutionService {
	return &PortfolioEnabledExecutionService{
		wrapped:          wrapped,
		integration:      integration,
		defaultPortfolio: defaultPortfolio,
	}
}

// SubmitOrder submits an order using the default portfolio
func (pes *PortfolioEnabledExecutionService) SubmitOrder(ctx context.Context, order *domain.Order) (*ports.ExecutionResult, error) {
	return pes.integration.SubmitOrderWithPortfolio(ctx, pes.defaultPortfolio, order)
}

// SubmitOrderToPortfolio submits an order to a specific portfolio
func (pes *PortfolioEnabledExecutionService) SubmitOrderToPortfolio(ctx context.Context, portfolioID string, order *domain.Order) (*ports.ExecutionResult, error) {
	return pes.integration.SubmitOrderWithPortfolio(ctx, portfolioID, order)
}

// GetOrderStatus delegates to the wrapped service
func (pes *PortfolioEnabledExecutionService) GetOrderStatus(ctx context.Context, orderID string) (*domain.Order, error) {
	return pes.wrapped.GetOrderStatus(ctx, orderID)
}

// CancelOrder delegates to the wrapped service
func (pes *PortfolioEnabledExecutionService) CancelOrder(ctx context.Context, orderID string) error {
	err := pes.wrapped.CancelOrder(ctx, orderID)
	if err == nil {
		// Clean up tracking
		pes.integration.OnOrderCompleted(orderID)
	}
	return err
}

// GetMetrics delegates to the wrapped service
func (pes *PortfolioEnabledExecutionService) GetMetrics() ports.ExecutionMetrics {
	return pes.wrapped.GetMetrics()
}

// Start delegates to the wrapped service and starts integration
func (pes *PortfolioEnabledExecutionService) Start(ctx context.Context) error {
	if err := pes.wrapped.Start(ctx); err != nil {
		return err
	}
	return pes.integration.Start(ctx)
}

// Stop delegates to the wrapped service and stops integration  
func (pes *PortfolioEnabledExecutionService) Stop(ctx context.Context) error {
	pes.integration.Stop()
	return pes.wrapped.Stop(ctx)
}