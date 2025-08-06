package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/victoralfred/um_sys/internal/domain/feature"
	"github.com/victoralfred/um_sys/internal/services"
)

type MockFlagRepository struct {
	mock.Mock
}

func (m *MockFlagRepository) Create(ctx context.Context, flag *feature.Flag) error {
	args := m.Called(ctx, flag)
	return args.Error(0)
}

func (m *MockFlagRepository) GetByID(ctx context.Context, id uuid.UUID) (*feature.Flag, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*feature.Flag), args.Error(1)
}

func (m *MockFlagRepository) GetByKey(ctx context.Context, key string) (*feature.Flag, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*feature.Flag), args.Error(1)
}

func (m *MockFlagRepository) List(ctx context.Context, limit, offset int) ([]*feature.Flag, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*feature.Flag), args.Get(1).(int64), args.Error(2)
}

func (m *MockFlagRepository) ListByTags(ctx context.Context, tags []string) ([]*feature.Flag, error) {
	args := m.Called(ctx, tags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*feature.Flag), args.Error(1)
}

func (m *MockFlagRepository) Update(ctx context.Context, flag *feature.Flag) error {
	args := m.Called(ctx, flag)
	return args.Error(0)
}

func (m *MockFlagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFlagRepository) Archive(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockOverrideRepository struct {
	mock.Mock
}

func (m *MockOverrideRepository) Create(ctx context.Context, override *feature.Override) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *MockOverrideRepository) GetByID(ctx context.Context, id uuid.UUID) (*feature.Override, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*feature.Override), args.Error(1)
}

func (m *MockOverrideRepository) GetByUser(ctx context.Context, flagID, userID uuid.UUID) (*feature.Override, error) {
	args := m.Called(ctx, flagID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*feature.Override), args.Error(1)
}

func (m *MockOverrideRepository) GetByGroup(ctx context.Context, flagID, groupID uuid.UUID) (*feature.Override, error) {
	args := m.Called(ctx, flagID, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*feature.Override), args.Error(1)
}

func (m *MockOverrideRepository) ListByFlag(ctx context.Context, flagID uuid.UUID) ([]*feature.Override, error) {
	args := m.Called(ctx, flagID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*feature.Override), args.Error(1)
}

func (m *MockOverrideRepository) Update(ctx context.Context, override *feature.Override) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *MockOverrideRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOverrideRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

type MockEventRepository struct {
	mock.Mock
}

func (m *MockEventRepository) Record(ctx context.Context, event *feature.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventRepository) GetEvents(ctx context.Context, flagKey string, from, to time.Time) ([]*feature.Event, error) {
	args := m.Called(ctx, flagKey, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*feature.Event), args.Error(1)
}

func (m *MockEventRepository) GetAnalytics(ctx context.Context, flagKey string, from, to time.Time) (*feature.Analytics, error) {
	args := m.Called(ctx, flagKey, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*feature.Analytics), args.Error(1)
}

func (m *MockEventRepository) GetUserEvents(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*feature.Event, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*feature.Event), args.Get(1).(int64), args.Error(2)
}

func (m *MockEventRepository) DeleteOldEvents(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func TestFeatureService_CreateFlag(t *testing.T) {
	ctx := context.Background()

	t.Run("successful flag creation", func(t *testing.T) {
		mockFlagRepo := new(MockFlagRepository)
		mockOverrideRepo := new(MockOverrideRepository)
		mockEventRepo := new(MockEventRepository)

		featureService := services.NewFeatureService(
			mockFlagRepo,
			nil,
			mockOverrideRepo,
			nil,
			mockEventRepo,
			nil,
			nil,
			nil,
		)

		createdBy := uuid.New()
		req := &feature.CreateFlagRequest{
			Key:          "new-feature",
			Name:         "New Feature",
			Description:  "A new feature flag",
			Type:         feature.FlagTypeBoolean,
			DefaultValue: false,
			IsEnabled:    true,
			Tags:         []string{"beta", "frontend"},
		}

		mockFlagRepo.On("GetByKey", ctx, req.Key).Return(nil, feature.ErrFlagNotFound)
		mockFlagRepo.On("Create", ctx, mock.AnythingOfType("*feature.Flag")).Return(nil)

		flag, err := featureService.CreateFlag(ctx, req, createdBy)

		require.NoError(t, err)
		assert.NotNil(t, flag)
		assert.Equal(t, req.Key, flag.Key)
		assert.Equal(t, req.Name, flag.Name)
		assert.Equal(t, req.Type, flag.Type)
		assert.Equal(t, req.DefaultValue, flag.DefaultValue)
		assert.Equal(t, createdBy, flag.CreatedBy)

		mockFlagRepo.AssertExpectations(t)
	})

	t.Run("flag already exists", func(t *testing.T) {
		mockFlagRepo := new(MockFlagRepository)

		featureService := services.NewFeatureService(
			mockFlagRepo,
			nil, nil, nil, nil, nil, nil, nil,
		)

		existingFlag := &feature.Flag{
			ID:  uuid.New(),
			Key: "existing-feature",
		}

		req := &feature.CreateFlagRequest{
			Key:          "existing-feature",
			Name:         "Existing Feature",
			Type:         feature.FlagTypeBoolean,
			DefaultValue: false,
		}

		mockFlagRepo.On("GetByKey", ctx, req.Key).Return(existingFlag, nil)

		flag, err := featureService.CreateFlag(ctx, req, uuid.New())

		assert.Error(t, err)
		assert.Equal(t, feature.ErrFlagAlreadyExists, err)
		assert.Nil(t, flag)

		mockFlagRepo.AssertExpectations(t)
	})
}

func TestFeatureService_Evaluate(t *testing.T) {
	ctx := context.Background()

	t.Run("evaluate boolean flag", func(t *testing.T) {
		mockFlagRepo := new(MockFlagRepository)
		mockOverrideRepo := new(MockOverrideRepository)
		mockEventRepo := new(MockEventRepository)

		featureService := services.NewFeatureService(
			mockFlagRepo,
			nil,
			mockOverrideRepo,
			nil,
			mockEventRepo,
			nil,
			nil,
			nil,
		)

		userID := uuid.New()
		flag := &feature.Flag{
			ID:           uuid.New(),
			Key:          "test-feature",
			Type:         feature.FlagTypeBoolean,
			DefaultValue: true,
			IsEnabled:    true,
		}

		req := &feature.EvaluationRequest{
			FlagKey: "test-feature",
			Context: feature.Context{
				UserID: &userID,
			},
		}

		mockFlagRepo.On("GetByKey", ctx, req.FlagKey).Return(flag, nil)
		mockOverrideRepo.On("GetByUser", ctx, flag.ID, userID).Return(nil, feature.ErrOverrideNotFound)
		mockEventRepo.On("Record", ctx, mock.AnythingOfType("*feature.Event")).Return(nil)

		evaluation, err := featureService.Evaluate(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, evaluation)
		assert.Equal(t, req.FlagKey, evaluation.FlagKey)
		assert.Equal(t, true, evaluation.Value)
		assert.True(t, evaluation.IsDefault)

		mockFlagRepo.AssertExpectations(t)
		mockOverrideRepo.AssertExpectations(t)
	})

	t.Run("evaluate with override", func(t *testing.T) {
		mockFlagRepo := new(MockFlagRepository)
		mockOverrideRepo := new(MockOverrideRepository)
		mockEventRepo := new(MockEventRepository)

		featureService := services.NewFeatureService(
			mockFlagRepo,
			nil,
			mockOverrideRepo,
			nil,
			mockEventRepo,
			nil,
			nil,
			nil,
		)

		userID := uuid.New()
		flag := &feature.Flag{
			ID:           uuid.New(),
			Key:          "test-feature",
			Type:         feature.FlagTypeBoolean,
			DefaultValue: false,
			IsEnabled:    true,
		}

		override := &feature.Override{
			ID:     uuid.New(),
			FlagID: flag.ID,
			UserID: &userID,
			Value:  true,
			Reason: "User in beta group",
		}

		req := &feature.EvaluationRequest{
			FlagKey: "test-feature",
			Context: feature.Context{
				UserID: &userID,
			},
		}

		mockFlagRepo.On("GetByKey", ctx, req.FlagKey).Return(flag, nil)
		mockOverrideRepo.On("GetByUser", ctx, flag.ID, userID).Return(override, nil)
		mockEventRepo.On("Record", ctx, mock.AnythingOfType("*feature.Event")).Return(nil)

		evaluation, err := featureService.Evaluate(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, evaluation)
		assert.Equal(t, req.FlagKey, evaluation.FlagKey)
		assert.Equal(t, true, evaluation.Value)
		assert.False(t, evaluation.IsDefault)
		assert.Equal(t, "User override", evaluation.Reason)

		mockFlagRepo.AssertExpectations(t)
		mockOverrideRepo.AssertExpectations(t)
	})

	t.Run("flag disabled", func(t *testing.T) {
		mockFlagRepo := new(MockFlagRepository)

		featureService := services.NewFeatureService(
			mockFlagRepo,
			nil, nil, nil, nil, nil, nil, nil,
		)

		flag := &feature.Flag{
			ID:           uuid.New(),
			Key:          "disabled-feature",
			Type:         feature.FlagTypeBoolean,
			DefaultValue: true,
			IsEnabled:    false,
		}

		req := &feature.EvaluationRequest{
			FlagKey: "disabled-feature",
			Context: feature.Context{},
		}

		mockFlagRepo.On("GetByKey", ctx, req.FlagKey).Return(flag, nil)

		evaluation, err := featureService.Evaluate(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, feature.ErrFlagDisabled, err)
		assert.Nil(t, evaluation)

		mockFlagRepo.AssertExpectations(t)
	})

	t.Run("flag not found", func(t *testing.T) {
		mockFlagRepo := new(MockFlagRepository)

		featureService := services.NewFeatureService(
			mockFlagRepo,
			nil, nil, nil, nil, nil, nil, nil,
		)

		req := &feature.EvaluationRequest{
			FlagKey: "non-existent",
			Context: feature.Context{},
		}

		mockFlagRepo.On("GetByKey", ctx, req.FlagKey).Return(nil, feature.ErrFlagNotFound)

		evaluation, err := featureService.Evaluate(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, feature.ErrFlagNotFound, err)
		assert.Nil(t, evaluation)

		mockFlagRepo.AssertExpectations(t)
	})
}
