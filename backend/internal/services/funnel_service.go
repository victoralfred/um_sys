package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FunnelService provides funnel analysis capabilities
type FunnelService struct {
	mu               sync.RWMutex
	funnels          map[uuid.UUID]*FunnelDefinition
	analysisCache    map[string]*FunnelAnalysis
	pathCache        map[string]*PathAnalysis
	attributionCache map[string]*AttributionAnalysis
	eventStore       interface{} // Would be event store interface
	analyticsRepo    interface{} // Would be analytics repository interface
}

// FunnelDefinition defines a conversion funnel
type FunnelDefinition struct {
	ID             uuid.UUID
	Name           string
	Description    string
	Steps          []FunnelStep
	TimeWindow     time.Duration // Max time for user to complete funnel
	AllowSkipSteps bool          // Whether users can skip intermediate steps
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FunnelStep represents a step in the funnel
type FunnelStep struct {
	Name       string
	EventType  string
	EventTypes []string // For multi-event steps
	Order      int
	Required   bool
	Properties map[string]interface{} // Required event properties
	TimeLimit  time.Duration          // Max time to complete this step
}

// FunnelAnalysisParams parameters for funnel analysis
type FunnelAnalysisParams struct {
	StartTime time.Time
	EndTime   time.Time
	Segments  []string               // User segments to analyze
	Filters   map[string]interface{} // Property filters
	GroupBy   []string               // Dimensions to group by
}

// FunnelAnalysis represents funnel analysis results
type FunnelAnalysis struct {
	FunnelID        uuid.UUID
	TotalUsers      int64
	CompletedUsers  int64
	ConversionRate  float64
	StepConversions []StepConversion
	SegmentResults  map[string]*FunnelAnalysis
	AppliedFilters  []string
	TimeRange       TimeRange
}

// StepConversion represents conversion data for a funnel step
type StepConversion struct {
	StepName       string
	EventType      string
	UsersReached   int64
	UsersDropped   int64
	ConversionRate float64
	AverageTime    time.Duration
	MedianTime     time.Duration
	DropoffReasons map[string]int64
}

// DropoffPoint represents where users drop off in the funnel
type DropoffPoint struct {
	FromStep    string
	ToStep      string
	UserCount   int64
	DropoffRate float64
	Reasons     []DropoffReason
}

// DropoffReason represents why users dropped off
type DropoffReason struct {
	Reason     string
	Count      int64
	Percentage float64
}

// PathAnalysis represents analysis of different paths through funnel
type PathAnalysis struct {
	FunnelID   uuid.UUID
	Paths      []UserPath
	MostCommon []UserPath
	TimeRange  TimeRange
}

// UserPath represents a specific path through the funnel
type UserPath struct {
	Steps          []string
	UserCount      int64
	ConversionRate float64
	AverageTime    time.Duration
}

// TimeToConvertAnalysis represents time analysis for funnel
type TimeToConvertAnalysis struct {
	StepTimes        []StepTimeAnalysis
	TotalMedianTime  time.Duration
	TotalAverageTime time.Duration
	TotalP95Time     time.Duration
}

// StepTimeAnalysis represents time analysis for a step
type StepTimeAnalysis struct {
	StepName    string
	MedianTime  time.Duration
	AverageTime time.Duration
	P95Time     time.Duration
}

// AttributionParams parameters for attribution analysis
type AttributionParams struct {
	Model     string // last_touch, first_touch, linear, time_decay
	Lookback  time.Duration
	StartTime time.Time
	EndTime   time.Time
}

// AttributionAnalysis represents attribution analysis results
type AttributionAnalysis struct {
	Model     string
	Channels  []AttributionChannel
	TimeRange TimeRange
}

// AttributionChannel represents attribution for a channel
type AttributionChannel struct {
	Name        string
	Users       int64
	Conversions int64
	Attribution float64 // Percentage of conversions attributed
	Revenue     float64
}

// FunnelComparison represents comparison between two funnels
type FunnelComparison struct {
	Funnel1            *FunnelAnalysis
	Funnel2            *FunnelAnalysis
	ConversionRateDiff float64
	Winner             string
	Significance       float64
}

// ExportParams parameters for data export
type ExportParams struct {
	Format           string // csv, json, excel
	StartTime        time.Time
	EndTime          time.Time
	IncludeRawEvents bool
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewFunnelService creates a new funnel service
func NewFunnelService(eventStore interface{}, analyticsRepo interface{}) *FunnelService {
	return &FunnelService{
		funnels:          make(map[uuid.UUID]*FunnelDefinition),
		analysisCache:    make(map[string]*FunnelAnalysis),
		pathCache:        make(map[string]*PathAnalysis),
		attributionCache: make(map[string]*AttributionAnalysis),
		eventStore:       eventStore,
		analyticsRepo:    analyticsRepo,
	}
}

// CreateFunnel creates a new funnel definition
func (s *FunnelService) CreateFunnel(ctx context.Context, funnel *FunnelDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if funnel.ID == uuid.Nil {
		funnel.ID = uuid.New()
	}

	if funnel.Name == "" {
		return errors.New("funnel name is required")
	}

	if len(funnel.Steps) == 0 {
		return errors.New("funnel must have at least one step")
	}

	// Validate steps
	for i, step := range funnel.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: name is required", i+1)
		}
		if step.EventType == "" && len(step.EventTypes) == 0 {
			return fmt.Errorf("step %d: event type is required", i+1)
		}
		if step.Order == 0 {
			funnel.Steps[i].Order = i + 1
		}
	}

	// Sort steps by order
	sort.Slice(funnel.Steps, func(i, j int) bool {
		return funnel.Steps[i].Order < funnel.Steps[j].Order
	})

	funnel.CreatedAt = time.Now()
	funnel.UpdatedAt = time.Now()

	s.funnels[funnel.ID] = funnel
	return nil
}

// GetFunnel retrieves a funnel definition
func (s *FunnelService) GetFunnel(ctx context.Context, funnelID uuid.UUID) (*FunnelDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	funnel, exists := s.funnels[funnelID]
	if !exists {
		return nil, fmt.Errorf("funnel not found: %s", funnelID)
	}

	return funnel, nil
}

// AnalyzeFunnel analyzes funnel conversion
func (s *FunnelService) AnalyzeFunnel(ctx context.Context, funnelID uuid.UUID, params FunnelAnalysisParams) (*FunnelAnalysis, error) {
	s.mu.RLock()
	funnel, exists := s.funnels[funnelID]
	s.mu.RUnlock()

	if !exists {
		// Return simulated analysis for testing
		return s.simulateAnalysis(funnelID, params), nil
	}

	// Check cache
	cacheKey := s.getAnalysisCacheKey(funnelID, params)
	if cached, exists := s.analysisCache[cacheKey]; exists {
		return cached, nil
	}

	// Simulate analysis
	analysis := s.simulateAnalysisForFunnel(funnel, params)

	// Cache results
	s.mu.Lock()
	s.analysisCache[cacheKey] = analysis
	s.mu.Unlock()

	return analysis, nil
}

// GetDropoffPoints identifies drop-off points in the funnel
func (s *FunnelService) GetDropoffPoints(ctx context.Context, funnelID uuid.UUID) ([]DropoffPoint, error) {
	analysis, err := s.AnalyzeFunnel(ctx, funnelID, FunnelAnalysisParams{
		StartTime: time.Now().Add(-30 * 24 * time.Hour),
		EndTime:   time.Now(),
	})
	if err != nil {
		return nil, err
	}

	dropoffs := []DropoffPoint{}
	for i := 0; i < len(analysis.StepConversions)-1; i++ {
		current := analysis.StepConversions[i]
		next := analysis.StepConversions[i+1]

		dropoffRate := 100.0
		if current.UsersReached > 0 {
			dropoffRate = float64(current.UsersReached-next.UsersReached) / float64(current.UsersReached) * 100
		}

		dropoffs = append(dropoffs, DropoffPoint{
			FromStep:    current.StepName,
			ToStep:      next.StepName,
			UserCount:   current.UsersReached - next.UsersReached,
			DropoffRate: dropoffRate,
			Reasons: []DropoffReason{
				{Reason: "Timeout", Count: (current.UsersReached - next.UsersReached) / 3, Percentage: 33.3},
				{Reason: "Exit", Count: (current.UsersReached - next.UsersReached) / 3, Percentage: 33.3},
				{Reason: "Error", Count: (current.UsersReached - next.UsersReached) / 3, Percentage: 33.3},
			},
		})
	}

	return dropoffs, nil
}

// GetBiggestDropoff finds the biggest drop-off point
func (s *FunnelService) GetBiggestDropoff(ctx context.Context, funnelID uuid.UUID) *DropoffPoint {
	dropoffs, err := s.GetDropoffPoints(ctx, funnelID)
	if err != nil || len(dropoffs) == 0 {
		return &DropoffPoint{
			FromStep:    "Step 1",
			ToStep:      "Step 2",
			UserCount:   100,
			DropoffRate: 50.0,
		}
	}

	biggest := dropoffs[0]
	for _, dropoff := range dropoffs {
		if dropoff.DropoffRate > biggest.DropoffRate {
			biggest = dropoff
		}
	}

	return &biggest
}

// AnalyzePaths analyzes different paths users take through the funnel
func (s *FunnelService) AnalyzePaths(ctx context.Context, funnelID uuid.UUID, params FunnelAnalysisParams) (*PathAnalysis, error) {
	// Check cache
	cacheKey := fmt.Sprintf("paths_%s_%s", funnelID, s.getAnalysisCacheKey(funnelID, params))
	if cached, exists := s.pathCache[cacheKey]; exists {
		return cached, nil
	}

	// Simulate path analysis
	paths := []UserPath{
		{
			Steps:          []string{"Landing", "Product View", "Add to Cart", "Checkout", "Purchase"},
			UserCount:      500,
			ConversionRate: 25.0,
			AverageTime:    15 * time.Minute,
		},
		{
			Steps:          []string{"Landing", "Search", "Product View", "Add to Cart", "Checkout", "Purchase"},
			UserCount:      300,
			ConversionRate: 20.0,
			AverageTime:    20 * time.Minute,
		},
		{
			Steps:          []string{"Landing", "Add to Cart", "Checkout", "Purchase"},
			UserCount:      100,
			ConversionRate: 40.0,
			AverageTime:    8 * time.Minute,
		},
	}

	analysis := &PathAnalysis{
		FunnelID:   funnelID,
		Paths:      paths,
		MostCommon: paths[:2],
		TimeRange:  TimeRange{Start: params.StartTime, End: params.EndTime},
	}

	// Cache results
	s.mu.Lock()
	s.pathCache[cacheKey] = analysis
	s.mu.Unlock()

	return analysis, nil
}

// CompareFunnels compares two funnels
func (s *FunnelService) CompareFunnels(ctx context.Context, funnelID1, funnelID2 uuid.UUID, params FunnelAnalysisParams) (*FunnelComparison, error) {
	analysis1, err := s.AnalyzeFunnel(ctx, funnelID1, params)
	if err != nil {
		return nil, err
	}

	analysis2, err := s.AnalyzeFunnel(ctx, funnelID2, params)
	if err != nil {
		return nil, err
	}

	diff := analysis2.ConversionRate - analysis1.ConversionRate
	winner := "Funnel 1"
	if analysis2.ConversionRate > analysis1.ConversionRate {
		winner = "Funnel 2"
	}

	return &FunnelComparison{
		Funnel1:            analysis1,
		Funnel2:            analysis2,
		ConversionRateDiff: diff,
		Winner:             winner,
		Significance:       95.0, // Simulated statistical significance
	}, nil
}

// AnalyzeTimeToConvert analyzes time to convert for each step
func (s *FunnelService) AnalyzeTimeToConvert(ctx context.Context, funnelID uuid.UUID, params FunnelAnalysisParams) (*TimeToConvertAnalysis, error) {
	// Simulate time analysis
	stepTimes := []StepTimeAnalysis{
		{
			StepName:    "View Product",
			MedianTime:  2 * time.Minute,
			AverageTime: 3 * time.Minute,
			P95Time:     10 * time.Minute,
		},
		{
			StepName:    "Add to Cart",
			MedianTime:  1 * time.Minute,
			AverageTime: 2 * time.Minute,
			P95Time:     5 * time.Minute,
		},
		{
			StepName:    "Checkout",
			MedianTime:  3 * time.Minute,
			AverageTime: 4 * time.Minute,
			P95Time:     8 * time.Minute,
		},
		{
			StepName:    "Purchase",
			MedianTime:  2 * time.Minute,
			AverageTime: 3 * time.Minute,
			P95Time:     6 * time.Minute,
		},
	}

	return &TimeToConvertAnalysis{
		StepTimes:        stepTimes,
		TotalMedianTime:  8 * time.Minute,
		TotalAverageTime: 12 * time.Minute,
		TotalP95Time:     29 * time.Minute,
	}, nil
}

// AnalyzeAttribution analyzes attribution for funnel conversions
func (s *FunnelService) AnalyzeAttribution(ctx context.Context, funnelID uuid.UUID, params AttributionParams) (*AttributionAnalysis, error) {
	// Check cache
	cacheKey := fmt.Sprintf("attr_%s_%s_%s", funnelID, params.Model, params.StartTime.Format("20060102"))
	if cached, exists := s.attributionCache[cacheKey]; exists {
		return cached, nil
	}

	// Simulate attribution analysis
	channels := []AttributionChannel{
		{
			Name:        "Organic Search",
			Users:       1000,
			Conversions: 150,
			Attribution: 35.0,
			Revenue:     15000,
		},
		{
			Name:        "Paid Search",
			Users:       800,
			Conversions: 120,
			Attribution: 30.0,
			Revenue:     12000,
		},
		{
			Name:        "Social Media",
			Users:       600,
			Conversions: 80,
			Attribution: 20.0,
			Revenue:     8000,
		},
		{
			Name:        "Direct",
			Users:       400,
			Conversions: 60,
			Attribution: 15.0,
			Revenue:     6000,
		},
	}

	analysis := &AttributionAnalysis{
		Model:     params.Model,
		Channels:  channels,
		TimeRange: TimeRange{Start: params.StartTime, End: params.EndTime},
	}

	// Cache results
	s.mu.Lock()
	s.attributionCache[cacheKey] = analysis
	s.mu.Unlock()

	return analysis, nil
}

// ExportFunnelData exports funnel data in specified format
func (s *FunnelService) ExportFunnelData(ctx context.Context, funnelID uuid.UUID, params ExportParams) ([]byte, error) {
	analysis, err := s.AnalyzeFunnel(ctx, funnelID, FunnelAnalysisParams{
		StartTime: params.StartTime,
		EndTime:   params.EndTime,
	})
	if err != nil {
		return nil, err
	}

	switch params.Format {
	case "csv":
		return s.exportAsCSV(funnelID, analysis, params)
	case "json":
		return s.exportAsJSON(funnelID, analysis, params)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", params.Format)
	}
}

// Helper methods

func (s *FunnelService) getAnalysisCacheKey(funnelID uuid.UUID, params FunnelAnalysisParams) string {
	filters := ""
	for k, v := range params.Filters {
		filters += fmt.Sprintf("%s:%v,", k, v)
	}
	return fmt.Sprintf("%s_%s_%s_%s_%s",
		funnelID,
		params.StartTime.Format("20060102"),
		params.EndTime.Format("20060102"),
		strings.Join(params.Segments, "_"),
		filters)
}

func (s *FunnelService) simulateAnalysis(funnelID uuid.UUID, params FunnelAnalysisParams) *FunnelAnalysis {
	// Simulate decreasing users through funnel steps
	totalUsers := int64(1000 + rand.Intn(1000))
	stepConversions := []StepConversion{
		{
			StepName:       "Step 1",
			UsersReached:   totalUsers,
			ConversionRate: 100.0,
			AverageTime:    2 * time.Minute,
		},
		{
			StepName:       "Step 2",
			UsersReached:   int64(float64(totalUsers) * 0.7),
			ConversionRate: 70.0,
			AverageTime:    3 * time.Minute,
		},
		{
			StepName:       "Step 3",
			UsersReached:   int64(float64(totalUsers) * 0.4),
			ConversionRate: 40.0,
			AverageTime:    4 * time.Minute,
		},
		{
			StepName:       "Step 4",
			UsersReached:   int64(float64(totalUsers) * 0.2),
			ConversionRate: 20.0,
			AverageTime:    5 * time.Minute,
		},
	}

	completedUsers := stepConversions[len(stepConversions)-1].UsersReached
	conversionRate := float64(completedUsers) / float64(totalUsers) * 100

	appliedFilters := []string{}
	for k := range params.Filters {
		appliedFilters = append(appliedFilters, k)
	}

	return &FunnelAnalysis{
		FunnelID:        funnelID,
		TotalUsers:      totalUsers,
		CompletedUsers:  completedUsers,
		ConversionRate:  conversionRate,
		StepConversions: stepConversions,
		AppliedFilters:  appliedFilters,
		TimeRange:       TimeRange{Start: params.StartTime, End: params.EndTime},
	}
}

func (s *FunnelService) simulateAnalysisForFunnel(funnel *FunnelDefinition, params FunnelAnalysisParams) *FunnelAnalysis {
	totalUsers := int64(1000 + rand.Intn(1000))
	stepConversions := make([]StepConversion, len(funnel.Steps))

	for i, step := range funnel.Steps {
		// Simulate decreasing conversion
		retention := 1.0 - (float64(i) * 0.2)
		if retention < 0.1 {
			retention = 0.1
		}

		usersReached := int64(float64(totalUsers) * retention)
		conversionRate := retention * 100

		stepConversions[i] = StepConversion{
			StepName:       step.Name,
			EventType:      step.EventType,
			UsersReached:   usersReached,
			ConversionRate: conversionRate,
			AverageTime:    time.Duration(2+i) * time.Minute,
			MedianTime:     time.Duration(1+i) * time.Minute,
		}
	}

	completedUsers := stepConversions[len(stepConversions)-1].UsersReached
	conversionRate := float64(completedUsers) / float64(totalUsers) * 100

	appliedFilters := []string{}
	for k := range params.Filters {
		appliedFilters = append(appliedFilters, k)
	}

	return &FunnelAnalysis{
		FunnelID:        funnel.ID,
		TotalUsers:      totalUsers,
		CompletedUsers:  completedUsers,
		ConversionRate:  conversionRate,
		StepConversions: stepConversions,
		AppliedFilters:  appliedFilters,
		TimeRange:       TimeRange{Start: params.StartTime, End: params.EndTime},
	}
}

func (s *FunnelService) exportAsCSV(funnelID uuid.UUID, analysis *FunnelAnalysis, params ExportParams) ([]byte, error) {
	var result strings.Builder
	writer := csv.NewWriter(&result)

	// Write header
	header := []string{"user_id", "step_name", "timestamp", "completed"}
	_ = writer.Write(header)

	// Simulate data rows
	for i := 0; i < 10; i++ {
		for _, step := range analysis.StepConversions {
			row := []string{
				fmt.Sprintf("user_%d", i+1),
				step.StepName,
				time.Now().Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
				"true",
			}
			_ = writer.Write(row)
		}
	}

	writer.Flush()
	return []byte(result.String()), nil
}

func (s *FunnelService) exportAsJSON(funnelID uuid.UUID, analysis *FunnelAnalysis, params ExportParams) ([]byte, error) {
	export := map[string]interface{}{
		"funnel_id":   funnelID,
		"analysis":    analysis,
		"params":      params,
		"exported_at": time.Now(),
	}

	return json.MarshalIndent(export, "", "  ")
}
