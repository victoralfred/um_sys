package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/victoralfred/um_sys/internal/domain/billing"
)

type BillingService struct {
	planRepo          billing.PlanRepository
	subscriptionRepo  billing.SubscriptionRepository
	paymentRepo       billing.PaymentRepository
	invoiceRepo       billing.InvoiceRepository
	paymentMethodRepo billing.PaymentMethodRepository
	usageRepo         billing.UsageRepository
	couponRepo        billing.CouponRepository
	gateway           billing.PaymentGateway
	notificationSvc   billing.NotificationService
}

func NewBillingService(
	planRepo billing.PlanRepository,
	subscriptionRepo billing.SubscriptionRepository,
	paymentRepo billing.PaymentRepository,
	invoiceRepo billing.InvoiceRepository,
	paymentMethodRepo billing.PaymentMethodRepository,
	usageRepo billing.UsageRepository,
	couponRepo billing.CouponRepository,
	gateway billing.PaymentGateway,
	notificationSvc billing.NotificationService,
) *BillingService {
	return &BillingService{
		planRepo:          planRepo,
		subscriptionRepo:  subscriptionRepo,
		paymentRepo:       paymentRepo,
		invoiceRepo:       invoiceRepo,
		paymentMethodRepo: paymentMethodRepo,
		usageRepo:         usageRepo,
		couponRepo:        couponRepo,
		gateway:           gateway,
		notificationSvc:   notificationSvc,
	}
}

func (s *BillingService) CreateSubscription(ctx context.Context, req *billing.CreateSubscriptionRequest) (*billing.Subscription, error) {
	plan, err := s.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		return nil, err
	}

	if !plan.IsActive {
		return nil, billing.ErrPlanInactive
	}

	existingSub, err := s.subscriptionRepo.GetActiveByUserID(ctx, req.UserID)
	if err == nil && existingSub != nil {
		return nil, billing.ErrSubscriptionAlreadyExists
	} else if err != nil && err != billing.ErrNoActiveSubscription {
		return nil, err
	}

	stripeCustomerID, err := s.gateway.CreateCustomer(ctx, req.UserID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	trialDays := plan.TrialDays
	if req.TrialDays > 0 {
		trialDays = req.TrialDays
	}

	stripeSubscriptionID, err := s.gateway.CreateSubscription(ctx, stripeCustomerID, "", trialDays)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	now := time.Now()
	subscription := &billing.Subscription{
		ID:                   uuid.New(),
		UserID:               req.UserID,
		PlanID:               req.PlanID,
		Status:               billing.SubscriptionStatusActive,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.AddDate(0, 1, 0),
		StripeSubscriptionID: stripeSubscriptionID,
		StripeCustomerID:     stripeCustomerID,
		Metadata:             req.Metadata,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if trialDays > 0 {
		subscription.Status = billing.SubscriptionStatusTrialing
		trialStart := now
		trialEnd := now.AddDate(0, 0, trialDays)
		subscription.TrialStart = &trialStart
		subscription.TrialEnd = &trialEnd
	}

	if req.CouponCode != "" {
		coupon, err := s.couponRepo.GetByCode(ctx, req.CouponCode)
		if err == nil && coupon != nil && coupon.IsActive {
			_ = s.couponRepo.IncrementRedemption(ctx, coupon.ID)
		}
	}

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		return nil, err
	}

	if s.notificationSvc != nil {
		_ = s.notificationSvc.SendPaymentConfirmation(ctx, req.UserID, nil)
	}

	return subscription, nil
}

func (s *BillingService) UpdateSubscription(ctx context.Context, req *billing.UpdateSubscriptionRequest) (*billing.Subscription, error) {
	subscription, err := s.subscriptionRepo.GetByID(ctx, req.SubscriptionID)
	if err != nil {
		return nil, err
	}

	if subscription.Status == billing.SubscriptionStatusCanceled {
		return nil, billing.ErrSubscriptionCanceled
	}

	if req.PlanID != nil {
		plan, err := s.planRepo.GetByID(ctx, *req.PlanID)
		if err != nil {
			return nil, err
		}

		if !plan.IsActive {
			return nil, billing.ErrPlanInactive
		}

		currentPlan, err := s.planRepo.GetByID(ctx, subscription.PlanID)
		if err != nil {
			return nil, err
		}

		if plan.Price.LessThan(currentPlan.Price) {
			return nil, billing.ErrCannotDowngradePlan
		}

		subscription.PlanID = *req.PlanID

		if subscription.StripeSubscriptionID != "" {
			if err := s.gateway.UpdateSubscription(ctx, subscription.StripeSubscriptionID, ""); err != nil {
				return nil, fmt.Errorf("failed to update subscription: %w", err)
			}
		}
	}

	if req.PaymentMethodID != nil {
		method, err := s.paymentMethodRepo.GetByID(ctx, *req.PaymentMethodID)
		if err != nil {
			return nil, err
		}

		if method.UserID != subscription.UserID {
			return nil, billing.ErrPermissionDenied
		}

		if err := s.gateway.SetDefaultPaymentMethod(ctx, subscription.StripeCustomerID, method.StripePaymentMethodID); err != nil {
			return nil, fmt.Errorf("failed to update payment method: %w", err)
		}
	}

	if req.Metadata != nil {
		subscription.Metadata = req.Metadata
	}

	subscription.UpdatedAt = time.Now()

	if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

func (s *BillingService) CancelSubscription(ctx context.Context, req *billing.CancelSubscriptionRequest) error {
	subscription, err := s.subscriptionRepo.GetByID(ctx, req.SubscriptionID)
	if err != nil {
		return err
	}

	if subscription.Status == billing.SubscriptionStatusCanceled {
		return billing.ErrSubscriptionCanceled
	}

	cancelAtPeriodEnd := !req.CancelImmediately

	if subscription.StripeSubscriptionID != "" {
		if err := s.gateway.CancelSubscription(ctx, subscription.StripeSubscriptionID, cancelAtPeriodEnd); err != nil {
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}
	}

	if err := s.subscriptionRepo.Cancel(ctx, req.SubscriptionID, cancelAtPeriodEnd); err != nil {
		return err
	}

	if s.notificationSvc != nil {
		_ = s.notificationSvc.SendCancellationConfirmation(ctx, subscription.UserID, subscription)
	}

	return nil
}

func (s *BillingService) ResumeSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	subscription, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if subscription.Status != billing.SubscriptionStatusCanceled || !subscription.CancelAtPeriodEnd {
		return fmt.Errorf("subscription cannot be resumed")
	}

	if subscription.StripeSubscriptionID != "" {
		if err := s.gateway.ResumeSubscription(ctx, subscription.StripeSubscriptionID); err != nil {
			return fmt.Errorf("failed to resume subscription: %w", err)
		}
	}

	subscription.Status = billing.SubscriptionStatusActive
	subscription.CancelAtPeriodEnd = false
	subscription.CanceledAt = nil
	subscription.UpdatedAt = time.Now()

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *BillingService) GetSubscription(ctx context.Context, subscriptionID uuid.UUID) (*billing.Subscription, error) {
	return s.subscriptionRepo.GetByID(ctx, subscriptionID)
}

func (s *BillingService) GetUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]*billing.Subscription, error) {
	return s.subscriptionRepo.GetByUserID(ctx, userID)
}

func (s *BillingService) GetCurrentSubscription(ctx context.Context, userID uuid.UUID) (*billing.Subscription, error) {
	return s.subscriptionRepo.GetActiveByUserID(ctx, userID)
}

func (s *BillingService) CreatePlan(ctx context.Context, plan *billing.Plan) error {
	if plan.ID == uuid.Nil {
		plan.ID = uuid.New()
	}
	plan.CreatedAt = time.Now()
	plan.UpdatedAt = time.Now()
	return s.planRepo.Create(ctx, plan)
}

func (s *BillingService) UpdatePlan(ctx context.Context, plan *billing.Plan) error {
	existing, err := s.planRepo.GetByID(ctx, plan.ID)
	if err != nil {
		return err
	}

	plan.CreatedAt = existing.CreatedAt
	plan.UpdatedAt = time.Now()

	return s.planRepo.Update(ctx, plan)
}

func (s *BillingService) DeletePlan(ctx context.Context, planID uuid.UUID) error {
	return s.planRepo.Delete(ctx, planID)
}

func (s *BillingService) GetPlan(ctx context.Context, planID uuid.UUID) (*billing.Plan, error) {
	return s.planRepo.GetByID(ctx, planID)
}

func (s *BillingService) ListPlans(ctx context.Context) ([]*billing.Plan, error) {
	return s.planRepo.List(ctx)
}

func (s *BillingService) ProcessPayment(ctx context.Context, req *billing.CreatePaymentRequest) (*billing.Payment, error) {
	method, err := s.paymentMethodRepo.GetByID(ctx, req.PaymentMethodID)
	if err != nil {
		return nil, err
	}

	if method.UserID != req.UserID {
		return nil, billing.ErrPermissionDenied
	}

	stripePaymentID, err := s.gateway.ChargePayment(ctx, req.Amount, req.Currency, "", method.StripePaymentMethodID)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	now := time.Now()
	payment := &billing.Payment{
		ID:              uuid.New(),
		UserID:          req.UserID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          billing.PaymentStatusSucceeded,
		Description:     req.Description,
		PaymentMethod:   method.Type,
		StripePaymentID: stripePaymentID,
		RefundedAmount:  decimal.Zero,
		Metadata:        req.Metadata,
		PaidAt:          &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	if s.notificationSvc != nil {
		_ = s.notificationSvc.SendPaymentConfirmation(ctx, req.UserID, payment)
	}

	return payment, nil
}

func (s *BillingService) RefundPayment(ctx context.Context, req *billing.RefundRequest) (*billing.Payment, error) {
	payment, err := s.paymentRepo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, err
	}

	if payment.Status == billing.PaymentStatusRefunded {
		return nil, billing.ErrPaymentAlreadyRefunded
	}

	refundAmount := req.Amount
	if refundAmount.IsZero() {
		refundAmount = payment.Amount
	}

	if refundAmount.GreaterThan(payment.Amount.Sub(payment.RefundedAmount)) {
		return nil, billing.ErrRefundAmountExceedsPayment
	}

	_, err = s.gateway.RefundPayment(ctx, payment.StripePaymentID, refundAmount)
	if err != nil {
		return nil, fmt.Errorf("refund failed: %w", err)
	}

	payment.RefundedAmount = payment.RefundedAmount.Add(refundAmount)
	if payment.RefundedAmount.Equal(payment.Amount) {
		payment.Status = billing.PaymentStatusRefunded
	}
	payment.UpdatedAt = time.Now()

	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *BillingService) GetPayment(ctx context.Context, paymentID uuid.UUID) (*billing.Payment, error) {
	return s.paymentRepo.GetByID(ctx, paymentID)
}

func (s *BillingService) GetUserPayments(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*billing.Payment, int64, error) {
	return s.paymentRepo.GetByUserID(ctx, userID, limit, offset)
}

func (s *BillingService) CreateInvoice(ctx context.Context, userID uuid.UUID, items []billing.InvoiceLineItem) (*billing.Invoice, error) {
	stripeInvoiceID, err := s.gateway.CreateInvoice(ctx, "", items)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	var total decimal.Decimal
	for _, item := range items {
		total = total.Add(item.Total)
	}

	now := time.Now()
	invoice := &billing.Invoice{
		ID:              uuid.New(),
		UserID:          userID,
		InvoiceNumber:   fmt.Sprintf("INV-%d", time.Now().Unix()),
		Amount:          total,
		Tax:             decimal.Zero,
		Total:           total,
		Currency:        "USD",
		Status:          "open",
		DueDate:         now.AddDate(0, 0, 30),
		LineItems:       items,
		StripeInvoiceID: stripeInvoiceID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}

func (s *BillingService) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*billing.Invoice, error) {
	return s.invoiceRepo.GetByID(ctx, invoiceID)
}

func (s *BillingService) GetUserInvoices(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*billing.Invoice, int64, error) {
	return s.invoiceRepo.GetByUserID(ctx, userID, limit, offset)
}

func (s *BillingService) SendInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	invoice, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}

	if s.notificationSvc != nil {
		return s.notificationSvc.SendInvoice(ctx, invoice.UserID, invoice)
	}

	return nil
}

func (s *BillingService) MarkInvoiceAsPaid(ctx context.Context, invoiceID uuid.UUID) error {
	invoice, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}

	if invoice.Status == "paid" {
		return billing.ErrInvoiceAlreadyPaid
	}

	now := time.Now()
	invoice.Status = "paid"
	invoice.PaidAt = &now
	invoice.UpdatedAt = now

	return s.invoiceRepo.Update(ctx, invoice)
}

func (s *BillingService) AddPaymentMethod(ctx context.Context, method *billing.PaymentMethod) error {
	if method.ID == uuid.Nil {
		method.ID = uuid.New()
	}
	method.CreatedAt = time.Now()
	method.UpdatedAt = time.Now()

	if method.StripePaymentMethodID != "" {
		subscription, err := s.subscriptionRepo.GetActiveByUserID(ctx, method.UserID)
		if err == nil && subscription != nil {
			if err := s.gateway.AttachPaymentMethod(ctx, method.StripePaymentMethodID, subscription.StripeCustomerID); err != nil {
				return fmt.Errorf("failed to attach payment method: %w", err)
			}
		}
	}

	return s.paymentMethodRepo.Create(ctx, method)
}

func (s *BillingService) UpdatePaymentMethod(ctx context.Context, method *billing.PaymentMethod) error {
	existing, err := s.paymentMethodRepo.GetByID(ctx, method.ID)
	if err != nil {
		return err
	}

	method.UserID = existing.UserID
	method.CreatedAt = existing.CreatedAt
	method.UpdatedAt = time.Now()

	return s.paymentMethodRepo.Update(ctx, method)
}

func (s *BillingService) DeletePaymentMethod(ctx context.Context, methodID uuid.UUID) error {
	method, err := s.paymentMethodRepo.GetByID(ctx, methodID)
	if err != nil {
		return err
	}

	if method.StripePaymentMethodID != "" {
		if err := s.gateway.DetachPaymentMethod(ctx, method.StripePaymentMethodID); err != nil {
			return fmt.Errorf("failed to detach payment method: %w", err)
		}
	}

	return s.paymentMethodRepo.Delete(ctx, methodID)
}

func (s *BillingService) SetDefaultPaymentMethod(ctx context.Context, userID, methodID uuid.UUID) error {
	method, err := s.paymentMethodRepo.GetByID(ctx, methodID)
	if err != nil {
		return err
	}

	if method.UserID != userID {
		return billing.ErrPermissionDenied
	}

	subscription, err := s.subscriptionRepo.GetActiveByUserID(ctx, userID)
	if err == nil && subscription != nil && method.StripePaymentMethodID != "" {
		if err := s.gateway.SetDefaultPaymentMethod(ctx, subscription.StripeCustomerID, method.StripePaymentMethodID); err != nil {
			return fmt.Errorf("failed to set default payment method: %w", err)
		}
	}

	return s.paymentMethodRepo.SetDefault(ctx, userID, methodID)
}

func (s *BillingService) GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*billing.PaymentMethod, error) {
	return s.paymentMethodRepo.GetByUserID(ctx, userID)
}

func (s *BillingService) RecordUsage(ctx context.Context, record *billing.UsageRecord) error {
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	record.CreatedAt = time.Now()
	return s.usageRepo.RecordUsage(ctx, record)
}

func (s *BillingService) GetUsageReport(ctx context.Context, userID uuid.UUID, from, to time.Time) (map[string]decimal.Decimal, error) {
	subscription, err := s.subscriptionRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.usageRepo.GetAggregatedUsage(ctx, subscription.ID, from, to)
}

func (s *BillingService) CreateCoupon(ctx context.Context, coupon *billing.Coupon) error {
	if coupon.ID == uuid.Nil {
		coupon.ID = uuid.New()
	}
	coupon.CreatedAt = time.Now()
	coupon.UpdatedAt = time.Now()
	return s.couponRepo.Create(ctx, coupon)
}

func (s *BillingService) ValidateCoupon(ctx context.Context, code string) (*billing.Coupon, error) {
	coupon, err := s.couponRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if !coupon.IsActive {
		return nil, billing.ErrCouponInactive
	}

	if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil) {
		return nil, billing.ErrCouponExpired
	}

	if coupon.MaxRedemptions > 0 && coupon.TimesRedeemed >= coupon.MaxRedemptions {
		return nil, billing.ErrCouponLimitReached
	}

	return coupon, nil
}

func (s *BillingService) ApplyCoupon(ctx context.Context, subscriptionID uuid.UUID, code string) error {
	coupon, err := s.ValidateCoupon(ctx, code)
	if err != nil {
		return err
	}

	subscription, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if subscription.Metadata == nil {
		subscription.Metadata = make(map[string]string)
	}
	subscription.Metadata["coupon_code"] = coupon.Code
	subscription.Metadata["coupon_discount"] = coupon.DiscountValue.String()
	subscription.UpdatedAt = time.Now()

	if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
		return err
	}

	return s.couponRepo.IncrementRedemption(ctx, coupon.ID)
}

func (s *BillingService) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	if !s.gateway.ValidateWebhookSignature(payload, signature) {
		return billing.ErrWebhookValidationFailed
	}

	return nil
}

func (s *BillingService) GetRevenue(ctx context.Context, from, to time.Time) (decimal.Decimal, error) {
	return s.paymentRepo.GetRevenue(ctx, from, to)
}

func (s *BillingService) GetChurnRate(ctx context.Context, from, to time.Time) (float64, error) {
	subscriptions, err := s.subscriptionRepo.GetByUserID(ctx, uuid.Nil)
	if err != nil {
		return 0, err
	}

	var total, churned int
	for _, sub := range subscriptions {
		if sub.CreatedAt.After(from) && sub.CreatedAt.Before(to) {
			total++
			if sub.Status == billing.SubscriptionStatusCanceled && sub.CanceledAt != nil &&
				sub.CanceledAt.After(from) && sub.CanceledAt.Before(to) {
				churned++
			}
		}
	}

	if total == 0 {
		return 0, nil
	}

	return float64(churned) / float64(total), nil
}

func (s *BillingService) GetMRR(ctx context.Context) (decimal.Decimal, error) {
	subscriptions, err := s.subscriptionRepo.GetByUserID(ctx, uuid.Nil)
	if err != nil {
		return decimal.Zero, err
	}

	mrr := decimal.Zero
	for _, sub := range subscriptions {
		if sub.Status == billing.SubscriptionStatusActive || sub.Status == billing.SubscriptionStatusTrialing {
			plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
			if err != nil {
				continue
			}

			monthlyAmount := plan.Price
			switch plan.BillingInterval {
			case billing.IntervalYearly:
				monthlyAmount = plan.Price.Div(decimal.NewFromInt(12))
			case billing.IntervalQuarterly:
				monthlyAmount = plan.Price.Div(decimal.NewFromInt(3))
			}

			mrr = mrr.Add(monthlyAmount)
		}
	}

	return mrr, nil
}

func (s *BillingService) GetARR(ctx context.Context) (decimal.Decimal, error) {
	mrr, err := s.GetMRR(ctx)
	if err != nil {
		return decimal.Zero, err
	}

	return mrr.Mul(decimal.NewFromInt(12)), nil
}
