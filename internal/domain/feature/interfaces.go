package feature

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type FlagRepository interface {
	Create(ctx context.Context, flag *Flag) error

	GetByID(ctx context.Context, id uuid.UUID) (*Flag, error)

	GetByKey(ctx context.Context, key string) (*Flag, error)

	List(ctx context.Context, limit, offset int) ([]*Flag, int64, error)

	ListByTags(ctx context.Context, tags []string) ([]*Flag, error)

	Update(ctx context.Context, flag *Flag) error

	Delete(ctx context.Context, id uuid.UUID) error

	Archive(ctx context.Context, id uuid.UUID) error
}

type RuleRepository interface {
	CreateRule(ctx context.Context, flagID uuid.UUID, rule *Rule) error

	GetRuleByID(ctx context.Context, id uuid.UUID) (*Rule, error)

	ListRulesByFlag(ctx context.Context, flagID uuid.UUID) ([]Rule, error)

	UpdateRule(ctx context.Context, rule *Rule) error

	DeleteRule(ctx context.Context, id uuid.UUID) error

	ReorderRules(ctx context.Context, flagID uuid.UUID, ruleIDs []uuid.UUID) error
}

type OverrideRepository interface {
	Create(ctx context.Context, override *Override) error

	GetByID(ctx context.Context, id uuid.UUID) (*Override, error)

	GetByUser(ctx context.Context, flagID, userID uuid.UUID) (*Override, error)

	GetByGroup(ctx context.Context, flagID, groupID uuid.UUID) (*Override, error)

	ListByFlag(ctx context.Context, flagID uuid.UUID) ([]*Override, error)

	Update(ctx context.Context, override *Override) error

	Delete(ctx context.Context, id uuid.UUID) error

	DeleteExpired(ctx context.Context) (int64, error)
}

type SegmentRepository interface {
	Create(ctx context.Context, segment *Segment) error

	GetByID(ctx context.Context, id uuid.UUID) (*Segment, error)

	GetByKey(ctx context.Context, key string) (*Segment, error)

	List(ctx context.Context, limit, offset int) ([]*Segment, int64, error)

	Update(ctx context.Context, segment *Segment) error

	Delete(ctx context.Context, id uuid.UUID) error

	IsUserInSegment(ctx context.Context, segmentID, userID uuid.UUID) (bool, error)
}

type EventRepository interface {
	Record(ctx context.Context, event *Event) error

	GetEvents(ctx context.Context, flagKey string, from, to time.Time) ([]*Event, error)

	GetAnalytics(ctx context.Context, flagKey string, from, to time.Time) (*Analytics, error)

	GetUserEvents(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Event, int64, error)

	DeleteOldEvents(ctx context.Context, before time.Time) (int64, error)
}

type ChangeLogRepository interface {
	Record(ctx context.Context, change *ChangeLog) error

	GetByFlag(ctx context.Context, flagID uuid.UUID, limit, offset int) ([]*ChangeLog, int64, error)

	GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*ChangeLog, int64, error)

	GetRecent(ctx context.Context, limit int) ([]*ChangeLog, error)
}

type FeatureService interface {
	CreateFlag(ctx context.Context, req *CreateFlagRequest, createdBy uuid.UUID) (*Flag, error)

	UpdateFlag(ctx context.Context, id uuid.UUID, req *UpdateFlagRequest, updatedBy uuid.UUID) (*Flag, error)

	DeleteFlag(ctx context.Context, id uuid.UUID) error

	GetFlag(ctx context.Context, id uuid.UUID) (*Flag, error)

	GetFlagByKey(ctx context.Context, key string) (*Flag, error)

	ListFlags(ctx context.Context, limit, offset int) ([]*Flag, int64, error)

	Evaluate(ctx context.Context, req *EvaluationRequest) (*Evaluation, error)

	BatchEvaluate(ctx context.Context, req *BatchEvaluationRequest) (*BatchEvaluationResponse, error)

	GetFlagState(ctx context.Context, context Context) (*FlagState, error)

	CreateOverride(ctx context.Context, override *Override) error

	RemoveOverride(ctx context.Context, id uuid.UUID) error

	CreateSegment(ctx context.Context, segment *Segment) error

	UpdateSegment(ctx context.Context, segment *Segment) error

	DeleteSegment(ctx context.Context, id uuid.UUID) error

	GetSegment(ctx context.Context, id uuid.UUID) (*Segment, error)

	ListSegments(ctx context.Context, limit, offset int) ([]*Segment, int64, error)

	CreateExperiment(ctx context.Context, experiment *Experiment) error

	UpdateExperiment(ctx context.Context, experiment *Experiment) error

	EndExperiment(ctx context.Context, id uuid.UUID) error

	GetExperiment(ctx context.Context, id uuid.UUID) (*Experiment, error)

	GetAnalytics(ctx context.Context, flagKey string, from, to time.Time) (*Analytics, error)

	GetChangeLog(ctx context.Context, flagID uuid.UUID, limit, offset int) ([]*ChangeLog, int64, error)

	RecordEvent(ctx context.Context, event *Event) error

	CleanupExpiredData(ctx context.Context) error
}

type CacheService interface {
	Get(ctx context.Context, key string) (interface{}, error)

	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	Delete(ctx context.Context, key string) error

	Clear(ctx context.Context) error
}

type EvaluationEngine interface {
	Evaluate(flag *Flag, context Context) (*Evaluation, error)

	EvaluateRule(rule Rule, context Context) (bool, error)

	EvaluateCondition(condition Condition, context Context) (bool, error)

	GetVariant(variants []Variant, userID uuid.UUID) *Variant
}
