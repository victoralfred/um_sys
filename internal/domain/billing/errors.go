package billing

import "errors"

var (
	// Subscription errors
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrSubscriptionInactive      = errors.New("subscription is inactive")
	ErrSubscriptionCanceled      = errors.New("subscription is canceled")
	ErrSubscriptionPastDue       = errors.New("subscription is past due")
	ErrNoActiveSubscription      = errors.New("no active subscription found")
	ErrCannotDowngradePlan       = errors.New("cannot downgrade to this plan")
	ErrTrialAlreadyUsed          = errors.New("trial period already used")

	// Plan errors
	ErrPlanNotFound      = errors.New("plan not found")
	ErrPlanInactive      = errors.New("plan is inactive")
	ErrInvalidPlanType   = errors.New("invalid plan type")
	ErrPlanLimitExceeded = errors.New("plan limit exceeded")

	// Payment errors
	ErrPaymentNotFound            = errors.New("payment not found")
	ErrPaymentFailed              = errors.New("payment failed")
	ErrInsufficientFunds          = errors.New("insufficient funds")
	ErrCardDeclined               = errors.New("card declined")
	ErrInvalidPaymentMethod       = errors.New("invalid payment method")
	ErrPaymentMethodNotFound      = errors.New("payment method not found")
	ErrNoDefaultPaymentMethod     = errors.New("no default payment method set")
	ErrPaymentAlreadyRefunded     = errors.New("payment already refunded")
	ErrRefundAmountExceedsPayment = errors.New("refund amount exceeds payment amount")

	// Invoice errors
	ErrInvoiceNotFound    = errors.New("invoice not found")
	ErrInvoiceAlreadyPaid = errors.New("invoice already paid")
	ErrInvoiceVoid        = errors.New("invoice is void")
	ErrInvoiceOverdue     = errors.New("invoice is overdue")

	// Coupon errors
	ErrCouponNotFound     = errors.New("coupon not found")
	ErrCouponExpired      = errors.New("coupon has expired")
	ErrCouponInactive     = errors.New("coupon is inactive")
	ErrCouponAlreadyUsed  = errors.New("coupon already used")
	ErrCouponLimitReached = errors.New("coupon redemption limit reached")
	ErrInvalidCouponCode  = errors.New("invalid coupon code")

	// Usage errors
	ErrUsageNotFound      = errors.New("usage data not found")
	ErrUsageLimitExceeded = errors.New("usage limit exceeded")
	ErrInvalidUsageMetric = errors.New("invalid usage metric")

	// Gateway errors
	ErrGatewayConnection       = errors.New("payment gateway connection failed")
	ErrGatewayTimeout          = errors.New("payment gateway timeout")
	ErrGatewayInvalidResponse  = errors.New("invalid gateway response")
	ErrCustomerNotFound        = errors.New("customer not found in payment gateway")
	ErrWebhookValidationFailed = errors.New("webhook signature validation failed")

	// General errors
	ErrInvalidAmount    = errors.New("invalid amount")
	ErrInvalidCurrency  = errors.New("invalid currency")
	ErrBillingDisabled  = errors.New("billing is disabled for this user")
	ErrPermissionDenied = errors.New("permission denied for billing operation")
)
