package services

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sync"

	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

// EventTypeRegistry manages custom event type schemas and validation
type EventTypeRegistry struct {
	mu          sync.RWMutex
	schemas     map[string]*EventTypeSchema
	baseSchemas map[string]*EventTypeSchema
	versions    map[string]map[int]*EventTypeSchema
}

// EventTypeSchema defines the structure and validation rules for an event type
type EventTypeSchema struct {
	Name           string
	Category       string
	Version        int
	Extends        string // Base schema to inherit from
	RequiredFields []FieldDefinition
	OptionalFields []FieldDefinition
	Validators     []Validator
	ComputedFields []ComputedField
	MigrationRules []MigrationRule
}

// FieldDefinition defines a field in the event schema
type FieldDefinition struct {
	Name        string
	Type        string // string, number, integer, boolean, object, array
	Required    bool
	Description string
	Default     interface{}
}

// Validator interface for custom validation logic
type Validator interface {
	Validate(value interface{}) error
	GetField() string
}

// RangeValidator validates numeric values are within a range
type RangeValidator struct {
	Field string
	Min   float64
	Max   float64
}

func (v *RangeValidator) Validate(value interface{}) error {
	var num float64
	switch val := value.(type) {
	case float64:
		num = val
	case int:
		num = float64(val)
	case int64:
		num = float64(val)
	default:
		return fmt.Errorf("value is not a number")
	}

	if num < v.Min || num > v.Max {
		return fmt.Errorf("value out of range [%f, %f]: %f", v.Min, v.Max, num)
	}
	return nil
}

func (v *RangeValidator) GetField() string {
	return v.Field
}

// RegexValidator validates string values match a pattern
type RegexValidator struct {
	Field   string
	Pattern string
	regex   *regexp.Regexp
}

func (v *RegexValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value is not a string")
	}

	if v.regex == nil {
		var err error
		v.regex, err = regexp.Compile(v.Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	if !v.regex.MatchString(str) {
		return fmt.Errorf("invalid format for pattern %s", v.Pattern)
	}
	return nil
}

func (v *RegexValidator) GetField() string {
	return v.Field
}

// ComputedField defines a dynamically computed field
type ComputedField struct {
	Name    string
	Type    string
	Formula func(props map[string]interface{}) interface{}
}

// MigrationRule defines how to migrate from one schema version to another
type MigrationRule struct {
	FromVersion int
	ToVersion   int
	Transform   func(props map[string]interface{}) map[string]interface{}
}

// NewEventTypeRegistry creates a new event type registry
func NewEventTypeRegistry() *EventTypeRegistry {
	return &EventTypeRegistry{
		schemas:     make(map[string]*EventTypeSchema),
		baseSchemas: make(map[string]*EventTypeSchema),
		versions:    make(map[string]map[int]*EventTypeSchema),
	}
}

// Register registers a new event type schema
func (r *EventTypeRegistry) Register(eventType string, schema *EventTypeSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.schemas[eventType]; exists {
		return errors.New("event type already registered")
	}

	// If schema extends a base schema, merge the fields
	if schema.Extends != "" {
		base, exists := r.baseSchemas[schema.Extends]
		if !exists {
			return fmt.Errorf("base schema %s not found", schema.Extends)
		}
		schema = r.mergeSchemas(base, schema)
	}

	r.schemas[eventType] = schema
	return nil
}

// RegisterBase registers a base schema for inheritance
func (r *EventTypeRegistry) RegisterBase(name string, schema *EventTypeSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.baseSchemas[name]; exists {
		return errors.New("base schema already registered")
	}

	r.baseSchemas[name] = schema
	return nil
}

// RegisterVersion registers a versioned schema
func (r *EventTypeRegistry) RegisterVersion(eventType string, version int, schema *EventTypeSchema) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.versions[eventType] == nil {
		r.versions[eventType] = make(map[int]*EventTypeSchema)
	}

	if _, exists := r.versions[eventType][version]; exists {
		return fmt.Errorf("version %d already registered for event type %s", version, eventType)
	}

	schema.Version = version
	r.versions[eventType][version] = schema

	// Update main schema to latest version
	if currentSchema, exists := r.schemas[eventType]; !exists || currentSchema.Version < version {
		r.schemas[eventType] = schema
	}

	return nil
}

// Validate validates an event against its schema
func (r *EventTypeRegistry) Validate(event *analytics.Event) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[string(event.Type)]
	if !exists {
		// If no schema registered, allow any event (backward compatibility)
		return nil
	}

	// Check required fields
	for _, field := range schema.RequiredFields {
		value, exists := event.Properties[field.Name]
		if !exists || value == nil {
			return fmt.Errorf("missing required field: %s", field.Name)
		}

		// Validate field type
		if err := r.validateFieldType(field, value); err != nil {
			return fmt.Errorf("invalid type for field %s: %w", field.Name, err)
		}
	}

	// Run custom validators
	for _, validator := range schema.Validators {
		fieldName := validator.GetField()
		if value, exists := event.Properties[fieldName]; exists {
			if err := validator.Validate(value); err != nil {
				return fmt.Errorf("validation failed for field %s: %w", fieldName, err)
			}
		}
	}

	return nil
}

// GetSchema retrieves the schema for an event type
func (r *EventTypeRegistry) GetSchema(eventType string) (*EventTypeSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[eventType]
	if !exists {
		return nil, fmt.Errorf("schema not found for event type: %s", eventType)
	}

	return schema, nil
}

// ListTypes returns all registered event types
func (r *EventTypeRegistry) ListTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.schemas))
	for eventType := range r.schemas {
		types = append(types, eventType)
	}
	return types
}

// Process processes an event, applying computed fields
func (r *EventTypeRegistry) Process(ctx context.Context, event *analytics.Event) (*analytics.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[string(event.Type)]
	if !exists {
		return event, nil
	}

	// Apply computed fields
	for _, computed := range schema.ComputedFields {
		if computed.Formula != nil {
			value := computed.Formula(event.Properties)
			if event.Properties == nil {
				event.Properties = make(map[string]interface{})
			}
			event.Properties[computed.Name] = value
		}
	}

	return event, nil
}

// Migrate migrates an event from one version to another
func (r *EventTypeRegistry) Migrate(event *analytics.Event, targetVersion int) (*analytics.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if event.Version == 0 {
		event.Version = 1 // Default version
	}

	if event.Version == targetVersion {
		return event, nil
	}

	versions, exists := r.versions[string(event.Type)]
	if !exists {
		return nil, fmt.Errorf("no versions found for event type: %s", event.Type)
	}

	targetSchema, exists := versions[targetVersion]
	if !exists {
		return nil, fmt.Errorf("version %d not found for event type: %s", targetVersion, event.Type)
	}

	// Apply migration rules
	currentVersion := event.Version
	for currentVersion < targetVersion {
		migrated := false
		for _, rule := range targetSchema.MigrationRules {
			if rule.FromVersion == currentVersion && rule.ToVersion <= targetVersion {
				event.Properties = rule.Transform(event.Properties)
				currentVersion = rule.ToVersion
				migrated = true
				break
			}
		}
		if !migrated {
			// No direct migration path, increment version
			currentVersion++
		}
	}

	event.Version = targetVersion
	return event, nil
}

// validateFieldType validates that a value matches the expected field type
func (r *EventTypeRegistry) validateFieldType(field FieldDefinition, value interface{}) error {
	if value == nil && !field.Required {
		return nil
	}

	switch field.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return errors.New("expected string")
		}
	case "number":
		switch value.(type) {
		case float64, float32, int, int64, int32, int16, int8:
			// Valid number types
		default:
			return errors.New("expected number")
		}
	case "integer":
		switch v := value.(type) {
		case int, int64, int32, int16, int8:
			// Valid integer types
		case float64:
			// Check if it's a whole number
			if v != float64(int64(v)) {
				return errors.New("expected integer")
			}
		default:
			return errors.New("expected integer")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("expected boolean")
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return errors.New("expected object")
		}
	case "array":
		// Check if value is a slice or array
		if reflect.TypeOf(value).Kind() != reflect.Slice && reflect.TypeOf(value).Kind() != reflect.Array {
			return errors.New("expected array")
		}
	default:
		// Unknown type, allow any
		return nil
	}

	return nil
}

// mergeSchemas merges a base schema with a derived schema
func (r *EventTypeRegistry) mergeSchemas(base, derived *EventTypeSchema) *EventTypeSchema {
	merged := &EventTypeSchema{
		Name:           derived.Name,
		Category:       derived.Category,
		Version:        derived.Version,
		RequiredFields: append(base.RequiredFields, derived.RequiredFields...),
		OptionalFields: append(base.OptionalFields, derived.OptionalFields...),
		Validators:     append(base.Validators, derived.Validators...),
		ComputedFields: append(base.ComputedFields, derived.ComputedFields...),
		MigrationRules: derived.MigrationRules,
	}
	return merged
}
