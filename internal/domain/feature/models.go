package feature

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FlagType represents the type of a feature flag value
type FlagType string

const (
	FlagTypeBoolean FlagType = "boolean"
	FlagTypeString  FlagType = "string"
	FlagTypeNumber  FlagType = "number"
	FlagTypeJSON    FlagType = "json"
)

// RolloutType represents the type of rollout strategy
type RolloutType string

const (
	RolloutTypePercentage RolloutType = "percentage"
	RolloutTypeGradual    RolloutType = "gradual"
	RolloutTypeScheduled  RolloutType = "scheduled"
)

// OperatorType represents comparison operators for conditions
type OperatorType string

const (
	OperatorEquals              OperatorType = "equals"
	OperatorNotEquals           OperatorType = "not_equals"
	OperatorGreaterThan         OperatorType = "greater_than"
	OperatorGreaterThanOrEqual  OperatorType = "greater_than_or_equal"
	OperatorLessThan            OperatorType = "less_than"
	OperatorLessThanOrEqual     OperatorType = "less_than_or_equal"
	OperatorIn                  OperatorType = "in"
	OperatorNotIn               OperatorType = "not_in"
	OperatorContains            OperatorType = "contains"
	OperatorNotContains         OperatorType = "not_contains"
	OperatorBefore              OperatorType = "before"
	OperatorAfter               OperatorType = "after"
)

// OverrideType represents the type of override
type OverrideType string

const (
	OverrideTypeUser  OverrideType = "user"
	OverrideTypeGroup OverrideType = "group"
)

// FeatureFlag represents a feature flag configuration
type FeatureFlag struct {
	ID              uuid.UUID        `json:"id"`
	Key             string           `json:"key"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Type            FlagType         `json:"type"`
	DefaultValue    interface{}      `json:"default_value"`
	Enabled         bool             `json:"enabled"`
	Rules           []TargetingRule  `json:"rules,omitempty"`
	RolloutStrategy *RolloutStrategy `json:"rollout_strategy,omitempty"`
	Variants        []Variant        `json:"variants,omitempty"`
	Schedule        *Schedule        `json:"schedule,omitempty"`
	Dependencies    []string         `json:"dependencies,omitempty"`
	Overrides       []Override       `json:"overrides,omitempty"`
	Tags            []string         `json:"tags,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	CreatedBy       uuid.UUID        `json:"created_by,omitempty"`
	UpdatedBy       uuid.UUID        `json:"updated_by,omitempty"`
}

// TargetingRule represents a rule for targeting specific users
type TargetingRule struct {
	ID         uuid.UUID    `json:"id"`
	Priority   int          `json:"priority"`
	Conditions []Condition  `json:"conditions"`
	Value      interface{}  `json:"value"`
	Enabled    bool         `json:"enabled"`
}

// Condition represents a single condition in a targeting rule
type Condition struct {
	Property string       `json:"property"`
	Operator OperatorType `json:"operator"`
	Value    interface{}  `json:"value"`
}

// RolloutStrategy represents the rollout strategy for a flag
type RolloutStrategy struct {
	Type        RolloutType `json:"type"`
	Percentage  int         `json:"percentage,omitempty"`
	Sticky      bool        `json:"sticky"`
	BucketBy    string      `json:"bucket_by,omitempty"` // Property to bucket by
	StartDate   *time.Time  `json:"start_date,omitempty"`
	EndDate     *time.Time  `json:"end_date,omitempty"`
	Increments  []Increment `json:"increments,omitempty"` // For gradual rollout
}

// Increment represents a rollout increment for gradual rollouts
type Increment struct {
	Date       time.Time `json:"date"`
	Percentage int       `json:"percentage"`
}

// Variant represents a variant in a multi-variant flag
type Variant struct {
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Value       interface{} `json:"value"`
	Weight      int         `json:"weight"` // Percentage weight (0-100)
}

// Schedule represents a time-based schedule for a flag
type Schedule struct {
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Timezone   string    `json:"timezone"`
	Recurring  bool      `json:"recurring"`
	DaysOfWeek []int     `json:"days_of_week,omitempty"` // 0=Sunday, 6=Saturday
	TimeOfDay  *TimeSpan `json:"time_of_day,omitempty"`
}

// TimeSpan represents a time range within a day
type TimeSpan struct {
	StartHour   int `json:"start_hour"`
	StartMinute int `json:"start_minute"`
	EndHour     int `json:"end_hour"`
	EndMinute   int `json:"end_minute"`
}

// Override represents a user or group override
type Override struct {
	ID     uuid.UUID    `json:"id"`
	Type   OverrideType `json:"type"`
	Target string       `json:"target"` // User ID or Group ID
	Value  interface{}  `json:"value"`
}

// EvaluationContext represents the context for evaluating a feature flag
type EvaluationContext struct {
	UserID     uuid.UUID              `json:"user_id"`
	GroupIDs   []string               `json:"group_ids,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// EvaluationResult represents the result of evaluating a feature flag
type EvaluationResult struct {
	FlagKey   string      `json:"flag_key"`
	Value     interface{} `json:"value"`
	Reason    string      `json:"reason"`
	Variant   *string     `json:"variant,omitempty"`
	RuleID    *uuid.UUID  `json:"rule_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// FlagEvent represents an event related to a feature flag
type FlagEvent struct {
	ID        uuid.UUID   `json:"id"`
	FlagID    uuid.UUID   `json:"flag_id"`
	FlagKey   string      `json:"flag_key"`
	EventType string      `json:"event_type"` // created, updated, deleted, evaluated
	UserID    *uuid.UUID  `json:"user_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// FlagStats represents statistics for a feature flag
type FlagStats struct {
	FlagID          uuid.UUID            `json:"flag_id"`
	FlagKey         string               `json:"flag_key"`
	TotalEvaluations int64               `json:"total_evaluations"`
	UniqueUsers     int64                `json:"unique_users"`
	VariantCounts   map[string]int64     `json:"variant_counts,omitempty"`
	RuleCounts      map[uuid.UUID]int64  `json:"rule_counts,omitempty"`
	LastEvaluated   *time.Time           `json:"last_evaluated,omitempty"`
	Period          string               `json:"period"`
}

// Hash generates a consistent hash for a given key and salt
func Hash(key, salt string) uint32 {
	h := sha256.New()
	h.Write([]byte(key + salt))
	hashBytes := h.Sum(nil)
	
	// Convert first 4 bytes to uint32
	return uint32(hashBytes[0])<<24 | uint32(hashBytes[1])<<16 | 
		   uint32(hashBytes[2])<<8 | uint32(hashBytes[3])
}

// HashString generates a consistent hash string
func HashString(key, salt string) string {
	h := sha256.New()
	h.Write([]byte(key + salt))
	return hex.EncodeToString(h.Sum(nil))
}

// IsInBucket determines if a user is in a percentage bucket
func IsInBucket(userID uuid.UUID, flagKey string, percentage int) bool {
	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}
	
	hash := Hash(userID.String(), flagKey)
	bucket := hash % 100
	return int(bucket) < percentage
}

// SelectVariant selects a variant based on weights
func SelectVariant(userID uuid.UUID, flagKey string, variants []Variant) *Variant {
	if len(variants) == 0 {
		return nil
	}
	
	// Calculate total weight
	totalWeight := 0
	for _, v := range variants {
		totalWeight += v.Weight
	}
	
	if totalWeight == 0 {
		return &variants[0]
	}
	
	// Generate consistent random number for this user/flag combination
	hash := Hash(userID.String(), flagKey)
	bucket := int(hash % uint32(totalWeight))
	
	// Select variant based on weight
	cumulative := 0
	for _, v := range variants {
		cumulative += v.Weight
		if bucket < cumulative {
			return &v
		}
	}
	
	return &variants[len(variants)-1]
}

// ValidateFlag validates a feature flag configuration
func ValidateFlag(flag *FeatureFlag) error {
	if flag.Key == "" {
		return fmt.Errorf("flag key is required")
	}
	
	if flag.Type == "" {
		return fmt.Errorf("flag type is required")
	}
	
	// Validate variants weights sum to 100 if specified
	if len(flag.Variants) > 0 {
		totalWeight := 0
		for _, v := range flag.Variants {
			totalWeight += v.Weight
		}
		if totalWeight != 100 {
			return fmt.Errorf("variant weights must sum to 100, got %d", totalWeight)
		}
	}
	
	// Validate rollout percentage
	if flag.RolloutStrategy != nil {
		if flag.RolloutStrategy.Percentage < 0 || flag.RolloutStrategy.Percentage > 100 {
			return fmt.Errorf("rollout percentage must be between 0 and 100")
		}
	}
	
	return nil
}