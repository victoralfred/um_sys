package audit

import "errors"

var (
	ErrLogNotFound              = errors.New("log entry not found")
	ErrInvalidFilter            = errors.New("invalid filter parameters")
	ErrExportFailed             = errors.New("log export failed")
	ErrAlertRuleNotFound        = errors.New("alert rule not found")
	ErrInvalidAlertRule         = errors.New("invalid alert rule")
	ErrReportNotFound           = errors.New("compliance report not found")
	ErrReportGenerationFailed   = errors.New("compliance report generation failed")
	ErrAuditDisabled            = errors.New("audit logging is disabled")
	ErrRetentionPolicyViolation = errors.New("retention policy violation")
	ErrInvalidEventType         = errors.New("invalid event type")
	ErrInvalidSeverity          = errors.New("invalid severity level")
	ErrMissingRequiredFields    = errors.New("missing required fields")
	ErrExportInProgress         = errors.New("export already in progress")
	ErrExportNotReady           = errors.New("export not ready")
	ErrExportExpired            = errors.New("export has expired")
)
