package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/victoralfred/um_sys/internal/domain/audit"
	"github.com/victoralfred/um_sys/internal/services"
)

type MockLogRepository struct {
	mock.Mock
}

func (m *MockLogRepository) Create(ctx context.Context, entry *audit.LogEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*audit.LogEntry, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*audit.LogEntry), args.Error(1)
}

func (m *MockLogRepository) List(ctx context.Context, filter audit.LogFilter) ([]*audit.LogEntry, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*audit.LogEntry), args.Get(1).(int64), args.Error(2)
}

func (m *MockLogRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*audit.LogEntry, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*audit.LogEntry), args.Get(1).(int64), args.Error(2)
}

func (m *MockLogRepository) GetByEntityID(ctx context.Context, entityType, entityID string, limit, offset int) ([]*audit.LogEntry, int64, error) {
	args := m.Called(ctx, entityType, entityID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*audit.LogEntry), args.Get(1).(int64), args.Error(2)
}

func (m *MockLogRepository) GetSummary(ctx context.Context, filter audit.LogFilter) (*audit.LogSummary, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*audit.LogSummary), args.Error(1)
}

func (m *MockLogRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockLogRepository) Export(ctx context.Context, filter audit.LogFilter, format string) ([]byte, error) {
	args := m.Called(ctx, filter, format)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

type MockAlertRepository struct {
	mock.Mock
}

func (m *MockAlertRepository) CreateRule(ctx context.Context, rule *audit.AlertRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAlertRepository) GetRuleByID(ctx context.Context, id uuid.UUID) (*audit.AlertRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*audit.AlertRule), args.Error(1)
}

func (m *MockAlertRepository) ListRules(ctx context.Context, active bool) ([]*audit.AlertRule, error) {
	args := m.Called(ctx, active)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.AlertRule), args.Error(1)
}

func (m *MockAlertRepository) UpdateRule(ctx context.Context, rule *audit.AlertRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAlertRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAlertRepository) CheckRules(ctx context.Context, entry *audit.LogEntry) ([]*audit.AlertRule, error) {
	args := m.Called(ctx, entry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.AlertRule), args.Error(1)
}

type MockAuditNotificationService struct {
	mock.Mock
}

func (m *MockAuditNotificationService) SendAlert(ctx context.Context, rule *audit.AlertRule, entry *audit.LogEntry) error {
	args := m.Called(ctx, rule, entry)
	return args.Error(0)
}

func (m *MockAuditNotificationService) SendComplianceReport(ctx context.Context, report *audit.ComplianceReport, recipients []string) error {
	args := m.Called(ctx, report, recipients)
	return args.Error(0)
}

func TestAuditService_Log(t *testing.T) {
	ctx := context.Background()

	t.Run("successful log creation", func(t *testing.T) {
		mockLogRepo := new(MockLogRepository)
		mockAlertRepo := new(MockAlertRepository)
		mockNotificationSvc := new(MockAuditNotificationService)

		auditService := services.NewAuditService(
			mockLogRepo,
			mockAlertRepo,
			nil,
			mockNotificationSvc,
		)

		userID := uuid.New()
		req := &audit.CreateLogRequest{
			EventType:   audit.EventTypeUserLoggedIn,
			Severity:    audit.SeverityInfo,
			UserID:      &userID,
			EntityType:  "user",
			EntityID:    userID.String(),
			Action:      "login",
			Description: "User logged in successfully",
			IPAddress:   "192.168.1.1",
			UserAgent:   "Mozilla/5.0",
			Metadata: map[string]interface{}{
				"method": "password",
			},
		}

		mockLogRepo.On("Create", ctx, mock.AnythingOfType("*audit.LogEntry")).Return(nil)
		mockAlertRepo.On("CheckRules", ctx, mock.AnythingOfType("*audit.LogEntry")).Return([]*audit.AlertRule{}, nil)

		entry, err := auditService.Log(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, audit.EventTypeUserLoggedIn, entry.EventType)
		assert.Equal(t, audit.SeverityInfo, entry.Severity)
		assert.Equal(t, userID, *entry.UserID)
		assert.Equal(t, "192.168.1.1", entry.IPAddress)

		mockLogRepo.AssertExpectations(t)
		mockAlertRepo.AssertExpectations(t)
	})

	t.Run("log with alert trigger", func(t *testing.T) {
		mockLogRepo := new(MockLogRepository)
		mockAlertRepo := new(MockAlertRepository)
		mockNotificationSvc := new(MockAuditNotificationService)

		auditService := services.NewAuditService(
			mockLogRepo,
			mockAlertRepo,
			nil,
			mockNotificationSvc,
		)

		req := &audit.CreateLogRequest{
			EventType:   audit.EventTypeSecurityAlert,
			Severity:    audit.SeverityCritical,
			EntityType:  "system",
			EntityID:    "security",
			Action:      "breach_detected",
			Description: "Multiple failed login attempts detected",
		}

		alertRule := &audit.AlertRule{
			ID:         uuid.New(),
			Name:       "Security Alert",
			EventTypes: []audit.EventType{audit.EventTypeSecurityAlert},
			Severities: []audit.Severity{audit.SeverityCritical},
			IsActive:   true,
		}

		mockLogRepo.On("Create", ctx, mock.AnythingOfType("*audit.LogEntry")).Return(nil)
		mockAlertRepo.On("CheckRules", ctx, mock.AnythingOfType("*audit.LogEntry")).Return([]*audit.AlertRule{alertRule}, nil)
		mockNotificationSvc.On("SendAlert", ctx, alertRule, mock.AnythingOfType("*audit.LogEntry")).Return(nil)

		entry, err := auditService.Log(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, audit.EventTypeSecurityAlert, entry.EventType)
		assert.Equal(t, audit.SeverityCritical, entry.Severity)

		mockLogRepo.AssertExpectations(t)
		mockAlertRepo.AssertExpectations(t)
		mockNotificationSvc.AssertExpectations(t)
	})

	t.Run("missing required fields", func(t *testing.T) {
		mockLogRepo := new(MockLogRepository)
		mockAlertRepo := new(MockAlertRepository)

		auditService := services.NewAuditService(
			mockLogRepo,
			mockAlertRepo,
			nil,
			nil,
		)

		req := &audit.CreateLogRequest{
			EventType: audit.EventTypeUserLoggedIn,
		}

		entry, err := auditService.Log(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, audit.ErrMissingRequiredFields, err)
		assert.Nil(t, entry)
	})
}

func TestAuditService_GetLogs(t *testing.T) {
	ctx := context.Background()

	t.Run("successful log retrieval", func(t *testing.T) {
		mockLogRepo := new(MockLogRepository)

		auditService := services.NewAuditService(
			mockLogRepo,
			nil,
			nil,
			nil,
		)

		userID := uuid.New()
		filter := audit.LogFilter{
			UserID: &userID,
			Limit:  10,
			Offset: 0,
		}

		logs := []*audit.LogEntry{
			{
				ID:        uuid.New(),
				EventType: audit.EventTypeUserLoggedIn,
				Severity:  audit.SeverityInfo,
				UserID:    &userID,
			},
			{
				ID:        uuid.New(),
				EventType: audit.EventTypeUserUpdated,
				Severity:  audit.SeverityInfo,
				UserID:    &userID,
			},
		}

		mockLogRepo.On("List", ctx, filter).Return(logs, int64(2), nil)

		result, count, err := auditService.GetLogs(ctx, filter)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(2), count)

		mockLogRepo.AssertExpectations(t)
	})

	t.Run("empty results", func(t *testing.T) {
		mockLogRepo := new(MockLogRepository)

		auditService := services.NewAuditService(
			mockLogRepo,
			nil,
			nil,
			nil,
		)

		filter := audit.LogFilter{
			Limit:  10,
			Offset: 0,
		}

		mockLogRepo.On("List", ctx, filter).Return([]*audit.LogEntry{}, int64(0), nil)

		result, count, err := auditService.GetLogs(ctx, filter)

		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, int64(0), count)

		mockLogRepo.AssertExpectations(t)
	})
}

func TestAuditService_PurgeOldLogs(t *testing.T) {
	ctx := context.Background()

	t.Run("successful purge", func(t *testing.T) {
		mockLogRepo := new(MockLogRepository)

		auditService := services.NewAuditService(
			mockLogRepo,
			nil,
			nil,
			nil,
		)

		retentionDays := 90

		mockLogRepo.On("DeleteOlderThan", ctx, mock.AnythingOfType("time.Time")).Return(int64(100), nil)

		count, err := auditService.PurgeOldLogs(ctx, retentionDays)

		require.NoError(t, err)
		assert.Equal(t, int64(100), count)

		mockLogRepo.AssertExpectations(t)
	})

	t.Run("no logs to purge", func(t *testing.T) {
		mockLogRepo := new(MockLogRepository)

		auditService := services.NewAuditService(
			mockLogRepo,
			nil,
			nil,
			nil,
		)

		retentionDays := 30

		mockLogRepo.On("DeleteOlderThan", ctx, mock.AnythingOfType("time.Time")).Return(int64(0), nil)

		count, err := auditService.PurgeOldLogs(ctx, retentionDays)

		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		mockLogRepo.AssertExpectations(t)
	})
}
