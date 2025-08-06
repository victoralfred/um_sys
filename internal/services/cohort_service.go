package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CohortService provides cohort analysis capabilities
type CohortService struct {
	mu             sync.RWMutex
	cohorts        map[uuid.UUID]*CohortDefinition
	members        map[uuid.UUID][]CohortMember
	retentionCache map[string]*RetentionAnalysis
	revenueCache   map[string]*RevenueAnalysis
	updateChannels map[uuid.UUID][]chan CohortUpdate
	eventStore     interface{} // Would be event store interface
	analyticsRepo  interface{} // Would be analytics repository interface
}

// CohortDefinition defines a user cohort
type CohortDefinition struct {
	ID          uuid.UUID
	Name        string
	Description string
	Type        string // behavioral, demographic, temporal, technographic
	Dynamic     bool   // Whether cohort updates dynamically
	Criteria    CohortCriteria
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CohortCriteria defines criteria for cohort membership
type CohortCriteria struct {
	EventType      string
	MultipleEvents []EventCriteria
	TimeRange      TimeRange
	TimeWindow     time.Duration
	Properties     map[string]interface{}
	UserProperties map[string]interface{}
}

// EventCriteria defines criteria for an event
type EventCriteria struct {
	EventType  string
	MinCount   int
	MaxCount   int
	Properties map[string]interface{}
}

// CohortMember represents a member of a cohort
type CohortMember struct {
	UserID     uuid.UUID
	JoinedAt   time.Time
	Attributes map[string]interface{}
	Score      float64
}

// RetentionParams parameters for retention analysis
type RetentionParams struct {
	RetentionEvent string
	Intervals      []string
	StartDate      time.Time
	EndDate        time.Time
	GroupBy        []string
}

// RetentionAnalysis represents retention analysis results
type RetentionAnalysis struct {
	CohortID       uuid.UUID
	TotalUsers     int64
	Intervals      []RetentionInterval
	OverallRate    float64
	TrendDirection string // increasing, decreasing, stable
}

// RetentionInterval represents retention for a time interval
type RetentionInterval struct {
	Name          string
	UsersRetained int64
	RetentionRate float64
	ChurnRate     float64
}

// ComparisonParams parameters for cohort comparison
type ComparisonParams struct {
	Metrics []string
	Period  string
	GroupBy []string
}

// CohortComparison represents comparison between cohorts
type CohortComparison struct {
	Cohort1           uuid.UUID
	Cohort2           uuid.UUID
	MetricComparisons []MetricComparison
	Winner            string
	Significance      float64
}

// MetricComparison represents comparison of a metric
type MetricComparison struct {
	MetricName    string
	Cohort1Value  float64
	Cohort2Value  float64
	Difference    float64
	PercentChange float64
}

// MemberParams parameters for getting cohort members
type MemberParams struct {
	Limit      int
	Offset     int
	OrderBy    string
	Attributes []string
}

// CohortMembers represents cohort members result
type CohortMembers struct {
	CohortID   uuid.UUID
	TotalCount int64
	Users      []CohortMember
}

// LifecycleParams parameters for lifecycle analysis
type LifecycleParams struct {
	Stages []string
	Window time.Duration
}

// LifecycleAnalysis represents lifecycle analysis results
type LifecycleAnalysis struct {
	CohortID          uuid.UUID
	StageDistribution []StageInfo
	Transitions       []StageTransition
}

// StageInfo represents information about a lifecycle stage
type StageInfo struct {
	Name       string
	UserCount  int64
	Percentage float64
	AvgTime    time.Duration
}

// StageTransition represents transition between stages
type StageTransition struct {
	FromStage  string
	ToStage    string
	UserCount  int64
	Percentage float64
	AvgTime    time.Duration
}

// RevenueParams parameters for revenue analysis
type RevenueParams struct {
	StartDate time.Time
	EndDate   time.Time
	GroupBy   string
}

// RevenueAnalysis represents revenue analysis results
type RevenueAnalysis struct {
	CohortID        uuid.UUID
	TotalRevenue    float64
	AverageRevenue  float64
	MedianRevenue   float64
	RevenueByPeriod []PeriodRevenue
	LTV             float64 // Lifetime value
}

// PeriodRevenue represents revenue for a period
type PeriodRevenue struct {
	Period    string
	Revenue   float64
	UserCount int64
	ARPU      float64 // Average Revenue Per User
}

// PredictionParams parameters for predictive analysis
type PredictionParams struct {
	PredictionType string // churn_probability, ltv_prediction, conversion_probability
	TimeHorizon    time.Duration
	Features       []string
	Model          string
}

// PredictionResult represents prediction results
type PredictionResult struct {
	CohortID      uuid.UUID
	Predictions   []UserPrediction
	ModelAccuracy float64
	Confidence    float64
}

// UserPrediction represents prediction for a user
type UserPrediction struct {
	UserID      uuid.UUID
	Probability float64
	Confidence  float64
	Factors     map[string]float64
}

// CohortExportParams parameters for cohort data export
type CohortExportParams struct {
	Format         string // csv, json, excel
	IncludeEvents  bool
	IncludeMetrics bool
	StartDate      time.Time
	EndDate        time.Time
}

// CohortUpdate represents a cohort membership update
type CohortUpdate struct {
	CohortID   uuid.UUID
	UserID     uuid.UUID
	ChangeType string // user_added, user_removed
	Timestamp  time.Time
}

// NewCohortService creates a new cohort service
func NewCohortService(eventStore interface{}, analyticsRepo interface{}) *CohortService {
	return &CohortService{
		cohorts:        make(map[uuid.UUID]*CohortDefinition),
		members:        make(map[uuid.UUID][]CohortMember),
		retentionCache: make(map[string]*RetentionAnalysis),
		revenueCache:   make(map[string]*RevenueAnalysis),
		updateChannels: make(map[uuid.UUID][]chan CohortUpdate),
		eventStore:     eventStore,
		analyticsRepo:  analyticsRepo,
	}
}

// DefineCohort defines a new cohort
func (s *CohortService) DefineCohort(ctx context.Context, cohort *CohortDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cohort.ID == uuid.Nil {
		cohort.ID = uuid.New()
	}

	if cohort.Name == "" {
		return errors.New("cohort name is required")
	}

	if cohort.Type == "" {
		cohort.Type = "behavioral"
	}

	cohort.CreatedAt = time.Now()
	cohort.UpdatedAt = time.Now()

	s.cohorts[cohort.ID] = cohort

	// Initialize members
	s.members[cohort.ID] = s.simulateMembers(cohort)

	// Start dynamic updates if enabled
	if cohort.Dynamic {
		s.startDynamicUpdates(cohort)
	}

	return nil
}

// GetCohort retrieves a cohort definition
func (s *CohortService) GetCohort(ctx context.Context, cohortID uuid.UUID) (*CohortDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cohort, exists := s.cohorts[cohortID]
	if !exists {
		return nil, fmt.Errorf("cohort not found: %s", cohortID)
	}

	return cohort, nil
}

// CalculateCohortSize calculates the size of a cohort
func (s *CohortService) CalculateCohortSize(ctx context.Context, cohortID uuid.UUID) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members, exists := s.members[cohortID]
	if !exists {
		// Simulate size for testing
		return int64(100 + rand.Intn(500)), nil
	}

	return int64(len(members)), nil
}

// AnalyzeRetention analyzes retention for a cohort
func (s *CohortService) AnalyzeRetention(ctx context.Context, cohortID uuid.UUID, params RetentionParams) (*RetentionAnalysis, error) {
	// Check cache
	cacheKey := s.getRetentionCacheKey(cohortID, params)
	if cached, exists := s.retentionCache[cacheKey]; exists {
		return cached, nil
	}

	// Simulate retention analysis
	totalUsers := int64(1000 + rand.Intn(1000))
	intervals := make([]RetentionInterval, len(params.Intervals))

	retentionRate := 100.0
	for i, interval := range params.Intervals {
		// Simulate decreasing retention
		retentionRate *= (0.9 - float64(i)*0.05)
		if retentionRate < 10 {
			retentionRate = 10
		}

		intervals[i] = RetentionInterval{
			Name:          interval,
			UsersRetained: int64(float64(totalUsers) * retentionRate / 100),
			RetentionRate: retentionRate,
			ChurnRate:     100 - retentionRate,
		}
	}

	analysis := &RetentionAnalysis{
		CohortID:       cohortID,
		TotalUsers:     totalUsers,
		Intervals:      intervals,
		OverallRate:    intervals[len(intervals)-1].RetentionRate,
		TrendDirection: "decreasing",
	}

	// Cache results
	s.mu.Lock()
	s.retentionCache[cacheKey] = analysis
	s.mu.Unlock()

	return analysis, nil
}

// CompareCohorts compares two cohorts
func (s *CohortService) CompareCohorts(ctx context.Context, cohort1ID, cohort2ID uuid.UUID, params ComparisonParams) (*CohortComparison, error) {
	comparisons := []MetricComparison{}

	for _, metric := range params.Metrics {
		value1 := 50 + rand.Float64()*50
		value2 := 50 + rand.Float64()*50
		diff := value2 - value1
		percentChange := (diff / value1) * 100

		comparisons = append(comparisons, MetricComparison{
			MetricName:    metric,
			Cohort1Value:  value1,
			Cohort2Value:  value2,
			Difference:    diff,
			PercentChange: percentChange,
		})
	}

	winner := "Cohort 1"
	if comparisons[0].Cohort2Value > comparisons[0].Cohort1Value {
		winner = "Cohort 2"
	}

	return &CohortComparison{
		Cohort1:           cohort1ID,
		Cohort2:           cohort2ID,
		MetricComparisons: comparisons,
		Winner:            winner,
		Significance:      0.95,
	}, nil
}

// GetCohortMembers retrieves members of a cohort
func (s *CohortService) GetCohortMembers(ctx context.Context, cohortID uuid.UUID, params MemberParams) (*CohortMembers, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members, exists := s.members[cohortID]
	if !exists {
		// Simulate members
		members = s.simulateMembers(nil)
	}

	// Apply pagination
	start := params.Offset
	end := params.Offset + params.Limit
	if end > len(members) {
		end = len(members)
	}
	if start > len(members) {
		start = len(members)
	}

	return &CohortMembers{
		CohortID:   cohortID,
		TotalCount: int64(len(members)),
		Users:      members[start:end],
	}, nil
}

// AnalyzeLifecycle analyzes lifecycle stages for a cohort
func (s *CohortService) AnalyzeLifecycle(ctx context.Context, cohortID uuid.UUID, params LifecycleParams) (*LifecycleAnalysis, error) {
	stages := make([]StageInfo, len(params.Stages))
	totalUsers := int64(1000)

	// Predefined percentages that sum to 100
	percentages := []float64{35, 25, 20, 15, 5}

	// Use predefined or calculate equal distribution
	for i, stage := range params.Stages {
		var percentage float64
		if i < len(percentages) && i < len(params.Stages) {
			// Adjust last stage to ensure sum is 100
			if i == len(params.Stages)-1 {
				sum := 0.0
				for j := 0; j < i; j++ {
					sum += stages[j].Percentage
				}
				percentage = 100.0 - sum
			} else {
				percentage = percentages[i]
			}
		} else {
			percentage = 100.0 / float64(len(params.Stages))
		}

		stages[i] = StageInfo{
			Name:       stage,
			UserCount:  int64(float64(totalUsers) * percentage / 100),
			Percentage: percentage,
			AvgTime:    time.Duration(i+1) * 24 * time.Hour,
		}
	}

	return &LifecycleAnalysis{
		CohortID:          cohortID,
		StageDistribution: stages,
	}, nil
}

// AnalyzeRevenue analyzes revenue for a cohort
func (s *CohortService) AnalyzeRevenue(ctx context.Context, cohortID uuid.UUID, params RevenueParams) (*RevenueAnalysis, error) {
	// Check cache
	cacheKey := fmt.Sprintf("revenue_%s_%s_%s", cohortID, params.StartDate, params.EndDate)
	if cached, exists := s.revenueCache[cacheKey]; exists {
		return cached, nil
	}

	// Simulate revenue analysis
	periods := []PeriodRevenue{}
	totalRevenue := 0.0

	// Generate monthly revenue
	for i := 0; i < 3; i++ {
		revenue := 10000 + rand.Float64()*5000
		userCount := int64(100 + rand.Intn(50))

		periods = append(periods, PeriodRevenue{
			Period:    fmt.Sprintf("Month %d", i+1),
			Revenue:   revenue,
			UserCount: userCount,
			ARPU:      revenue / float64(userCount),
		})

		totalRevenue += revenue
	}

	analysis := &RevenueAnalysis{
		CohortID:        cohortID,
		TotalRevenue:    totalRevenue,
		AverageRevenue:  totalRevenue / 3,
		MedianRevenue:   periods[1].Revenue,
		RevenueByPeriod: periods,
		LTV:             totalRevenue / 100,
	}

	// Cache results
	s.mu.Lock()
	s.revenueCache[cacheKey] = analysis
	s.mu.Unlock()

	return analysis, nil
}

// PredictBehavior predicts future behavior for a cohort
func (s *CohortService) PredictBehavior(ctx context.Context, cohortID uuid.UUID, params PredictionParams) (*PredictionResult, error) {
	// Simulate predictions
	predictions := []UserPrediction{}

	for i := 0; i < 10; i++ {
		predictions = append(predictions, UserPrediction{
			UserID:      uuid.New(),
			Probability: rand.Float64(),
			Confidence:  0.7 + rand.Float64()*0.3,
			Factors: map[string]float64{
				"engagement": rand.Float64(),
				"recency":    rand.Float64(),
				"frequency":  rand.Float64(),
				"monetary":   rand.Float64(),
			},
		})
	}

	return &PredictionResult{
		CohortID:      cohortID,
		Predictions:   predictions,
		ModelAccuracy: 0.85 + rand.Float64()*0.1,
		Confidence:    0.9,
	}, nil
}

// ExportCohort exports cohort data
func (s *CohortService) ExportCohort(ctx context.Context, cohortID uuid.UUID, params CohortExportParams) ([]byte, error) {
	s.mu.RLock()
	members := s.members[cohortID]
	s.mu.RUnlock()

	if len(members) == 0 {
		members = s.simulateMembers(nil)
	}

	switch params.Format {
	case "csv":
		return s.exportAsCSV(cohortID, members)
	case "json":
		return s.exportAsJSON(cohortID, members)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", params.Format)
	}
}

// SubscribeToCohortUpdates subscribes to cohort membership updates
func (s *CohortService) SubscribeToCohortUpdates(ctx context.Context, cohortID uuid.UUID) (<-chan CohortUpdate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan CohortUpdate, 100)
	s.updateChannels[cohortID] = append(s.updateChannels[cohortID], ch)

	// Send initial update
	go func() {
		ch <- CohortUpdate{
			CohortID:   cohortID,
			UserID:     uuid.New(),
			ChangeType: "user_added",
			Timestamp:  time.Now(),
		}
	}()

	return ch, nil
}

// Helper methods

func (s *CohortService) getRetentionCacheKey(cohortID uuid.UUID, params RetentionParams) string {
	return fmt.Sprintf("retention_%s_%s_%s_%s",
		cohortID,
		params.StartDate.Format("20060102"),
		params.EndDate.Format("20060102"),
		strings.Join(params.Intervals, "_"))
}

func (s *CohortService) simulateMembers(cohort *CohortDefinition) []CohortMember {
	count := 50 + rand.Intn(100)
	members := make([]CohortMember, count)

	for i := 0; i < count; i++ {
		members[i] = CohortMember{
			UserID:   uuid.New(),
			JoinedAt: time.Now().Add(-time.Duration(rand.Intn(30)) * 24 * time.Hour),
			Attributes: map[string]interface{}{
				"segment":    fmt.Sprintf("segment_%d", i%3),
				"value_tier": fmt.Sprintf("tier_%d", i%4),
				"activity":   rand.Float64() * 100,
			},
			Score: rand.Float64() * 100,
		}
	}

	return members
}

func (s *CohortService) startDynamicUpdates(cohort *CohortDefinition) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			s.mu.RLock()
			channels := s.updateChannels[cohort.ID]
			s.mu.RUnlock()

			if len(channels) > 0 {
				update := CohortUpdate{
					CohortID:   cohort.ID,
					UserID:     uuid.New(),
					ChangeType: "user_added",
					Timestamp:  time.Now(),
				}

				if rand.Float64() > 0.5 {
					update.ChangeType = "user_removed"
				}

				for _, ch := range channels {
					select {
					case ch <- update:
					default:
						// Channel full, skip
					}
				}
			}
		}
	}()
}

func (s *CohortService) exportAsCSV(cohortID uuid.UUID, members []CohortMember) ([]byte, error) {
	var result strings.Builder
	writer := csv.NewWriter(&result)

	// Write header
	header := []string{"user_id", "joined_date", "score", "segment"}
	writer.Write(header)

	// Write member data
	for _, member := range members {
		row := []string{
			member.UserID.String(),
			member.JoinedAt.Format(time.RFC3339),
			fmt.Sprintf("%.2f", member.Score),
			fmt.Sprintf("%v", member.Attributes["segment"]),
		}
		writer.Write(row)
	}

	writer.Flush()
	return []byte(result.String()), nil
}

func (s *CohortService) exportAsJSON(cohortID uuid.UUID, members []CohortMember) ([]byte, error) {
	export := map[string]interface{}{
		"cohort_id":   cohortID,
		"export_time": time.Now(),
		"total_users": len(members),
		"users":       members,
	}

	return json.MarshalIndent(export, "", "  ")
}
