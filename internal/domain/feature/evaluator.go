package feature

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// FeatureFlagEvaluator evaluates feature flags
type FeatureFlagEvaluator struct {
	mu    sync.RWMutex
	flags map[string]*FeatureFlag
}

// NewFeatureFlagEvaluator creates a new feature flag evaluator
func NewFeatureFlagEvaluator() *FeatureFlagEvaluator {
	return &FeatureFlagEvaluator{
		flags: make(map[string]*FeatureFlag),
	}
}

// AddFlag adds a flag to the evaluator
func (e *FeatureFlagEvaluator) AddFlag(flag *FeatureFlag) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := ValidateFlag(flag); err != nil {
		return err
	}

	e.flags[flag.Key] = flag
	return nil
}

// RemoveFlag removes a flag from the evaluator
func (e *FeatureFlagEvaluator) RemoveFlag(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.flags, key)
}

// GetFlag retrieves a flag by key
func (e *FeatureFlagEvaluator) GetFlag(key string) (*FeatureFlag, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	flag, exists := e.flags[key]
	return flag, exists
}

// Evaluate evaluates a feature flag for a given context
func (e *FeatureFlagEvaluator) Evaluate(ctx context.Context, flag *FeatureFlag, evalContext EvaluationContext) *EvaluationResult {
	result := &EvaluationResult{
		FlagKey:   flag.Key,
		Value:     flag.DefaultValue,
		Reason:    "default",
		Timestamp: time.Now(),
	}

	// Check if flag is disabled
	if !flag.Enabled {
		result.Reason = "disabled"
		return result
	}

	// Initialize timestamp if not set
	if evalContext.Timestamp.IsZero() {
		evalContext.Timestamp = time.Now()
	}

	// Check schedule
	if flag.Schedule != nil && !e.isInSchedule(flag.Schedule, evalContext.Timestamp) {
		result.Reason = "outside_schedule"
		return result
	}

	// Check dependencies
	if len(flag.Dependencies) > 0 && !e.checkDependencies(ctx, flag.Dependencies, evalContext) {
		result.Reason = "dependency_not_met"
		return result
	}

	// Check overrides
	if override := e.checkOverrides(flag.Overrides, evalContext); override != nil {
		result.Value = override.Value
		result.Reason = "override"
		return result
	}

	// Evaluate targeting rules
	if len(flag.Rules) > 0 {
		// Sort rules by priority
		sortedRules := make([]TargetingRule, len(flag.Rules))
		copy(sortedRules, flag.Rules)
		sort.Slice(sortedRules, func(i, j int) bool {
			return sortedRules[i].Priority < sortedRules[j].Priority
		})

		for _, rule := range sortedRules {
			if !rule.Enabled {
				continue
			}

			if e.evaluateRule(rule, evalContext) {
				result.Value = rule.Value
				result.Reason = "rule_match"
				result.RuleID = &rule.ID
				return result
			}
		}
	}

	// Check variants
	if len(flag.Variants) > 0 {
		if variant := SelectVariant(evalContext.UserID, flag.Key, flag.Variants); variant != nil {
			result.Value = variant.Value
			result.Variant = &variant.Key
			result.Reason = "variant"
			return result
		}
	}

	// Check rollout strategy
	if flag.RolloutStrategy != nil {
		if e.isInRollout(flag.RolloutStrategy, evalContext, flag.Key) {
			// For boolean flags with rollout, return true if in rollout
			if flag.Type == FlagTypeBoolean {
				result.Value = true
				result.Reason = "rollout"
				return result
			}
		} else if flag.Type == FlagTypeBoolean {
			result.Value = false
			result.Reason = "rollout"
			return result
		}
	}

	return result
}

// evaluateRule evaluates a targeting rule
func (e *FeatureFlagEvaluator) evaluateRule(rule TargetingRule, context EvaluationContext) bool {
	// All conditions must match (AND logic)
	for _, condition := range rule.Conditions {
		if !e.evaluateCondition(condition, context) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition
func (e *FeatureFlagEvaluator) evaluateCondition(condition Condition, context EvaluationContext) bool {
	value := e.getPropertyValue(condition.Property, context)
	if value == nil {
		return false
	}

	switch condition.Operator {
	case OperatorEquals:
		return e.compareEquals(value, condition.Value)
	case OperatorNotEquals:
		return !e.compareEquals(value, condition.Value)
	case OperatorGreaterThan:
		return e.compareNumeric(value, condition.Value, ">")
	case OperatorGreaterThanOrEqual:
		return e.compareNumeric(value, condition.Value, ">=")
	case OperatorLessThan:
		return e.compareNumeric(value, condition.Value, "<")
	case OperatorLessThanOrEqual:
		return e.compareNumeric(value, condition.Value, "<=")
	case OperatorIn:
		return e.compareIn(value, condition.Value)
	case OperatorNotIn:
		return !e.compareIn(value, condition.Value)
	case OperatorContains:
		return e.compareContains(value, condition.Value)
	case OperatorNotContains:
		return !e.compareContains(value, condition.Value)
	case OperatorBefore:
		return e.compareTime(value, condition.Value, "before")
	case OperatorAfter:
		return e.compareTime(value, condition.Value, "after")
	default:
		return false
	}
}

// getPropertyValue retrieves a property value from the context
func (e *FeatureFlagEvaluator) getPropertyValue(property string, context EvaluationContext) interface{} {
	// Check for special properties
	switch property {
	case "user_id":
		return context.UserID.String()
	case "timestamp":
		return context.Timestamp
	}

	// Check properties map
	if context.Properties != nil {
		// Support nested properties with dot notation
		parts := strings.Split(property, ".")
		current := interface{}(context.Properties)

		for _, part := range parts {
			switch v := current.(type) {
			case map[string]interface{}:
				current = v[part]
			default:
				// Try direct property lookup
				return context.Properties[property]
			}
		}
		return current
	}

	return nil
}

// compareEquals compares two values for equality
func (e *FeatureFlagEvaluator) compareEquals(a, b interface{}) bool {
	// Handle different types
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr == bStr
}

// compareNumeric compares two numeric values
func (e *FeatureFlagEvaluator) compareNumeric(a, b interface{}, op string) bool {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)

	if !aOk || !bOk {
		return false
	}

	switch op {
	case ">":
		return aFloat > bFloat
	case ">=":
		return aFloat >= bFloat
	case "<":
		return aFloat < bFloat
	case "<=":
		return aFloat <= bFloat
	default:
		return false
	}
}

// compareIn checks if value is in a list
func (e *FeatureFlagEvaluator) compareIn(value, list interface{}) bool {
	listSlice, ok := toSlice(list)
	if !ok {
		return false
	}

	valueStr := fmt.Sprintf("%v", value)
	for _, item := range listSlice {
		if fmt.Sprintf("%v", item) == valueStr {
			return true
		}
	}
	return false
}

// compareContains checks if value contains substring
func (e *FeatureFlagEvaluator) compareContains(value, substring interface{}) bool {
	valueStr := fmt.Sprintf("%v", value)
	substringStr := fmt.Sprintf("%v", substring)
	return strings.Contains(valueStr, substringStr)
}

// compareTime compares two time values
func (e *FeatureFlagEvaluator) compareTime(a, b interface{}, op string) bool {
	aTime, aOk := toTime(a)
	bTime, bOk := toTime(b)

	if !aOk || !bOk {
		return false
	}

	switch op {
	case "before":
		return aTime.Before(bTime)
	case "after":
		return aTime.After(bTime)
	default:
		return false
	}
}

// isInSchedule checks if current time is within schedule
func (e *FeatureFlagEvaluator) isInSchedule(schedule *Schedule, currentTime time.Time) bool {
	// Check basic time range
	if currentTime.Before(schedule.StartTime) || currentTime.After(schedule.EndTime) {
		return false
	}

	// Check days of week if specified
	if len(schedule.DaysOfWeek) > 0 {
		currentDay := int(currentTime.Weekday())
		dayFound := false
		for _, day := range schedule.DaysOfWeek {
			if day == currentDay {
				dayFound = true
				break
			}
		}
		if !dayFound {
			return false
		}
	}

	// Check time of day if specified
	if schedule.TimeOfDay != nil {
		currentMinutes := currentTime.Hour()*60 + currentTime.Minute()
		startMinutes := schedule.TimeOfDay.StartHour*60 + schedule.TimeOfDay.StartMinute
		endMinutes := schedule.TimeOfDay.EndHour*60 + schedule.TimeOfDay.EndMinute

		if currentMinutes < startMinutes || currentMinutes > endMinutes {
			return false
		}
	}

	return true
}

// checkDependencies checks if all dependencies are met
func (e *FeatureFlagEvaluator) checkDependencies(ctx context.Context, dependencies []string, evalContext EvaluationContext) bool {
	for _, dep := range dependencies {
		depFlag, exists := e.GetFlag(dep)
		if !exists {
			return false
		}

		result := e.Evaluate(ctx, depFlag, evalContext)
		// For boolean dependencies, check if true
		if depFlag.Type == FlagTypeBoolean {
			if val, ok := result.Value.(bool); !ok || !val {
				return false
			}
		}
	}
	return true
}

// checkOverrides checks for user or group overrides
func (e *FeatureFlagEvaluator) checkOverrides(overrides []Override, context EvaluationContext) *Override {
	for _, override := range overrides {
		switch override.Type {
		case OverrideTypeUser:
			if override.Target == context.UserID.String() {
				return &override
			}
			// Also check if user.id property matches
			if context.Properties != nil {
				if userID, ok := context.Properties["user.id"].(string); ok && userID == override.Target {
					return &override
				}
			}
		case OverrideTypeGroup:
			for _, groupID := range context.GroupIDs {
				if override.Target == groupID {
					return &override
				}
			}
		}
	}
	return nil
}

// isInRollout checks if user is in rollout
func (e *FeatureFlagEvaluator) isInRollout(strategy *RolloutStrategy, context EvaluationContext, flagKey string) bool {
	switch strategy.Type {
	case RolloutTypePercentage:
		if strategy.Sticky {
			return IsInBucket(context.UserID, flagKey, strategy.Percentage)
		}
		// For non-sticky, use timestamp-based randomization
		hash := Hash(context.UserID.String()+context.Timestamp.String(), flagKey)
		return int(hash%100) < strategy.Percentage

	case RolloutTypeScheduled:
		if strategy.StartDate != nil && context.Timestamp.Before(*strategy.StartDate) {
			return false
		}
		if strategy.EndDate != nil && context.Timestamp.After(*strategy.EndDate) {
			return false
		}
		return true

	case RolloutTypeGradual:
		// Find the applicable increment based on current time
		currentPercentage := 0
		for _, increment := range strategy.Increments {
			if context.Timestamp.After(increment.Date) || context.Timestamp.Equal(increment.Date) {
				currentPercentage = increment.Percentage
			}
		}
		return IsInBucket(context.UserID, flagKey, currentPercentage)

	default:
		return false
	}
}

// Helper functions

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint64:
		return float64(val), true
	case uint32:
		return float64(val), true
	default:
		return 0, false
	}
}

func toSlice(v interface{}) ([]interface{}, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}

	slice := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		slice[i] = rv.Index(i).Interface()
	}
	return slice, true
}

func toTime(v interface{}) (time.Time, bool) {
	switch val := v.(type) {
	case time.Time:
		return val, true
	case *time.Time:
		if val != nil {
			return *val, true
		}
	case string:
		t, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
