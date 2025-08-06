package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/victoralfred/um_sys/internal/domain/analytics"
)

func TestEventTypeRegistry(t *testing.T) {
	ctx := context.Background()

	t.Run("Register custom event type", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		schema := &EventTypeSchema{
			Name:     "purchase_completed",
			Category: "e-commerce",
			RequiredFields: []FieldDefinition{
				{Name: "order_id", Type: "string", Required: true},
				{Name: "total_amount", Type: "number", Required: true},
				{Name: "currency", Type: "string", Required: true},
			},
			OptionalFields: []FieldDefinition{
				{Name: "discount_code", Type: "string", Required: false},
				{Name: "items_count", Type: "integer", Required: false},
			},
		}

		err := registry.Register("purchase_completed", schema)
		assert.NoError(t, err)

		// Verify registration
		registeredSchema, err := registry.GetSchema("purchase_completed")
		assert.NoError(t, err)
		assert.Equal(t, schema.Name, registeredSchema.Name)
		assert.Equal(t, schema.Category, registeredSchema.Category)
		assert.Len(t, registeredSchema.RequiredFields, 3)
	})

	t.Run("Validate event against schema", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		// Register schema
		schema := &EventTypeSchema{
			Name:     "user_subscription",
			Category: "billing",
			RequiredFields: []FieldDefinition{
				{Name: "plan_id", Type: "string", Required: true},
				{Name: "price", Type: "number", Required: true},
			},
		}

		err := registry.Register("user_subscription", schema)
		require.NoError(t, err)

		// Valid event
		validEvent := &analytics.Event{
			ID:   uuid.New(),
			Type: "user_subscription",
			Properties: map[string]interface{}{
				"plan_id": "pro-monthly",
				"price":   29.99,
			},
		}

		err = registry.Validate(validEvent)
		assert.NoError(t, err)

		// Invalid event - missing required field
		invalidEvent := &analytics.Event{
			ID:   uuid.New(),
			Type: "user_subscription",
			Properties: map[string]interface{}{
				"plan_id": "pro-monthly",
				// missing price
			},
		}

		err = registry.Validate(invalidEvent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required field: price")
	})

	t.Run("Type validation", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		schema := &EventTypeSchema{
			Name:     "api_request",
			Category: "system",
			RequiredFields: []FieldDefinition{
				{Name: "endpoint", Type: "string", Required: true},
				{Name: "response_time", Type: "number", Required: true},
				{Name: "status_code", Type: "integer", Required: true},
			},
		}

		err := registry.Register("api_request", schema)
		require.NoError(t, err)

		// Invalid type for field
		event := &analytics.Event{
			ID:   uuid.New(),
			Type: "api_request",
			Properties: map[string]interface{}{
				"endpoint":      "/api/users",
				"response_time": "fast", // should be number
				"status_code":   200,
			},
		}

		err = registry.Validate(event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type for field: response_time")
	})

	t.Run("List registered event types", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		// Register multiple event types
		schemas := []EventTypeSchema{
			{Name: "user_login", Category: "auth"},
			{Name: "page_view", Category: "engagement"},
			{Name: "error_occurred", Category: "system"},
		}

		for _, schema := range schemas {
			s := schema // capture range variable
			err := registry.Register(s.Name, &s)
			require.NoError(t, err)
		}

		// List all types
		types := registry.ListTypes()
		assert.Len(t, types, 3)
		assert.Contains(t, types, "user_login")
		assert.Contains(t, types, "page_view")
		assert.Contains(t, types, "error_occurred")
	})

	t.Run("Prevent duplicate registration", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		schema := &EventTypeSchema{
			Name:     "duplicate_event",
			Category: "test",
		}

		// First registration should succeed
		err := registry.Register("duplicate_event", schema)
		assert.NoError(t, err)

		// Second registration should fail
		err = registry.Register("duplicate_event", schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event type already registered")
	})

	t.Run("Custom validators", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		schema := &EventTypeSchema{
			Name:     "payment_processed",
			Category: "billing",
			RequiredFields: []FieldDefinition{
				{Name: "amount", Type: "number", Required: true},
				{Name: "currency", Type: "string", Required: true},
			},
			Validators: []Validator{
				&RangeValidator{Field: "amount", Min: 0.01, Max: 1000000},
				&RegexValidator{Field: "currency", Pattern: "^[A-Z]{3}$"},
			},
		}

		err := registry.Register("payment_processed", schema)
		require.NoError(t, err)

		// Valid event
		validEvent := &analytics.Event{
			ID:   uuid.New(),
			Type: "payment_processed",
			Properties: map[string]interface{}{
				"amount":   99.99,
				"currency": "USD",
			},
		}

		err = registry.Validate(validEvent)
		assert.NoError(t, err)

		// Invalid amount (out of range)
		invalidAmount := &analytics.Event{
			ID:   uuid.New(),
			Type: "payment_processed",
			Properties: map[string]interface{}{
				"amount":   -10.00,
				"currency": "USD",
			},
		}

		err = registry.Validate(invalidAmount)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value out of range")

		// Invalid currency format
		invalidCurrency := &analytics.Event{
			ID:   uuid.New(),
			Type: "payment_processed",
			Properties: map[string]interface{}{
				"amount":   99.99,
				"currency": "dollars",
			},
		}

		err = registry.Validate(invalidCurrency)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid format")
	})

	t.Run("Inheritance from base schemas", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		// Register base schema
		baseSchema := &EventTypeSchema{
			Name:     "base_interaction",
			Category: "interaction",
			RequiredFields: []FieldDefinition{
				{Name: "element_id", Type: "string", Required: true},
				{Name: "page_url", Type: "string", Required: true},
			},
		}

		err := registry.RegisterBase("base_interaction", baseSchema)
		require.NoError(t, err)

		// Register derived schema
		clickSchema := &EventTypeSchema{
			Name:     "button_click",
			Category: "interaction",
			Extends:  "base_interaction",
			RequiredFields: []FieldDefinition{
				{Name: "button_text", Type: "string", Required: true},
			},
		}

		err = registry.Register("button_click", clickSchema)
		require.NoError(t, err)

		// Validate event with both base and derived fields
		event := &analytics.Event{
			ID:   uuid.New(),
			Type: "button_click",
			Properties: map[string]interface{}{
				"element_id":  "submit-btn",
				"page_url":    "/checkout",
				"button_text": "Complete Order",
			},
		}

		err = registry.Validate(event)
		assert.NoError(t, err)

		// Missing base field should fail
		invalidEvent := &analytics.Event{
			ID:   uuid.New(),
			Type: "button_click",
			Properties: map[string]interface{}{
				"button_text": "Complete Order",
				// missing base fields
			},
		}

		err = registry.Validate(invalidEvent)
		assert.Error(t, err)
	})

	t.Run("Dynamic field computation", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		schema := &EventTypeSchema{
			Name:     "product_view",
			Category: "e-commerce",
			RequiredFields: []FieldDefinition{
				{Name: "product_id", Type: "string", Required: true},
				{Name: "price", Type: "number", Required: true},
			},
			ComputedFields: []ComputedField{
				{
					Name: "price_category",
					Type: "string",
					Formula: func(props map[string]interface{}) interface{} {
						price := props["price"].(float64)
						if price < 50 {
							return "budget"
						} else if price < 200 {
							return "mid-range"
						}
						return "premium"
					},
				},
			},
		}

		err := registry.Register("product_view", schema)
		require.NoError(t, err)

		// Process event with computed fields
		event := &analytics.Event{
			ID:   uuid.New(),
			Type: "product_view",
			Properties: map[string]interface{}{
				"product_id": "SKU-12345",
				"price":      75.00,
			},
		}

		processedEvent, err := registry.Process(ctx, event)
		assert.NoError(t, err)
		assert.Equal(t, "mid-range", processedEvent.Properties["price_category"])
	})

	t.Run("Schema versioning", func(t *testing.T) {
		registry := NewEventTypeRegistry()

		// Register v1 schema
		schemaV1 := &EventTypeSchema{
			Name:     "user_profile",
			Category: "user",
			Version:  1,
			RequiredFields: []FieldDefinition{
				{Name: "username", Type: "string", Required: true},
			},
		}

		err := registry.RegisterVersion("user_profile", 1, schemaV1)
		require.NoError(t, err)

		// Register v2 schema with additional fields
		schemaV2 := &EventTypeSchema{
			Name:     "user_profile",
			Category: "user",
			Version:  2,
			RequiredFields: []FieldDefinition{
				{Name: "username", Type: "string", Required: true},
				{Name: "email", Type: "string", Required: true},
			},
			MigrationRules: []MigrationRule{
				{
					FromVersion: 1,
					ToVersion:   2,
					Transform: func(props map[string]interface{}) map[string]interface{} {
						// Add default email if missing
						if _, ok := props["email"]; !ok {
							props["email"] = props["username"].(string) + "@example.com"
						}
						return props
					},
				},
			},
		}

		err = registry.RegisterVersion("user_profile", 2, schemaV2)
		require.NoError(t, err)

		// Validate v1 event (should auto-migrate to v2)
		v1Event := &analytics.Event{
			ID:      uuid.New(),
			Type:    "user_profile",
			Version: 1,
			Properties: map[string]interface{}{
				"username": "johndoe",
			},
		}

		migratedEvent, err := registry.Migrate(v1Event, 2)
		assert.NoError(t, err)
		assert.Equal(t, 2, migratedEvent.Version)
		assert.Equal(t, "johndoe@example.com", migratedEvent.Properties["email"])
	})
}
