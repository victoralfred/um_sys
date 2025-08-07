package portfolio

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/trading-engine/internal/core/domain"
	"github.com/trading-engine/internal/core/ports"
)

// MemoryRepository implements PortfolioRepository using in-memory storage
type MemoryRepository struct {
	portfolios map[string]*domain.Portfolio
	positions  map[string]map[string]*domain.Position // portfolioID -> positionID -> position
	snapshots  map[string][]*ports.PortfolioSnapshot  // portfolioID -> snapshots
	mutex      sync.RWMutex
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		portfolios: make(map[string]*domain.Portfolio),
		positions:  make(map[string]map[string]*domain.Position),
		snapshots:  make(map[string][]*ports.PortfolioSnapshot),
	}
}

// Save saves a portfolio to the repository
func (r *MemoryRepository) Save(ctx context.Context, portfolio *domain.Portfolio) error {
	if portfolio == nil {
		return fmt.Errorf("portfolio cannot be nil")
	}

	if portfolio.ID == "" {
		return fmt.Errorf("portfolio ID cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Deep copy to avoid external mutations
	portfolioCopy := *portfolio
	portfolioCopy.Positions = make(map[string]*domain.Position)
	portfolioCopy.AssetPositions = make(map[string]*domain.Position)

	for id, pos := range portfolio.Positions {
		positionCopy := *pos
		portfolioCopy.Positions[id] = &positionCopy
	}

	for symbol := range portfolio.AssetPositions {
		// Find the corresponding position in Positions map
		for _, p := range portfolioCopy.Positions {
			if p.Asset.Symbol == symbol {
				portfolioCopy.AssetPositions[symbol] = p
				break
			}
		}
	}

	r.portfolios[portfolio.ID] = &portfolioCopy

	return nil
}

// FindByID finds a portfolio by ID
func (r *MemoryRepository) FindByID(ctx context.Context, id string) (*domain.Portfolio, error) {
	if id == "" {
		return nil, fmt.Errorf("portfolio ID cannot be empty")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	portfolio, exists := r.portfolios[id]
	if !exists {
		return nil, fmt.Errorf("portfolio %s not found", id)
	}

	// Return a copy to avoid external mutations
	portfolioCopy := *portfolio
	portfolioCopy.Positions = make(map[string]*domain.Position)
	portfolioCopy.AssetPositions = make(map[string]*domain.Position)

	for id, pos := range portfolio.Positions {
		positionCopy := *pos
		portfolioCopy.Positions[id] = &positionCopy
	}

	for symbol := range portfolio.AssetPositions {
		// Find the corresponding position in Positions map
		for _, p := range portfolioCopy.Positions {
			if p.Asset.Symbol == symbol {
				portfolioCopy.AssetPositions[symbol] = p
				break
			}
		}
	}

	return &portfolioCopy, nil
}

// FindAll finds all portfolios matching the filter
func (r *MemoryRepository) FindAll(ctx context.Context, filter *ports.PortfolioFilter) ([]*domain.Portfolio, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*domain.Portfolio

	for _, portfolio := range r.portfolios {
		if r.matchesFilter(portfolio, filter) {
			// Return a copy
			portfolioCopy := *portfolio
			result = append(result, &portfolioCopy)
		}
	}

	// Sort by creation date (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	// Apply limit and offset
	if filter != nil {
		start := filter.Offset
		if start > len(result) {
			start = len(result)
		}

		end := len(result)
		if filter.Limit > 0 && start+filter.Limit < end {
			end = start + filter.Limit
		}

		result = result[start:end]
	}

	return result, nil
}

// Delete removes a portfolio from the repository
func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("portfolio ID cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.portfolios[id]; !exists {
		return fmt.Errorf("portfolio %s not found", id)
	}

	delete(r.portfolios, id)
	delete(r.positions, id)
	delete(r.snapshots, id)

	return nil
}

// SavePosition saves a position for a portfolio
func (r *MemoryRepository) SavePosition(ctx context.Context, portfolioID string, position *domain.Position) error {
	if portfolioID == "" {
		return fmt.Errorf("portfolio ID cannot be empty")
	}

	if position == nil {
		return fmt.Errorf("position cannot be nil")
	}

	if position.ID == "" {
		return fmt.Errorf("position ID cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Ensure portfolio exists
	if _, exists := r.portfolios[portfolioID]; !exists {
		return fmt.Errorf("portfolio %s not found", portfolioID)
	}

	// Initialize positions map for portfolio if needed
	if r.positions[portfolioID] == nil {
		r.positions[portfolioID] = make(map[string]*domain.Position)
	}

	// Save position copy
	positionCopy := *position
	r.positions[portfolioID][position.ID] = &positionCopy

	return nil
}

// FindPositions finds all positions for a portfolio
func (r *MemoryRepository) FindPositions(ctx context.Context, portfolioID string) ([]*domain.Position, error) {
	if portfolioID == "" {
		return nil, fmt.Errorf("portfolio ID cannot be empty")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	positions, exists := r.positions[portfolioID]
	if !exists {
		return []*domain.Position{}, nil
	}

	var result []*domain.Position
	for _, position := range positions {
		positionCopy := *position
		result = append(result, &positionCopy)
	}

	// Sort by opened date
	sort.Slice(result, func(i, j int) bool {
		return result[i].OpenedAt.After(result[j].OpenedAt)
	})

	return result, nil
}

// FindPositionBySymbol finds a position by symbol for a portfolio
func (r *MemoryRepository) FindPositionBySymbol(ctx context.Context, portfolioID string, symbol string) (*domain.Position, error) {
	if portfolioID == "" {
		return nil, fmt.Errorf("portfolio ID cannot be empty")
	}

	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	positions, exists := r.positions[portfolioID]
	if !exists {
		return nil, fmt.Errorf("position for symbol %s not found in portfolio %s", symbol, portfolioID)
	}

	for _, position := range positions {
		if position.Asset.Symbol == symbol {
			positionCopy := *position
			return &positionCopy, nil
		}
	}

	return nil, fmt.Errorf("position for symbol %s not found in portfolio %s", symbol, portfolioID)
}

// SaveSnapshot saves a portfolio snapshot
func (r *MemoryRepository) SaveSnapshot(ctx context.Context, snapshot *ports.PortfolioSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot cannot be nil")
	}

	if snapshot.PortfolioID == "" {
		return fmt.Errorf("portfolio ID cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Save snapshot copy
	snapshotCopy := *snapshot
	r.snapshots[snapshot.PortfolioID] = append(r.snapshots[snapshot.PortfolioID], &snapshotCopy)

	// Keep only last 1000 snapshots per portfolio to prevent memory growth
	snapshots := r.snapshots[snapshot.PortfolioID]
	if len(snapshots) > 1000 {
		// Remove oldest snapshots
		copy(snapshots[0:], snapshots[len(snapshots)-1000:])
		r.snapshots[snapshot.PortfolioID] = snapshots[:1000]
	}

	return nil
}

// GetSnapshots gets portfolio snapshots for a time period
func (r *MemoryRepository) GetSnapshots(ctx context.Context, portfolioID string, from, to time.Time) ([]*ports.PortfolioSnapshot, error) {
	if portfolioID == "" {
		return nil, fmt.Errorf("portfolio ID cannot be empty")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	snapshots, exists := r.snapshots[portfolioID]
	if !exists {
		return []*ports.PortfolioSnapshot{}, nil
	}

	var result []*ports.PortfolioSnapshot
	for _, snapshot := range snapshots {
		if (snapshot.Timestamp.Equal(from) || snapshot.Timestamp.After(from)) &&
			(snapshot.Timestamp.Equal(to) || snapshot.Timestamp.Before(to)) {
			snapshotCopy := *snapshot
			result = append(result, &snapshotCopy)
		}
	}

	// Sort by timestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

// Helper methods
func (r *MemoryRepository) matchesFilter(portfolio *domain.Portfolio, filter *ports.PortfolioFilter) bool {
	if filter == nil {
		return true
	}

	// Status filter
	if filter.Status != nil && portfolio.Status != *filter.Status {
		return false
	}

	// Capital filters
	if filter.MinCapital != nil && portfolio.InitialCapital.Cmp(*filter.MinCapital) < 0 {
		return false
	}

	if filter.MaxCapital != nil && portfolio.InitialCapital.Cmp(*filter.MaxCapital) > 0 {
		return false
	}

	// Date filters
	if filter.CreatedAfter != nil && portfolio.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && portfolio.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

// GetPortfolioCount returns the total number of portfolios (useful for testing)
func (r *MemoryRepository) GetPortfolioCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.portfolios)
}

// Clear removes all data from the repository (useful for testing)
func (r *MemoryRepository) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.portfolios = make(map[string]*domain.Portfolio)
	r.positions = make(map[string]map[string]*domain.Position)
	r.snapshots = make(map[string][]*ports.PortfolioSnapshot)
}
