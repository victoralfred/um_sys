package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/victoralfred/um_sys/internal/domain/billing"
	"github.com/victoralfred/um_sys/internal/services"
)

// Mock implementations
type MockPlanRepository struct {
	mock.Mock
}

func (m *MockPlanRepository) Create(ctx context.Context, plan *billing.Plan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockPlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*billing.Plan, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.Plan), args.Error(1)
}

func (m *MockPlanRepository) GetByType(ctx context.Context, planType billing.PlanType) ([]*billing.Plan, error) {
	args := m.Called(ctx, planType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.Plan), args.Error(1)
}

func (m *MockPlanRepository) Update(ctx context.Context, plan *billing.Plan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockPlanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPlanRepository) List(ctx context.Context) ([]*billing.Plan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.Plan), args.Error(1)
}

func (m *MockPlanRepository) GetFeatures(ctx context.Context, planID uuid.UUID) ([]billing.Feature, error) {
	args := m.Called(ctx, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]billing.Feature), args.Error(1)
}

type MockSubscriptionRepository struct {
	mock.Mock
}

func (m *MockSubscriptionRepository) Create(ctx context.Context, subscription *billing.Subscription) error {
	args := m.Called(ctx, subscription)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*billing.Subscription, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*billing.Subscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*billing.Subscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) Update(ctx context.Context, subscription *billing.Subscription) error {
	args := m.Called(ctx, subscription)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) Cancel(ctx context.Context, id uuid.UUID, cancelAtPeriodEnd bool) error {
	args := m.Called(ctx, id, cancelAtPeriodEnd)
	return args.Error(0)
}

func (m *MockSubscriptionRepository) GetExpiringSubscriptions(ctx context.Context, before time.Time) ([]*billing.Subscription, error) {
	args := m.Called(ctx, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) GetByStripeSubscriptionID(ctx context.Context, stripeID string) (*billing.Subscription, error) {
	args := m.Called(ctx, stripeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.Subscription), args.Error(1)
}

type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) Create(ctx context.Context, payment *billing.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*billing.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.Payment), args.Error(1)
}

func (m *MockPaymentRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*billing.Payment, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*billing.Payment), args.Get(1).(int64), args.Error(2)
}

func (m *MockPaymentRepository) GetBySubscriptionID(ctx context.Context, subscriptionID uuid.UUID) ([]*billing.Payment, error) {
	args := m.Called(ctx, subscriptionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.Payment), args.Error(1)
}

func (m *MockPaymentRepository) Update(ctx context.Context, payment *billing.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByStripePaymentID(ctx context.Context, stripeID string) (*billing.Payment, error) {
	args := m.Called(ctx, stripeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.Payment), args.Error(1)
}

func (m *MockPaymentRepository) GetRevenue(ctx context.Context, from, to time.Time) (decimal.Decimal, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) CreateCustomer(ctx context.Context, userID uuid.UUID, email string) (string, error) {
	args := m.Called(ctx, userID, email)
	return args.String(0), args.Error(1)
}

func (m *MockPaymentGateway) UpdateCustomer(ctx context.Context, customerID string, email string) error {
	args := m.Called(ctx, customerID, email)
	return args.Error(0)
}

func (m *MockPaymentGateway) DeleteCustomer(ctx context.Context, customerID string) error {
	args := m.Called(ctx, customerID)
	return args.Error(0)
}

func (m *MockPaymentGateway) CreateSubscription(ctx context.Context, customerID, priceID string, trialDays int) (string, error) {
	args := m.Called(ctx, customerID, priceID, trialDays)
	return args.String(0), args.Error(1)
}

func (m *MockPaymentGateway) UpdateSubscription(ctx context.Context, subscriptionID, priceID string) error {
	args := m.Called(ctx, subscriptionID, priceID)
	return args.Error(0)
}

func (m *MockPaymentGateway) CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) error {
	args := m.Called(ctx, subscriptionID, cancelAtPeriodEnd)
	return args.Error(0)
}

func (m *MockPaymentGateway) ResumeSubscription(ctx context.Context, subscriptionID string) error {
	args := m.Called(ctx, subscriptionID)
	return args.Error(0)
}

func (m *MockPaymentGateway) ChargePayment(ctx context.Context, amount decimal.Decimal, currency, customerID, paymentMethodID string) (string, error) {
	args := m.Called(ctx, amount, currency, customerID, paymentMethodID)
	return args.String(0), args.Error(1)
}

func (m *MockPaymentGateway) RefundPayment(ctx context.Context, paymentID string, amount decimal.Decimal) (string, error) {
	args := m.Called(ctx, paymentID, amount)
	return args.String(0), args.Error(1)
}

func (m *MockPaymentGateway) AttachPaymentMethod(ctx context.Context, paymentMethodID, customerID string) error {
	args := m.Called(ctx, paymentMethodID, customerID)
	return args.Error(0)
}

func (m *MockPaymentGateway) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	args := m.Called(ctx, paymentMethodID)
	return args.Error(0)
}

func (m *MockPaymentGateway) SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	args := m.Called(ctx, customerID, paymentMethodID)
	return args.Error(0)
}

func (m *MockPaymentGateway) CreateInvoice(ctx context.Context, customerID string, items []billing.InvoiceLineItem) (string, error) {
	args := m.Called(ctx, customerID, items)
	return args.String(0), args.Error(1)
}

func (m *MockPaymentGateway) FinalizeInvoice(ctx context.Context, invoiceID string) error {
	args := m.Called(ctx, invoiceID)
	return args.Error(0)
}

func (m *MockPaymentGateway) PayInvoice(ctx context.Context, invoiceID string) error {
	args := m.Called(ctx, invoiceID)
	return args.Error(0)
}

func (m *MockPaymentGateway) ValidateWebhookSignature(payload []byte, signature string) bool {
	args := m.Called(payload, signature)
	return args.Bool(0)
}

func TestBillingService_CreateSubscription(t *testing.T) {
	ctx := context.Background()

	t.Run("successful subscription creation", func(t *testing.T) {
		// Arrange
		mockPlanRepo := new(MockPlanRepository)
		mockSubRepo := new(MockSubscriptionRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockGateway := new(MockPaymentGateway)

		billingService := services.NewBillingService(
			mockPlanRepo,
			mockSubRepo,
			mockPaymentRepo,
			nil, // invoice repo
			nil, // payment method repo
			nil, // usage repo
			nil, // coupon repo
			mockGateway,
			nil, // notification service
		)

		userID := uuid.New()
		planID := uuid.New()

		plan := &billing.Plan{
			ID:              planID,
			Name:            "Pro Plan",
			Type:            billing.PlanTypePro,
			Price:           decimal.NewFromFloat(29.99),
			Currency:        "USD",
			BillingInterval: billing.IntervalMonthly,
			TrialDays:       14,
			IsActive:        true,
		}

		req := &billing.CreateSubscriptionRequest{
			UserID: userID,
			PlanID: planID,
		}

		stripeCustomerID := "cus_test123"
		stripeSubscriptionID := "sub_test123"

		mockPlanRepo.On("GetByID", ctx, planID).Return(plan, nil)
		mockSubRepo.On("GetActiveByUserID", ctx, userID).Return(nil, billing.ErrNoActiveSubscription)
		mockGateway.On("CreateCustomer", ctx, userID, mock.AnythingOfType("string")).Return(stripeCustomerID, nil)
		mockGateway.On("CreateSubscription", ctx, stripeCustomerID, mock.AnythingOfType("string"), plan.TrialDays).Return(stripeSubscriptionID, nil)
		mockSubRepo.On("Create", ctx, mock.AnythingOfType("*billing.Subscription")).Return(nil)

		// Act
		subscription, err := billingService.CreateSubscription(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, subscription)
		assert.Equal(t, userID, subscription.UserID)
		assert.Equal(t, planID, subscription.PlanID)
		assert.Equal(t, billing.SubscriptionStatusTrialing, subscription.Status)
		assert.Equal(t, stripeSubscriptionID, subscription.StripeSubscriptionID)

		mockPlanRepo.AssertExpectations(t)
		mockSubRepo.AssertExpectations(t)
		mockGateway.AssertExpectations(t)
	})

	t.Run("plan not found", func(t *testing.T) {
		// Arrange
		mockPlanRepo := new(MockPlanRepository)
		mockSubRepo := new(MockSubscriptionRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockGateway := new(MockPaymentGateway)

		billingService := services.NewBillingService(
			mockPlanRepo,
			mockSubRepo,
			mockPaymentRepo,
			nil, nil, nil, nil,
			mockGateway,
			nil,
		)

		userID := uuid.New()
		planID := uuid.New()

		req := &billing.CreateSubscriptionRequest{
			UserID: userID,
			PlanID: planID,
		}

		mockPlanRepo.On("GetByID", ctx, planID).Return(nil, billing.ErrPlanNotFound)

		// Act
		subscription, err := billingService.CreateSubscription(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, billing.ErrPlanNotFound, err)
		assert.Nil(t, subscription)

		mockPlanRepo.AssertExpectations(t)
	})

	t.Run("subscription already exists", func(t *testing.T) {
		// Arrange
		mockPlanRepo := new(MockPlanRepository)
		mockSubRepo := new(MockSubscriptionRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockGateway := new(MockPaymentGateway)

		billingService := services.NewBillingService(
			mockPlanRepo,
			mockSubRepo,
			mockPaymentRepo,
			nil, nil, nil, nil,
			mockGateway,
			nil,
		)

		userID := uuid.New()
		planID := uuid.New()

		plan := &billing.Plan{
			ID:       planID,
			IsActive: true,
		}

		existingSub := &billing.Subscription{
			ID:     uuid.New(),
			UserID: userID,
			Status: billing.SubscriptionStatusActive,
		}

		req := &billing.CreateSubscriptionRequest{
			UserID: userID,
			PlanID: planID,
		}

		mockPlanRepo.On("GetByID", ctx, planID).Return(plan, nil)
		mockSubRepo.On("GetActiveByUserID", ctx, userID).Return(existingSub, nil)

		// Act
		subscription, err := billingService.CreateSubscription(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, billing.ErrSubscriptionAlreadyExists, err)
		assert.Nil(t, subscription)

		mockPlanRepo.AssertExpectations(t)
		mockSubRepo.AssertExpectations(t)
	})
}

func TestBillingService_CancelSubscription(t *testing.T) {
	ctx := context.Background()

	t.Run("successful cancellation at period end", func(t *testing.T) {
		// Arrange
		mockPlanRepo := new(MockPlanRepository)
		mockSubRepo := new(MockSubscriptionRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockGateway := new(MockPaymentGateway)

		billingService := services.NewBillingService(
			mockPlanRepo,
			mockSubRepo,
			mockPaymentRepo,
			nil, nil, nil, nil,
			mockGateway,
			nil,
		)

		subID := uuid.New()
		stripeSubID := "sub_test123"

		subscription := &billing.Subscription{
			ID:                   subID,
			Status:               billing.SubscriptionStatusActive,
			StripeSubscriptionID: stripeSubID,
		}

		req := &billing.CancelSubscriptionRequest{
			SubscriptionID:    subID,
			CancelImmediately: false,
			Reason:            "Too expensive",
		}

		mockSubRepo.On("GetByID", ctx, subID).Return(subscription, nil)
		mockGateway.On("CancelSubscription", ctx, stripeSubID, true).Return(nil)
		mockSubRepo.On("Cancel", ctx, subID, true).Return(nil)

		// Act
		err := billingService.CancelSubscription(ctx, req)

		// Assert
		assert.NoError(t, err)

		mockSubRepo.AssertExpectations(t)
		mockGateway.AssertExpectations(t)
	})

	t.Run("subscription not found", func(t *testing.T) {
		// Arrange
		mockPlanRepo := new(MockPlanRepository)
		mockSubRepo := new(MockSubscriptionRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockGateway := new(MockPaymentGateway)

		billingService := services.NewBillingService(
			mockPlanRepo,
			mockSubRepo,
			mockPaymentRepo,
			nil, nil, nil, nil,
			mockGateway,
			nil,
		)

		subID := uuid.New()

		req := &billing.CancelSubscriptionRequest{
			SubscriptionID: subID,
		}

		mockSubRepo.On("GetByID", ctx, subID).Return(nil, billing.ErrSubscriptionNotFound)

		// Act
		err := billingService.CancelSubscription(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, billing.ErrSubscriptionNotFound, err)

		mockSubRepo.AssertExpectations(t)
	})
}

func TestBillingService_ProcessPayment(t *testing.T) {
	ctx := context.Background()

	t.Run("successful payment", func(t *testing.T) {
		// Arrange
		mockPlanRepo := new(MockPlanRepository)
		mockSubRepo := new(MockSubscriptionRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockPaymentMethodRepo := new(MockPaymentMethodRepository)
		mockGateway := new(MockPaymentGateway)

		billingService := services.NewBillingService(
			mockPlanRepo,
			mockSubRepo,
			mockPaymentRepo,
			nil,
			mockPaymentMethodRepo,
			nil, nil,
			mockGateway,
			nil,
		)

		userID := uuid.New()
		paymentMethodID := uuid.New()
		amount := decimal.NewFromFloat(99.99)

		paymentMethod := &billing.PaymentMethod{
			ID:                    paymentMethodID,
			UserID:                userID,
			Type:                  "card",
			StripePaymentMethodID: "pm_test123",
		}

		req := &billing.CreatePaymentRequest{
			UserID:          userID,
			Amount:          amount,
			Currency:        "USD",
			Description:     "One-time payment",
			PaymentMethodID: paymentMethodID,
		}

		stripePaymentID := "pi_test123"

		mockPaymentMethodRepo.On("GetByID", ctx, paymentMethodID).Return(paymentMethod, nil)
		mockGateway.On("ChargePayment", ctx, amount, "USD", mock.AnythingOfType("string"), paymentMethod.StripePaymentMethodID).Return(stripePaymentID, nil)
		mockPaymentRepo.On("Create", ctx, mock.AnythingOfType("*billing.Payment")).Return(nil)

		// Act
		payment, err := billingService.ProcessPayment(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, payment)
		assert.Equal(t, userID, payment.UserID)
		assert.Equal(t, amount, payment.Amount)
		assert.Equal(t, billing.PaymentStatusSucceeded, payment.Status)
		assert.Equal(t, stripePaymentID, payment.StripePaymentID)

		mockPaymentMethodRepo.AssertExpectations(t)
		mockGateway.AssertExpectations(t)
		mockPaymentRepo.AssertExpectations(t)
	})

	t.Run("payment method not found", func(t *testing.T) {
		// Arrange
		mockPlanRepo := new(MockPlanRepository)
		mockSubRepo := new(MockSubscriptionRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockPaymentMethodRepo := new(MockPaymentMethodRepository)
		mockGateway := new(MockPaymentGateway)

		billingService := services.NewBillingService(
			mockPlanRepo,
			mockSubRepo,
			mockPaymentRepo,
			nil,
			mockPaymentMethodRepo,
			nil, nil,
			mockGateway,
			nil,
		)

		userID := uuid.New()
		paymentMethodID := uuid.New()

		req := &billing.CreatePaymentRequest{
			UserID:          userID,
			Amount:          decimal.NewFromFloat(99.99),
			Currency:        "USD",
			PaymentMethodID: paymentMethodID,
		}

		mockPaymentMethodRepo.On("GetByID", ctx, paymentMethodID).Return(nil, billing.ErrPaymentMethodNotFound)

		// Act
		payment, err := billingService.ProcessPayment(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, billing.ErrPaymentMethodNotFound, err)
		assert.Nil(t, payment)

		mockPaymentMethodRepo.AssertExpectations(t)
	})
}

// Add this mock for PaymentMethodRepository
type MockPaymentMethodRepository struct {
	mock.Mock
}

func (m *MockPaymentMethodRepository) Create(ctx context.Context, method *billing.PaymentMethod) error {
	args := m.Called(ctx, method)
	return args.Error(0)
}

func (m *MockPaymentMethodRepository) GetByID(ctx context.Context, id uuid.UUID) (*billing.PaymentMethod, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.PaymentMethod), args.Error(1)
}

func (m *MockPaymentMethodRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*billing.PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*billing.PaymentMethod), args.Error(1)
}

func (m *MockPaymentMethodRepository) GetDefaultByUserID(ctx context.Context, userID uuid.UUID) (*billing.PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billing.PaymentMethod), args.Error(1)
}

func (m *MockPaymentMethodRepository) Update(ctx context.Context, method *billing.PaymentMethod) error {
	args := m.Called(ctx, method)
	return args.Error(0)
}

func (m *MockPaymentMethodRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPaymentMethodRepository) SetDefault(ctx context.Context, userID, methodID uuid.UUID) error {
	args := m.Called(ctx, userID, methodID)
	return args.Error(0)
}
