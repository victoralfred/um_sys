package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PlanRepository defines the interface for plan persistence
type PlanRepository interface {
	// Create creates a new plan
	Create(ctx context.Context, plan *Plan) error

	// GetByID retrieves a plan by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Plan, error)

	// GetByType retrieves plans by type
	GetByType(ctx context.Context, planType PlanType) ([]*Plan, error)

	// Update updates a plan
	Update(ctx context.Context, plan *Plan) error

	// Delete deletes a plan
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all active plans
	List(ctx context.Context) ([]*Plan, error)

	// GetFeatures retrieves features for a plan
	GetFeatures(ctx context.Context, planID uuid.UUID) ([]Feature, error)
}

// SubscriptionRepository defines the interface for subscription persistence
type SubscriptionRepository interface {
	// Create creates a new subscription
	Create(ctx context.Context, subscription *Subscription) error

	// GetByID retrieves a subscription by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Subscription, error)

	// GetByUserID retrieves subscriptions for a user
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Subscription, error)

	// GetActiveByUserID retrieves the active subscription for a user
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error)

	// Update updates a subscription
	Update(ctx context.Context, subscription *Subscription) error

	// Cancel cancels a subscription
	Cancel(ctx context.Context, id uuid.UUID, cancelAtPeriodEnd bool) error

	// GetExpiringSubscriptions retrieves subscriptions expiring within a time range
	GetExpiringSubscriptions(ctx context.Context, before time.Time) ([]*Subscription, error)

	// GetByStripeSubscriptionID retrieves a subscription by Stripe ID
	GetByStripeSubscriptionID(ctx context.Context, stripeID string) (*Subscription, error)
}

// PaymentRepository defines the interface for payment persistence
type PaymentRepository interface {
	// Create creates a new payment
	Create(ctx context.Context, payment *Payment) error

	// GetByID retrieves a payment by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Payment, error)

	// GetByUserID retrieves payments for a user
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Payment, int64, error)

	// GetBySubscriptionID retrieves payments for a subscription
	GetBySubscriptionID(ctx context.Context, subscriptionID uuid.UUID) ([]*Payment, error)

	// Update updates a payment
	Update(ctx context.Context, payment *Payment) error

	// GetByStripePaymentID retrieves a payment by Stripe ID
	GetByStripePaymentID(ctx context.Context, stripeID string) (*Payment, error)

	// GetRevenue retrieves total revenue for a period
	GetRevenue(ctx context.Context, from, to time.Time) (decimal.Decimal, error)
}

// InvoiceRepository defines the interface for invoice persistence
type InvoiceRepository interface {
	// Create creates a new invoice
	Create(ctx context.Context, invoice *Invoice) error

	// GetByID retrieves an invoice by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Invoice, error)

	// GetByUserID retrieves invoices for a user
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Invoice, int64, error)

	// GetByNumber retrieves an invoice by number
	GetByNumber(ctx context.Context, number string) (*Invoice, error)

	// Update updates an invoice
	Update(ctx context.Context, invoice *Invoice) error

	// GetUnpaidInvoices retrieves unpaid invoices
	GetUnpaidInvoices(ctx context.Context, overdueDays int) ([]*Invoice, error)
}

// PaymentMethodRepository defines the interface for payment method persistence
type PaymentMethodRepository interface {
	// Create creates a new payment method
	Create(ctx context.Context, method *PaymentMethod) error

	// GetByID retrieves a payment method by ID
	GetByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error)

	// GetByUserID retrieves payment methods for a user
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*PaymentMethod, error)

	// GetDefaultByUserID retrieves the default payment method for a user
	GetDefaultByUserID(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error)

	// Update updates a payment method
	Update(ctx context.Context, method *PaymentMethod) error

	// Delete deletes a payment method
	Delete(ctx context.Context, id uuid.UUID) error

	// SetDefault sets a payment method as default
	SetDefault(ctx context.Context, userID, methodID uuid.UUID) error
}

// UsageRepository defines the interface for usage tracking
type UsageRepository interface {
	// RecordUsage records usage data
	RecordUsage(ctx context.Context, record *UsageRecord) error

	// GetUsage retrieves usage for a subscription within a period
	GetUsage(ctx context.Context, subscriptionID uuid.UUID, metricName string, from, to time.Time) ([]UsageRecord, error)

	// GetAggregatedUsage retrieves aggregated usage for billing
	GetAggregatedUsage(ctx context.Context, subscriptionID uuid.UUID, from, to time.Time) (map[string]decimal.Decimal, error)
}

// CouponRepository defines the interface for coupon persistence
type CouponRepository interface {
	// Create creates a new coupon
	Create(ctx context.Context, coupon *Coupon) error

	// GetByID retrieves a coupon by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Coupon, error)

	// GetByCode retrieves a coupon by code
	GetByCode(ctx context.Context, code string) (*Coupon, error)

	// Update updates a coupon
	Update(ctx context.Context, coupon *Coupon) error

	// IncrementRedemption increments the redemption count
	IncrementRedemption(ctx context.Context, id uuid.UUID) error

	// IsValid checks if a coupon is valid
	IsValid(ctx context.Context, code string) (bool, error)
}

// BillingService defines the main interface for billing operations
type BillingService interface {
	// Subscription management
	CreateSubscription(ctx context.Context, req *CreateSubscriptionRequest) (*Subscription, error)
	UpdateSubscription(ctx context.Context, req *UpdateSubscriptionRequest) (*Subscription, error)
	CancelSubscription(ctx context.Context, req *CancelSubscriptionRequest) error
	ResumeSubscription(ctx context.Context, subscriptionID uuid.UUID) error
	GetSubscription(ctx context.Context, subscriptionID uuid.UUID) (*Subscription, error)
	GetUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]*Subscription, error)
	GetCurrentSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error)

	// Plan management
	CreatePlan(ctx context.Context, plan *Plan) error
	UpdatePlan(ctx context.Context, plan *Plan) error
	DeletePlan(ctx context.Context, planID uuid.UUID) error
	GetPlan(ctx context.Context, planID uuid.UUID) (*Plan, error)
	ListPlans(ctx context.Context) ([]*Plan, error)

	// Payment processing
	ProcessPayment(ctx context.Context, req *CreatePaymentRequest) (*Payment, error)
	RefundPayment(ctx context.Context, req *RefundRequest) (*Payment, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error)
	GetUserPayments(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Payment, int64, error)

	// Invoice management
	CreateInvoice(ctx context.Context, userID uuid.UUID, items []InvoiceLineItem) (*Invoice, error)
	GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error)
	GetUserInvoices(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Invoice, int64, error)
	SendInvoice(ctx context.Context, invoiceID uuid.UUID) error
	MarkInvoiceAsPaid(ctx context.Context, invoiceID uuid.UUID) error

	// Payment method management
	AddPaymentMethod(ctx context.Context, method *PaymentMethod) error
	UpdatePaymentMethod(ctx context.Context, method *PaymentMethod) error
	DeletePaymentMethod(ctx context.Context, methodID uuid.UUID) error
	SetDefaultPaymentMethod(ctx context.Context, userID, methodID uuid.UUID) error
	GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*PaymentMethod, error)

	// Usage tracking
	RecordUsage(ctx context.Context, record *UsageRecord) error
	GetUsageReport(ctx context.Context, userID uuid.UUID, from, to time.Time) (map[string]decimal.Decimal, error)

	// Coupon management
	CreateCoupon(ctx context.Context, coupon *Coupon) error
	ValidateCoupon(ctx context.Context, code string) (*Coupon, error)
	ApplyCoupon(ctx context.Context, subscriptionID uuid.UUID, code string) error

	// Webhooks
	HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error

	// Analytics
	GetRevenue(ctx context.Context, from, to time.Time) (decimal.Decimal, error)
	GetChurnRate(ctx context.Context, from, to time.Time) (float64, error)
	GetMRR(ctx context.Context) (decimal.Decimal, error) // Monthly Recurring Revenue
	GetARR(ctx context.Context) (decimal.Decimal, error) // Annual Recurring Revenue
}

// PaymentGateway defines the interface for payment processing
type PaymentGateway interface {
	// Customer management
	CreateCustomer(ctx context.Context, userID uuid.UUID, email string) (string, error)
	UpdateCustomer(ctx context.Context, customerID string, email string) error
	DeleteCustomer(ctx context.Context, customerID string) error

	// Subscription management
	CreateSubscription(ctx context.Context, customerID, priceID string, trialDays int) (string, error)
	UpdateSubscription(ctx context.Context, subscriptionID, priceID string) error
	CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) error
	ResumeSubscription(ctx context.Context, subscriptionID string) error

	// Payment processing
	ChargePayment(ctx context.Context, amount decimal.Decimal, currency, customerID, paymentMethodID string) (string, error)
	RefundPayment(ctx context.Context, paymentID string, amount decimal.Decimal) (string, error)

	// Payment method management
	AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error
	DetachPaymentMethod(ctx context.Context, paymentMethodID string) error
	SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error

	// Invoice management
	CreateInvoice(ctx context.Context, customerID string, items []InvoiceLineItem) (string, error)
	FinalizeInvoice(ctx context.Context, invoiceID string) error
	PayInvoice(ctx context.Context, invoiceID string) error

	// Webhook validation
	ValidateWebhookSignature(payload []byte, signature string) bool
}

// NotificationService defines the interface for billing notifications
type NotificationService interface {
	// Send payment confirmation
	SendPaymentConfirmation(ctx context.Context, userID uuid.UUID, payment *Payment) error

	// Send invoice
	SendInvoice(ctx context.Context, userID uuid.UUID, invoice *Invoice) error

	// Send subscription renewal reminder
	SendRenewalReminder(ctx context.Context, userID uuid.UUID, subscription *Subscription) error

	// Send payment failure notification
	SendPaymentFailureNotification(ctx context.Context, userID uuid.UUID, payment *Payment) error

	// Send subscription cancellation confirmation
	SendCancellationConfirmation(ctx context.Context, userID uuid.UUID, subscription *Subscription) error
}
