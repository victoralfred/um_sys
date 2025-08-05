package feature

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type FlagType string

const (
	FlagTypeBoolean    FlagType = "boolean"
	FlagTypeString     FlagType = "string"
	FlagTypeNumber     FlagType = "number"
	FlagTypePercentage FlagType = "percentage"
	FlagTypeJSON       FlagType = "json"
)

type TargetingType string

const (
	TargetingTypeAll         TargetingType = "all"
	TargetingTypeUser        TargetingType = "user"
	TargetingTypeGroup       TargetingType = "group"
	TargetingTypePercentage  TargetingType = "percentage"
	TargetingTypeCustom      TargetingType = "custom"
	TargetingTypeEnvironment TargetingType = "environment"
)

type Flag struct {
	ID           uuid.UUID              `json:"id"`
	Key          string                 `json:"key"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         FlagType               `json:"type"`
	DefaultValue interface{}            `json:"default_value"`
	IsEnabled    bool                   `json:"is_enabled"`
	Rules        []Rule                 `json:"rules"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CreatedBy    uuid.UUID              `json:"created_by"`
	UpdatedBy    uuid.UUID              `json:"updated_by"`
}

type Rule struct {
	ID         uuid.UUID   `json:"id"`
	Priority   int         `json:"priority"`
	Name       string      `json:"name"`
	Conditions []Condition `json:"conditions"`
	Value      interface{} `json:"value"`
	Percentage int         `json:"percentage,omitempty"`
	IsEnabled  bool        `json:"is_enabled"`
}

type Condition struct {
	Type     string      `json:"type"`
	Property string      `json:"property"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type Evaluation struct {
	FlagKey   string      `json:"flag_key"`
	Value     interface{} `json:"value"`
	Rule      *Rule       `json:"rule,omitempty"`
	IsDefault bool        `json:"is_default"`
	Reason    string      `json:"reason"`
	Timestamp time.Time   `json:"timestamp"`
}

type Context struct {
	UserID      *uuid.UUID             `json:"user_id,omitempty"`
	GroupID     *uuid.UUID             `json:"group_id,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Country     string                 `json:"country,omitempty"`
	Region      string                 `json:"region,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

type Override struct {
	ID        uuid.UUID              `json:"id"`
	FlagID    uuid.UUID              `json:"flag_id"`
	UserID    *uuid.UUID             `json:"user_id,omitempty"`
	GroupID   *uuid.UUID             `json:"group_id,omitempty"`
	Value     interface{}            `json:"value"`
	Reason    string                 `json:"reason"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	CreatedBy uuid.UUID              `json:"created_by"`
}

type Segment struct {
	ID          uuid.UUID              `json:"id"`
	Key         string                 `json:"key"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Rules       []SegmentRule          `json:"rules"`
	UserIDs     []uuid.UUID            `json:"user_ids,omitempty"`
	GroupIDs    []uuid.UUID            `json:"group_ids,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type SegmentRule struct {
	Conditions []Condition `json:"conditions"`
	Match      string      `json:"match"`
}

type Variant struct {
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
	Weight      int         `json:"weight"`
}

type Experiment struct {
	ID          uuid.UUID              `json:"id"`
	Key         string                 `json:"key"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	FlagID      uuid.UUID              `json:"flag_id"`
	Variants    []Variant              `json:"variants"`
	IsActive    bool                   `json:"is_active"`
	StartDate   time.Time              `json:"start_date"`
	EndDate     *time.Time             `json:"end_date,omitempty"`
	Metrics     []string               `json:"metrics"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type Event struct {
	ID        uuid.UUID              `json:"id"`
	FlagKey   string                 `json:"flag_key"`
	UserID    *uuid.UUID             `json:"user_id,omitempty"`
	GroupID   *uuid.UUID             `json:"group_id,omitempty"`
	Value     interface{}            `json:"value"`
	Context   Context                `json:"context"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type Analytics struct {
	FlagKey       string                 `json:"flag_key"`
	TotalRequests int64                  `json:"total_requests"`
	UniqueUsers   int64                  `json:"unique_users"`
	ValueCounts   map[string]int64       `json:"value_counts"`
	ErrorRate     float64                `json:"error_rate"`
	AvgLatencyMs  float64                `json:"avg_latency_ms"`
	TimeRange     TimeRange              `json:"time_range"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type CreateFlagRequest struct {
	Key          string                 `json:"key"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         FlagType               `json:"type"`
	DefaultValue interface{}            `json:"default_value"`
	IsEnabled    bool                   `json:"is_enabled"`
	Rules        []Rule                 `json:"rules,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateFlagRequest struct {
	Name         *string                `json:"name,omitempty"`
	Description  *string                `json:"description,omitempty"`
	DefaultValue interface{}            `json:"default_value,omitempty"`
	IsEnabled    *bool                  `json:"is_enabled,omitempty"`
	Rules        []Rule                 `json:"rules,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type EvaluationRequest struct {
	FlagKey string  `json:"flag_key"`
	Context Context `json:"context"`
}

type BatchEvaluationRequest struct {
	FlagKeys []string `json:"flag_keys"`
	Context  Context  `json:"context"`
}

type BatchEvaluationResponse struct {
	Evaluations map[string]*Evaluation `json:"evaluations"`
}

type FlagState struct {
	Flags map[string]interface{} `json:"flags"`
}

type ChangeLog struct {
	ID        uuid.UUID       `json:"id"`
	FlagID    uuid.UUID       `json:"flag_id"`
	FlagKey   string          `json:"flag_key"`
	Action    string          `json:"action"`
	Before    json.RawMessage `json:"before,omitempty"`
	After     json.RawMessage `json:"after,omitempty"`
	Reason    string          `json:"reason,omitempty"`
	ChangedBy uuid.UUID       `json:"changed_by"`
	ChangedAt time.Time       `json:"changed_at"`
}
