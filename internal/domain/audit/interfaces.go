package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type LogRepository interface {
	Create(ctx context.Context, entry *LogEntry) error

	GetByID(ctx context.Context, id uuid.UUID) (*LogEntry, error)

	List(ctx context.Context, filter LogFilter) ([]*LogEntry, int64, error)

	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*LogEntry, int64, error)

	GetByEntityID(ctx context.Context, entityType, entityID string, limit, offset int) ([]*LogEntry, int64, error)

	GetSummary(ctx context.Context, filter LogFilter) (*LogSummary, error)

	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)

	Export(ctx context.Context, filter LogFilter, format string) ([]byte, error)
}

type AlertRepository interface {
	CreateRule(ctx context.Context, rule *AlertRule) error

	GetRuleByID(ctx context.Context, id uuid.UUID) (*AlertRule, error)

	ListRules(ctx context.Context, active bool) ([]*AlertRule, error)

	UpdateRule(ctx context.Context, rule *AlertRule) error

	DeleteRule(ctx context.Context, id uuid.UUID) error

	CheckRules(ctx context.Context, entry *LogEntry) ([]*AlertRule, error)
}

type ComplianceRepository interface {
	CreateReport(ctx context.Context, report *ComplianceReport) error

	GetReportByID(ctx context.Context, id uuid.UUID) (*ComplianceReport, error)

	ListReports(ctx context.Context, reportType string, limit, offset int) ([]*ComplianceReport, int64, error)

	UpdateReport(ctx context.Context, report *ComplianceReport) error
}

type AuditService interface {
	Log(ctx context.Context, req *CreateLogRequest) (*LogEntry, error)

	LogUserEvent(ctx context.Context, userID uuid.UUID, eventType EventType, description string, metadata map[string]interface{}) error

	LogSecurityEvent(ctx context.Context, eventType EventType, severity Severity, description string, metadata map[string]interface{}) error

	GetLog(ctx context.Context, id uuid.UUID) (*LogEntry, error)

	GetLogs(ctx context.Context, filter LogFilter) ([]*LogEntry, int64, error)

	GetUserLogs(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*LogEntry, int64, error)

	GetEntityLogs(ctx context.Context, entityType, entityID string, limit, offset int) ([]*LogEntry, int64, error)

	GetLogSummary(ctx context.Context, filter LogFilter) (*LogSummary, error)

	ExportLogs(ctx context.Context, req *ExportRequest) (*ExportResponse, error)

	CreateAlertRule(ctx context.Context, rule *AlertRule) error

	UpdateAlertRule(ctx context.Context, rule *AlertRule) error

	DeleteAlertRule(ctx context.Context, id uuid.UUID) error

	GetAlertRules(ctx context.Context, active bool) ([]*AlertRule, error)

	GenerateComplianceReport(ctx context.Context, reportType string, startDate, endDate time.Time) (*ComplianceReport, error)

	GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error)

	ListComplianceReports(ctx context.Context, reportType string, limit, offset int) ([]*ComplianceReport, int64, error)

	PurgeOldLogs(ctx context.Context, retentionDays int) (int64, error)

	GetConfig(ctx context.Context) (*AuditConfig, error)

	UpdateConfig(ctx context.Context, config *AuditConfig) error
}

type NotificationService interface {
	SendAlert(ctx context.Context, rule *AlertRule, entry *LogEntry) error

	SendComplianceReport(ctx context.Context, report *ComplianceReport, recipients []string) error
}
