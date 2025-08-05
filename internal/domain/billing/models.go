package billing

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PlanType represents the type of subscription plan
type PlanType string

const (
	PlanTypeFree       PlanType = "free"
	PlanTypeBasic      PlanType = "basic"
	PlanTypePro        PlanType = "pro"
	PlanTypeEnterprise PlanType = "enterprise"
	PlanTypeCustom     PlanType = "custom"
)

// BillingInterval represents the billing interval
type BillingInterval string

const (
	IntervalMonthly   BillingInterval = "monthly"
	IntervalYearly    BillingInterval = "yearly"
	IntervalQuarterly BillingInterval = "quarterly"
	IntervalOneTime   BillingInterval = "one_time"
)

// PaymentStatus represents the payment status
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCanceled  PaymentStatus = "canceled"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// SubscriptionStatus represents the subscription status
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid   SubscriptionStatus = "unpaid"
	SubscriptionStatusPaused   SubscriptionStatus = "paused"
)

// Plan represents a subscription plan
type Plan struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Type            PlanType        `json:"type"`
	Description     string          `json:"description"`
	Price           decimal.Decimal `json:"price"`
	Currency        string          `json:"currency"` // USD, EUR, etc.
	BillingInterval BillingInterval `json:"billing_interval"`
	TrialDays       int             `json:"trial_days"`
	Features        []Feature       `json:"features"`
	Limits          PlanLimits      `json:"limits"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Feature represents a plan feature
type Feature struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Key         string    `json:"key"` // e.g., "api_calls", "storage_gb"
	Value       string    `json:"value"`
	Type        string    `json:"type"` // boolean, numeric, string
}

// PlanLimits represents the limits for a plan
type PlanLimits struct {
	MaxUsers            int `json:"max_users"`
	MaxProjects         int `json:"max_projects"`
	MaxAPICallsPerMonth int `json:"max_api_calls_per_month"`
	MaxStorageGB        int `json:"max_storage_gb"`
	MaxTeamMembers      int `json:"max_team_members"`
}

// Subscription represents a user's subscription
type Subscription struct {
	ID                   uuid.UUID          `json:"id"`
	UserID               uuid.UUID          `json:"user_id"`
	PlanID               uuid.UUID          `json:"plan_id"`
	Status               SubscriptionStatus `json:"status"`
	CurrentPeriodStart   time.Time          `json:"current_period_start"`
	CurrentPeriodEnd     time.Time          `json:"current_period_end"`
	TrialStart           *time.Time         `json:"trial_start,omitempty"`
	TrialEnd             *time.Time         `json:"trial_end,omitempty"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end"`
	StripeSubscriptionID string             `json:"stripe_subscription_id,omitempty"`
	StripeCustomerID     string             `json:"stripe_customer_id,omitempty"`
	Metadata             map[string]string  `json:"metadata,omitempty"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// Payment represents a payment transaction
type Payment struct {
	ID              uuid.UUID         `json:"id"`
	UserID          uuid.UUID         `json:"user_id"`
	SubscriptionID  *uuid.UUID        `json:"subscription_id,omitempty"`
	Amount          decimal.Decimal   `json:"amount"`
	Currency        string            `json:"currency"`
	Status          PaymentStatus     `json:"status"`
	Description     string            `json:"description"`
	PaymentMethod   string            `json:"payment_method"` // card, paypal, bank_transfer
	StripePaymentID string            `json:"stripe_payment_id,omitempty"`
	StripeInvoiceID string            `json:"stripe_invoice_id,omitempty"`
	FailureReason   string            `json:"failure_reason,omitempty"`
	RefundedAmount  decimal.Decimal   `json:"refunded_amount"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	PaidAt          *time.Time        `json:"paid_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// Invoice represents an invoice
type Invoice struct {
	ID              uuid.UUID         `json:"id"`
	UserID          uuid.UUID         `json:"user_id"`
	SubscriptionID  *uuid.UUID        `json:"subscription_id,omitempty"`
	InvoiceNumber   string            `json:"invoice_number"`
	Amount          decimal.Decimal   `json:"amount"`
	Tax             decimal.Decimal   `json:"tax"`
	Total           decimal.Decimal   `json:"total"`
	Currency        string            `json:"currency"`
	Status          string            `json:"status"` // draft, open, paid, void, uncollectible
	DueDate         time.Time         `json:"due_date"`
	PaidAt          *time.Time        `json:"paid_at,omitempty"`
	LineItems       []InvoiceLineItem `json:"line_items"`
	StripeInvoiceID string            `json:"stripe_invoice_id,omitempty"`
	PDFUrl          string            `json:"pdf_url,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// InvoiceLineItem represents a line item on an invoice
type InvoiceLineItem struct {
	ID          uuid.UUID       `json:"id"`
	Description string          `json:"description"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Amount      decimal.Decimal `json:"amount"`
	Tax         decimal.Decimal `json:"tax"`
	Total       decimal.Decimal `json:"total"`
}

// PaymentMethod represents a saved payment method
type PaymentMethod struct {
	ID                    uuid.UUID `json:"id"`
	UserID                uuid.UUID `json:"user_id"`
	Type                  string    `json:"type"` // card, bank_account, paypal
	IsDefault             bool      `json:"is_default"`
	StripePaymentMethodID string    `json:"stripe_payment_method_id,omitempty"`

	// Card details (masked)
	CardBrand    string `json:"card_brand,omitempty"` // visa, mastercard, amex
	CardLast4    string `json:"card_last4,omitempty"`
	CardExpMonth int    `json:"card_exp_month,omitempty"`
	CardExpYear  int    `json:"card_exp_year,omitempty"`

	// Bank account details (masked)
	BankName         string `json:"bank_name,omitempty"`
	BankAccountLast4 string `json:"bank_account_last4,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UsageRecord represents usage tracking for metered billing
type UsageRecord struct {
	ID             uuid.UUID         `json:"id"`
	UserID         uuid.UUID         `json:"user_id"`
	SubscriptionID uuid.UUID         `json:"subscription_id"`
	MetricName     string            `json:"metric_name"` // e.g., "api_calls", "storage_gb"
	Quantity       decimal.Decimal   `json:"quantity"`
	Unit           string            `json:"unit"` // e.g., "calls", "GB", "hours"
	Timestamp      time.Time         `json:"timestamp"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

// Coupon represents a discount coupon
type Coupon struct {
	ID               uuid.UUID       `json:"id"`
	Code             string          `json:"code"`
	Description      string          `json:"description"`
	DiscountType     string          `json:"discount_type"` // percentage, fixed_amount
	DiscountValue    decimal.Decimal `json:"discount_value"`
	Currency         string          `json:"currency,omitempty"`           // For fixed_amount type
	Duration         string          `json:"duration"`                     // once, forever, repeating
	DurationInMonths int             `json:"duration_in_months,omitempty"` // For repeating
	MaxRedemptions   int             `json:"max_redemptions"`
	TimesRedeemed    int             `json:"times_redeemed"`
	ValidFrom        time.Time       `json:"valid_from"`
	ValidUntil       time.Time       `json:"valid_until"`
	IsActive         bool            `json:"is_active"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// CreateSubscriptionRequest represents a request to create a subscription
type CreateSubscriptionRequest struct {
	UserID          uuid.UUID         `json:"user_id"`
	PlanID          uuid.UUID         `json:"plan_id"`
	PaymentMethodID *uuid.UUID        `json:"payment_method_id,omitempty"`
	CouponCode      string            `json:"coupon_code,omitempty"`
	TrialDays       int               `json:"trial_days,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// UpdateSubscriptionRequest represents a request to update a subscription
type UpdateSubscriptionRequest struct {
	SubscriptionID  uuid.UUID         `json:"subscription_id"`
	PlanID          *uuid.UUID        `json:"plan_id,omitempty"`
	PaymentMethodID *uuid.UUID        `json:"payment_method_id,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// CancelSubscriptionRequest represents a request to cancel a subscription
type CancelSubscriptionRequest struct {
	SubscriptionID    uuid.UUID `json:"subscription_id"`
	CancelImmediately bool      `json:"cancel_immediately"`
	Reason            string    `json:"reason,omitempty"`
}

// CreatePaymentRequest represents a request to create a payment
type CreatePaymentRequest struct {
	UserID          uuid.UUID         `json:"user_id"`
	Amount          decimal.Decimal   `json:"amount"`
	Currency        string            `json:"currency"`
	Description     string            `json:"description"`
	PaymentMethodID uuid.UUID         `json:"payment_method_id"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// RefundRequest represents a request to refund a payment
type RefundRequest struct {
	PaymentID uuid.UUID       `json:"payment_id"`
	Amount    decimal.Decimal `json:"amount,omitempty"` // Partial refund if specified
	Reason    string          `json:"reason"`
}
