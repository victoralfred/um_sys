package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/victoralfred/um_sys/internal/domain/feature"
)

// ExperimentVariant represents a variant in an experiment
type ExperimentVariant struct {
	Name   string      `json:"name"`
	Value  interface{} `json:"value"`
	Weight int         `json:"weight"`
}

// FeatureFlagService handles feature flag operations
type FeatureFlagService struct {
	evaluator *feature.FeatureFlagEvaluator
	storage   sync.Map // Simple in-memory storage for testing
	history   sync.Map // Track flag history
	overrides sync.Map // Track overrides
}

// NewFeatureFlagService creates a new feature flag service
func NewFeatureFlagService(flagRepo interface{}, evaluator interface{}) *FeatureFlagService {
	return &FeatureFlagService{
		evaluator: feature.NewFeatureFlagEvaluator(),
	}
}

// CreateFlag creates a new feature flag
func (s *FeatureFlagService) CreateFlag(ctx context.Context, key, name, description string, defaultValue interface{}) (*feature.FeatureFlag, error) {
	flag := &feature.FeatureFlag{
		ID:           uuid.New(),
		Key:          key,
		Name:         name,
		Description:  description,
		Type:         s.detectFlagType(defaultValue),
		DefaultValue: defaultValue,
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := feature.ValidateFlag(flag); err != nil {
		return nil, err
	}

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	// Record creation in history
	s.recordHistory(flag.Key, "created", nil, flag)

	return flag, nil
}

// CreateStringFlag creates a string flag
func (s *FeatureFlagService) CreateStringFlag(ctx context.Context, key, name, description string, defaultValue string) (*feature.FeatureFlag, error) {
	return s.CreateFlag(ctx, key, name, description, defaultValue)
}

// CreateJSONFlag creates a JSON flag
func (s *FeatureFlagService) CreateJSONFlag(ctx context.Context, key, name, description string, defaultValue interface{}) (*feature.FeatureFlag, error) {
	flag := &feature.FeatureFlag{
		ID:           uuid.New(),
		Key:          key,
		Name:         name,
		Description:  description,
		Type:         feature.FlagTypeJSON,
		DefaultValue: defaultValue,
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := feature.ValidateFlag(flag); err != nil {
		return nil, err
	}

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	return flag, nil
}

// GetFlag retrieves a flag by key
func (s *FeatureFlagService) GetFlag(ctx context.Context, key string) (*feature.FeatureFlag, error) {
	if val, ok := s.storage.Load(key); ok {
		return val.(*feature.FeatureFlag), nil
	}
	return nil, errors.New("flag not found")
}

// UpdateFlag updates an existing flag
func (s *FeatureFlagService) UpdateFlag(ctx context.Context, key, name, description string, defaultValue interface{}) error {
	val, ok := s.storage.Load(key)
	if !ok {
		return errors.New("flag not found")
	}

	flag := val.(*feature.FeatureFlag)
	oldFlag := *flag // Copy for history

	flag.Name = name
	flag.Description = description
	flag.DefaultValue = defaultValue
	flag.UpdatedAt = time.Now()

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	// Record update in history
	s.recordHistory(key, "updated", &oldFlag, flag)

	return nil
}

// DeleteFlag deletes a flag
func (s *FeatureFlagService) DeleteFlag(ctx context.Context, key string) error {
	val, ok := s.storage.Load(key)
	if !ok {
		return errors.New("flag not found")
	}

	flag := val.(*feature.FeatureFlag)
	s.storage.Delete(key)
	s.evaluator.RemoveFlag(key)

	// Record deletion in history
	s.recordHistory(key, "deleted", flag, nil)

	return nil
}

// DisableFlag disables a flag
func (s *FeatureFlagService) DisableFlag(ctx context.Context, key string) error {
	val, ok := s.storage.Load(key)
	if !ok {
		return errors.New("flag not found")
	}

	flag := val.(*feature.FeatureFlag)
	flag.Enabled = false
	flag.UpdatedAt = time.Now()

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	return nil
}

// EvaluateForUser evaluates a flag for a specific user
func (s *FeatureFlagService) EvaluateForUser(ctx context.Context, key string, userID uuid.UUID, properties map[string]interface{}) (*feature.EvaluationResult, error) {
	flag, err := s.GetFlag(ctx, key)
	if err != nil {
		return nil, err
	}

	evalContext := feature.EvaluationContext{
		UserID:     userID,
		Properties: properties,
		Timestamp:  time.Now(),
	}

	// Check for overrides first
	overrideKey := fmt.Sprintf("%s:%s", key, userID.String())
	if val, ok := s.overrides.Load(overrideKey); ok {
		override := val.(*Override)
		return &feature.EvaluationResult{
			FlagKey:   key,
			Value:     override.Value,
			Reason:    "override",
			Timestamp: time.Now(),
		}, nil
	}

	// Check if flag is disabled
	if !flag.Enabled {
		return &feature.EvaluationResult{
			FlagKey:   key,
			Value:     flag.DefaultValue,
			Reason:    "flag_disabled",
			Timestamp: time.Now(),
		}, nil
	}

	result := s.evaluator.Evaluate(ctx, flag, evalContext)
	return result, nil
}

// EvaluateForUserInEnvironment evaluates a flag for a user in a specific environment
func (s *FeatureFlagService) EvaluateForUserInEnvironment(ctx context.Context, key string, userID uuid.UUID, environment string, properties map[string]interface{}) (*feature.EvaluationResult, error) {
	if properties == nil {
		properties = make(map[string]interface{})
	}
	properties["environment"] = environment

	// Check if environment is enabled
	envKey := fmt.Sprintf("%s:env:%s", key, environment)
	if val, ok := s.storage.Load(envKey); ok {
		if enabled := val.(bool); enabled {
			return &feature.EvaluationResult{
				FlagKey:   key,
				Value:     true,
				Reason:    "environment",
				Timestamp: time.Now(),
			}, nil
		}
	}

	return s.EvaluateForUser(ctx, key, userID, properties)
}

// EvaluateAll evaluates multiple flags
func (s *FeatureFlagService) EvaluateAll(ctx context.Context, keys []string, userID uuid.UUID, properties map[string]interface{}) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	for _, key := range keys {
		result, err := s.EvaluateForUser(ctx, key, userID, properties)
		if err != nil {
			continue // Skip flags that don't exist
		}
		results[key] = result.Value
	}

	return results, nil
}

// AddUserToFlag adds a user to a flag's targeted users
func (s *FeatureFlagService) AddUserToFlag(ctx context.Context, key string, userID uuid.UUID) error {
	val, ok := s.storage.Load(key)
	if !ok {
		return errors.New("flag not found")
	}

	flag := val.(*feature.FeatureFlag)

	// Add a rule for this specific user
	rule := feature.TargetingRule{
		ID:       uuid.New(),
		Priority: 0, // Highest priority
		Conditions: []feature.Condition{
			{
				Property: "user_id",
				Operator: feature.OperatorEquals,
				Value:    userID.String(),
			},
		},
		Value:   true,
		Enabled: true,
	}

	flag.Rules = append([]feature.TargetingRule{rule}, flag.Rules...)
	flag.UpdatedAt = time.Now()

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	return nil
}

// SetPercentageRollout sets percentage rollout for a flag
func (s *FeatureFlagService) SetPercentageRollout(ctx context.Context, key string, percentage int) error {
	val, ok := s.storage.Load(key)
	if !ok {
		return errors.New("flag not found")
	}

	flag := val.(*feature.FeatureFlag)

	flag.RolloutStrategy = &feature.RolloutStrategy{
		Type:       feature.RolloutTypePercentage,
		Percentage: percentage,
		Sticky:     true, // Ensure consistent results
	}
	flag.UpdatedAt = time.Now()

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	return nil
}

// AddPropertyRule adds a property-based rule to a flag
func (s *FeatureFlagService) AddPropertyRule(ctx context.Context, key, property, operator string, value interface{}, result interface{}) error {
	val, ok := s.storage.Load(key)
	if !ok {
		return errors.New("flag not found")
	}

	flag := val.(*feature.FeatureFlag)

	// Convert operator string to OperatorType
	var op feature.OperatorType
	switch operator {
	case "equals":
		op = feature.OperatorEquals
	case "not_equals":
		op = feature.OperatorNotEquals
	case "greater_than":
		op = feature.OperatorGreaterThan
	case "less_than":
		op = feature.OperatorLessThan
	case "in":
		op = feature.OperatorIn
	case "contains":
		op = feature.OperatorContains
	default:
		return fmt.Errorf("unknown operator: %s", operator)
	}

	rule := feature.TargetingRule{
		ID:       uuid.New(),
		Priority: len(flag.Rules) + 1,
		Conditions: []feature.Condition{
			{
				Property: property,
				Operator: op,
				Value:    value,
			},
		},
		Value:   result,
		Enabled: true,
	}

	flag.Rules = append(flag.Rules, rule)
	flag.UpdatedAt = time.Now()

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	return nil
}

// AddVariant adds a variant to a flag
func (s *FeatureFlagService) AddVariant(ctx context.Context, key, variantKey string, value interface{}, weight int) error {
	val, ok := s.storage.Load(key)
	if !ok {
		return errors.New("flag not found")
	}

	flag := val.(*feature.FeatureFlag)

	variant := feature.Variant{
		Key:    variantKey,
		Name:   variantKey,
		Value:  value,
		Weight: weight,
	}

	flag.Variants = append(flag.Variants, variant)
	flag.UpdatedAt = time.Now()

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	return nil
}

// CreateOverride creates an override for a specific user
func (s *FeatureFlagService) CreateOverride(ctx context.Context, key string, userID uuid.UUID, value interface{}, reason string) error {
	overrideKey := fmt.Sprintf("%s:%s", key, userID.String())
	override := &Override{
		UserID: userID,
		Value:  value,
		Reason: reason,
	}
	s.overrides.Store(overrideKey, override)
	return nil
}

// EnableForEnvironment enables a flag for a specific environment
func (s *FeatureFlagService) EnableForEnvironment(ctx context.Context, key, environment string) error {
	envKey := fmt.Sprintf("%s:env:%s", key, environment)
	s.storage.Store(envKey, true)
	return nil
}

// ScheduleFlag schedules a flag activation
func (s *FeatureFlagService) ScheduleFlag(ctx context.Context, key string, startTime, endTime time.Time) error {
	val, ok := s.storage.Load(key)
	if !ok {
		return errors.New("flag not found")
	}

	flag := val.(*feature.FeatureFlag)

	flag.Schedule = &feature.Schedule{
		StartTime: startTime,
		EndTime:   endTime,
		Timezone:  "UTC",
	}
	flag.UpdatedAt = time.Now()

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	return nil
}

// CreateExperiment creates an A/B test experiment
func (s *FeatureFlagService) CreateExperiment(ctx context.Context, key string, variants []ExperimentVariant) error {
	val, ok := s.storage.Load(key)
	if !ok {
		return errors.New("flag not found")
	}

	flag := val.(*feature.FeatureFlag)

	// Convert experiment variants to flag variants
	flag.Variants = nil
	for _, v := range variants {
		flag.Variants = append(flag.Variants, feature.Variant{
			Key:    v.Name,
			Name:   v.Name,
			Value:  v.Value,
			Weight: v.Weight,
		})
	}

	flag.UpdatedAt = time.Now()

	s.storage.Store(key, flag)
	_ = s.evaluator.AddFlag(flag)

	return nil
}

// GetFlagHistory retrieves the change history for a flag
func (s *FeatureFlagService) GetFlagHistory(ctx context.Context, key string) ([]HistoryEntry, error) {
	val, ok := s.history.Load(key)
	if !ok {
		return []HistoryEntry{}, nil
	}
	return val.([]HistoryEntry), nil
}

// Helper types and methods

type Override struct {
	UserID uuid.UUID
	Value  interface{}
	Reason string
}

type HistoryEntry struct {
	Action    string
	Before    interface{}
	After     interface{}
	Timestamp time.Time
}

func (s *FeatureFlagService) detectFlagType(value interface{}) feature.FlagType {
	switch value.(type) {
	case bool:
		return feature.FlagTypeBoolean
	case string:
		return feature.FlagTypeString
	case int, int32, int64, float32, float64:
		return feature.FlagTypeNumber
	default:
		return feature.FlagTypeJSON
	}
}

func (s *FeatureFlagService) recordHistory(key, action string, before, after interface{}) {
	entry := HistoryEntry{
		Action:    action,
		Before:    before,
		After:     after,
		Timestamp: time.Now(),
	}

	var history []HistoryEntry
	if val, ok := s.history.Load(key); ok {
		history = val.([]HistoryEntry)
	}
	history = append(history, entry)
	s.history.Store(key, history)
}
