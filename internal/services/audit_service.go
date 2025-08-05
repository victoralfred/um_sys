package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/audit"
)

type AuditService struct {
	logRepo         audit.LogRepository
	alertRepo       audit.AlertRepository
	complianceRepo  audit.ComplianceRepository
	notificationSvc audit.NotificationService
	config          *audit.AuditConfig
}

func NewAuditService(
	logRepo audit.LogRepository,
	alertRepo audit.AlertRepository,
	complianceRepo audit.ComplianceRepository,
	notificationSvc audit.NotificationService,
) *AuditService {
	return &AuditService{
		logRepo:         logRepo,
		alertRepo:       alertRepo,
		complianceRepo:  complianceRepo,
		notificationSvc: notificationSvc,
		config: &audit.AuditConfig{
			Enabled:       true,
			RetentionDays: 90,
		},
	}
}

func (s *AuditService) Log(ctx context.Context, req *audit.CreateLogRequest) (*audit.LogEntry, error) {
	if !s.config.Enabled {
		return nil, audit.ErrAuditDisabled
	}

	if req.EntityType == "" || req.EntityID == "" || req.Action == "" {
		return nil, audit.ErrMissingRequiredFields
	}

	now := time.Now()
	entry := &audit.LogEntry{
		ID:          uuid.New(),
		Timestamp:   now,
		EventType:   req.EventType,
		Severity:    req.Severity,
		UserID:      req.UserID,
		ActorID:     req.ActorID,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		Action:      req.Action,
		Description: req.Description,
		IPAddress:   req.IPAddress,
		UserAgent:   req.UserAgent,
		Metadata:    req.Metadata,
		Changes:     req.Changes,
		RequestID:   req.RequestID,
		SessionID:   req.SessionID,
		TraceID:     req.TraceID,
		CreatedAt:   now,
	}

	if err := s.logRepo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to create log entry: %w", err)
	}

	triggeredRules, err := s.alertRepo.CheckRules(ctx, entry)
	if err == nil && len(triggeredRules) > 0 {
		for _, rule := range triggeredRules {
			if s.notificationSvc != nil {
				_ = s.notificationSvc.SendAlert(ctx, rule, entry)
			}
		}
	}

	return entry, nil
}

func (s *AuditService) LogUserEvent(ctx context.Context, userID uuid.UUID, eventType audit.EventType, description string, metadata map[string]interface{}) error {
	req := &audit.CreateLogRequest{
		EventType:   eventType,
		Severity:    audit.SeverityInfo,
		UserID:      &userID,
		EntityType:  "user",
		EntityID:    userID.String(),
		Action:      string(eventType),
		Description: description,
		Metadata:    metadata,
	}

	_, err := s.Log(ctx, req)
	return err
}

func (s *AuditService) LogSecurityEvent(ctx context.Context, eventType audit.EventType, severity audit.Severity, description string, metadata map[string]interface{}) error {
	req := &audit.CreateLogRequest{
		EventType:   eventType,
		Severity:    severity,
		EntityType:  "system",
		EntityID:    "security",
		Action:      string(eventType),
		Description: description,
		Metadata:    metadata,
	}

	_, err := s.Log(ctx, req)
	return err
}

func (s *AuditService) GetLog(ctx context.Context, id uuid.UUID) (*audit.LogEntry, error) {
	return s.logRepo.GetByID(ctx, id)
}

func (s *AuditService) GetLogs(ctx context.Context, filter audit.LogFilter) ([]*audit.LogEntry, int64, error) {
	return s.logRepo.List(ctx, filter)
}

func (s *AuditService) GetUserLogs(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*audit.LogEntry, int64, error) {
	return s.logRepo.GetByUserID(ctx, userID, limit, offset)
}

func (s *AuditService) GetEntityLogs(ctx context.Context, entityType, entityID string, limit, offset int) ([]*audit.LogEntry, int64, error) {
	return s.logRepo.GetByEntityID(ctx, entityType, entityID, limit, offset)
}

func (s *AuditService) GetLogSummary(ctx context.Context, filter audit.LogFilter) (*audit.LogSummary, error) {
	return s.logRepo.GetSummary(ctx, filter)
}

func (s *AuditService) ExportLogs(ctx context.Context, req *audit.ExportRequest) (*audit.ExportResponse, error) {
	_, err := s.logRepo.Export(ctx, req.Filter, req.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to export logs: %w", err)
	}

	response := &audit.ExportResponse{
		ID:        uuid.New(),
		Status:    "completed",
		URL:       fmt.Sprintf("/exports/%s.%s", uuid.New().String(), req.Format),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return response, nil
}

func (s *AuditService) CreateAlertRule(ctx context.Context, rule *audit.AlertRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	return s.alertRepo.CreateRule(ctx, rule)
}

func (s *AuditService) UpdateAlertRule(ctx context.Context, rule *audit.AlertRule) error {
	existing, err := s.alertRepo.GetRuleByID(ctx, rule.ID)
	if err != nil {
		return err
	}

	rule.CreatedAt = existing.CreatedAt
	rule.UpdatedAt = time.Now()

	return s.alertRepo.UpdateRule(ctx, rule)
}

func (s *AuditService) DeleteAlertRule(ctx context.Context, id uuid.UUID) error {
	return s.alertRepo.DeleteRule(ctx, id)
}

func (s *AuditService) GetAlertRules(ctx context.Context, active bool) ([]*audit.AlertRule, error) {
	return s.alertRepo.ListRules(ctx, active)
}

func (s *AuditService) GenerateComplianceReport(ctx context.Context, reportType string, startDate, endDate time.Time) (*audit.ComplianceReport, error) {
	filter := audit.LogFilter{
		StartTime: startDate,
		EndTime:   endDate,
	}

	summary, err := s.logRepo.GetSummary(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get log summary: %w", err)
	}

	report := &audit.ComplianceReport{
		ID:         uuid.New(),
		ReportType: reportType,
		StartDate:  startDate,
		EndDate:    endDate,
		Status:     "completed",
		Summary: map[string]interface{}{
			"total_events":       summary.TotalEvents,
			"events_by_type":     summary.EventsByType,
			"events_by_severity": summary.EventsBySeverity,
			"unique_users":       summary.UniqueUsers,
			"unique_ips":         summary.UniqueIPs,
		},
		GeneratedAt: time.Now(),
	}

	if s.complianceRepo != nil {
		if err := s.complianceRepo.CreateReport(ctx, report); err != nil {
			return nil, fmt.Errorf("failed to save compliance report: %w", err)
		}
	}

	return report, nil
}

func (s *AuditService) GetComplianceReport(ctx context.Context, id uuid.UUID) (*audit.ComplianceReport, error) {
	if s.complianceRepo == nil {
		return nil, audit.ErrReportNotFound
	}
	return s.complianceRepo.GetReportByID(ctx, id)
}

func (s *AuditService) ListComplianceReports(ctx context.Context, reportType string, limit, offset int) ([]*audit.ComplianceReport, int64, error) {
	if s.complianceRepo == nil {
		return nil, 0, nil
	}
	return s.complianceRepo.ListReports(ctx, reportType, limit, offset)
}

func (s *AuditService) PurgeOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	before := time.Now().AddDate(0, 0, -retentionDays)
	return s.logRepo.DeleteOlderThan(ctx, before)
}

func (s *AuditService) GetConfig(ctx context.Context) (*audit.AuditConfig, error) {
	return s.config, nil
}

func (s *AuditService) UpdateConfig(ctx context.Context, config *audit.AuditConfig) error {
	s.config = config
	return nil
}
