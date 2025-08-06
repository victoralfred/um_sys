package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/feature"
)

type FeatureService struct {
	flagRepo      feature.FlagRepository
	ruleRepo      feature.RuleRepository
	overrideRepo  feature.OverrideRepository
	segmentRepo   feature.SegmentRepository
	eventRepo     feature.EventRepository
	changeLogRepo feature.ChangeLogRepository
	cacheService  feature.CacheService
	evalEngine    feature.EvaluationEngine
}

func NewFeatureService(
	flagRepo feature.FlagRepository,
	ruleRepo feature.RuleRepository,
	overrideRepo feature.OverrideRepository,
	segmentRepo feature.SegmentRepository,
	eventRepo feature.EventRepository,
	changeLogRepo feature.ChangeLogRepository,
	cacheService feature.CacheService,
	evalEngine feature.EvaluationEngine,
) *FeatureService {
	return &FeatureService{
		flagRepo:      flagRepo,
		ruleRepo:      ruleRepo,
		overrideRepo:  overrideRepo,
		segmentRepo:   segmentRepo,
		eventRepo:     eventRepo,
		changeLogRepo: changeLogRepo,
		cacheService:  cacheService,
		evalEngine:    evalEngine,
	}
}

func (s *FeatureService) CreateFlag(ctx context.Context, req *feature.CreateFlagRequest, createdBy uuid.UUID) (*feature.Flag, error) {
	existing, _ := s.flagRepo.GetByKey(ctx, req.Key)
	if existing != nil {
		return nil, feature.ErrFlagAlreadyExists
	}

	now := time.Now()
	flag := &feature.Flag{
		ID:           uuid.New(),
		Key:          req.Key,
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		DefaultValue: req.DefaultValue,
		IsEnabled:    req.IsEnabled,
		Rules:        req.Rules,
		Tags:         req.Tags,
		Metadata:     req.Metadata,
		CreatedAt:    now,
		UpdatedAt:    now,
		CreatedBy:    createdBy,
		UpdatedBy:    createdBy,
	}

	if err := s.flagRepo.Create(ctx, flag); err != nil {
		return nil, fmt.Errorf("failed to create flag: %w", err)
	}

	if s.changeLogRepo != nil {
		after, _ := json.Marshal(flag)
		change := &feature.ChangeLog{
			ID:        uuid.New(),
			FlagID:    flag.ID,
			FlagKey:   flag.Key,
			Action:    "created",
			After:     after,
			Reason:    "Flag created",
			ChangedBy: createdBy,
			ChangedAt: now,
		}
		_ = s.changeLogRepo.Record(ctx, change)
	}

	return flag, nil
}

func (s *FeatureService) UpdateFlag(ctx context.Context, id uuid.UUID, req *feature.UpdateFlagRequest, updatedBy uuid.UUID) (*feature.Flag, error) {
	flag, err := s.flagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var before []byte
	if s.changeLogRepo != nil {
		before, _ = json.Marshal(flag)
	}

	if req.Name != nil {
		flag.Name = *req.Name
	}
	if req.Description != nil {
		flag.Description = *req.Description
	}
	if req.DefaultValue != nil {
		flag.DefaultValue = req.DefaultValue
	}
	if req.IsEnabled != nil {
		flag.IsEnabled = *req.IsEnabled
	}
	if req.Rules != nil {
		flag.Rules = req.Rules
	}
	if req.Tags != nil {
		flag.Tags = req.Tags
	}
	if req.Metadata != nil {
		flag.Metadata = req.Metadata
	}

	flag.UpdatedAt = time.Now()
	flag.UpdatedBy = updatedBy

	if err := s.flagRepo.Update(ctx, flag); err != nil {
		return nil, fmt.Errorf("failed to update flag: %w", err)
	}

	if s.changeLogRepo != nil && before != nil {
		after, _ := json.Marshal(flag)
		change := &feature.ChangeLog{
			ID:        uuid.New(),
			FlagID:    flag.ID,
			FlagKey:   flag.Key,
			Action:    "updated",
			Before:    before,
			After:     after,
			Reason:    "Flag updated",
			ChangedBy: updatedBy,
			ChangedAt: flag.UpdatedAt,
		}
		_ = s.changeLogRepo.Record(ctx, change)
	}

	if s.cacheService != nil {
		_ = s.cacheService.Delete(ctx, fmt.Sprintf("flag:%s", flag.Key))
	}

	return flag, nil
}

func (s *FeatureService) DeleteFlag(ctx context.Context, id uuid.UUID) error {
	flag, err := s.flagRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.flagRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete flag: %w", err)
	}

	if s.cacheService != nil {
		_ = s.cacheService.Delete(ctx, fmt.Sprintf("flag:%s", flag.Key))
	}

	return nil
}

func (s *FeatureService) GetFlag(ctx context.Context, id uuid.UUID) (*feature.Flag, error) {
	return s.flagRepo.GetByID(ctx, id)
}

func (s *FeatureService) GetFlagByKey(ctx context.Context, key string) (*feature.Flag, error) {
	if s.cacheService != nil {
		cacheKey := fmt.Sprintf("flag:%s", key)
		if cached, err := s.cacheService.Get(ctx, cacheKey); err == nil {
			if flag, ok := cached.(*feature.Flag); ok {
				return flag, nil
			}
		}
	}

	flag, err := s.flagRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	if s.cacheService != nil {
		cacheKey := fmt.Sprintf("flag:%s", key)
		_ = s.cacheService.Set(ctx, cacheKey, flag, 5*time.Minute)
	}

	return flag, nil
}

func (s *FeatureService) ListFlags(ctx context.Context, limit, offset int) ([]*feature.Flag, int64, error) {
	return s.flagRepo.List(ctx, limit, offset)
}

func (s *FeatureService) Evaluate(ctx context.Context, req *feature.EvaluationRequest) (*feature.Evaluation, error) {
	flag, err := s.GetFlagByKey(ctx, req.FlagKey)
	if err != nil {
		return nil, err
	}

	if !flag.IsEnabled {
		return nil, feature.ErrFlagDisabled
	}

	evaluation := &feature.Evaluation{
		FlagKey:   req.FlagKey,
		Value:     flag.DefaultValue,
		IsDefault: true,
		Reason:    "Default value",
		Timestamp: time.Now(),
	}

	if req.Context.UserID != nil {
		override, err := s.overrideRepo.GetByUser(ctx, flag.ID, *req.Context.UserID)
		if err == nil && override != nil {
			if override.ExpiresAt == nil || override.ExpiresAt.After(time.Now()) {
				evaluation.Value = override.Value
				evaluation.IsDefault = false
				evaluation.Reason = "User override"

				s.recordEvent(ctx, flag.Key, &req.Context, evaluation.Value)
				return evaluation, nil
			}
		}
	}

	if req.Context.GroupID != nil {
		override, err := s.overrideRepo.GetByGroup(ctx, flag.ID, *req.Context.GroupID)
		if err == nil && override != nil {
			if override.ExpiresAt == nil || override.ExpiresAt.After(time.Now()) {
				evaluation.Value = override.Value
				evaluation.IsDefault = false
				evaluation.Reason = "Group override"

				s.recordEvent(ctx, flag.Key, &req.Context, evaluation.Value)
				return evaluation, nil
			}
		}
	}

	if s.evalEngine != nil && len(flag.Rules) > 0 {
		engineEval, err := s.evalEngine.Evaluate(flag, req.Context)
		if err == nil && engineEval != nil {
			evaluation = engineEval
		}
	}

	s.recordEvent(ctx, flag.Key, &req.Context, evaluation.Value)
	return evaluation, nil
}

func (s *FeatureService) BatchEvaluate(ctx context.Context, req *feature.BatchEvaluationRequest) (*feature.BatchEvaluationResponse, error) {
	response := &feature.BatchEvaluationResponse{
		Evaluations: make(map[string]*feature.Evaluation),
	}

	for _, flagKey := range req.FlagKeys {
		evalReq := &feature.EvaluationRequest{
			FlagKey: flagKey,
			Context: req.Context,
		}

		eval, err := s.Evaluate(ctx, evalReq)
		if err == nil {
			response.Evaluations[flagKey] = eval
		}
	}

	return response, nil
}

func (s *FeatureService) GetFlagState(ctx context.Context, context feature.Context) (*feature.FlagState, error) {
	flags, _, err := s.flagRepo.List(ctx, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list flags: %w", err)
	}

	state := &feature.FlagState{
		Flags: make(map[string]interface{}),
	}

	for _, flag := range flags {
		if flag.IsEnabled {
			evalReq := &feature.EvaluationRequest{
				FlagKey: flag.Key,
				Context: context,
			}

			eval, err := s.Evaluate(ctx, evalReq)
			if err == nil {
				state.Flags[flag.Key] = eval.Value
			}
		}
	}

	return state, nil
}

func (s *FeatureService) CreateOverride(ctx context.Context, override *feature.Override) error {
	if override.ID == uuid.Nil {
		override.ID = uuid.New()
	}
	override.CreatedAt = time.Now()

	return s.overrideRepo.Create(ctx, override)
}

func (s *FeatureService) RemoveOverride(ctx context.Context, id uuid.UUID) error {
	return s.overrideRepo.Delete(ctx, id)
}

func (s *FeatureService) CreateSegment(ctx context.Context, segment *feature.Segment) error {
	if segment.ID == uuid.Nil {
		segment.ID = uuid.New()
	}
	segment.CreatedAt = time.Now()
	segment.UpdatedAt = time.Now()

	if s.segmentRepo == nil {
		return fmt.Errorf("segment repository not configured")
	}

	return s.segmentRepo.Create(ctx, segment)
}

func (s *FeatureService) UpdateSegment(ctx context.Context, segment *feature.Segment) error {
	if s.segmentRepo == nil {
		return fmt.Errorf("segment repository not configured")
	}

	segment.UpdatedAt = time.Now()
	return s.segmentRepo.Update(ctx, segment)
}

func (s *FeatureService) DeleteSegment(ctx context.Context, id uuid.UUID) error {
	if s.segmentRepo == nil {
		return fmt.Errorf("segment repository not configured")
	}

	return s.segmentRepo.Delete(ctx, id)
}

func (s *FeatureService) GetSegment(ctx context.Context, id uuid.UUID) (*feature.Segment, error) {
	if s.segmentRepo == nil {
		return nil, fmt.Errorf("segment repository not configured")
	}

	return s.segmentRepo.GetByID(ctx, id)
}

func (s *FeatureService) ListSegments(ctx context.Context, limit, offset int) ([]*feature.Segment, int64, error) {
	if s.segmentRepo == nil {
		return nil, 0, fmt.Errorf("segment repository not configured")
	}

	return s.segmentRepo.List(ctx, limit, offset)
}

func (s *FeatureService) CreateExperiment(ctx context.Context, experiment *feature.Experiment) error {
	if experiment.ID == uuid.Nil {
		experiment.ID = uuid.New()
	}
	experiment.CreatedAt = time.Now()
	experiment.UpdatedAt = time.Now()

	return nil
}

func (s *FeatureService) UpdateExperiment(ctx context.Context, experiment *feature.Experiment) error {
	experiment.UpdatedAt = time.Now()
	return nil
}

func (s *FeatureService) EndExperiment(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *FeatureService) GetExperiment(ctx context.Context, id uuid.UUID) (*feature.Experiment, error) {
	return nil, feature.ErrExperimentNotFound
}

func (s *FeatureService) GetAnalytics(ctx context.Context, flagKey string, from, to time.Time) (*feature.Analytics, error) {
	if s.eventRepo == nil {
		return nil, fmt.Errorf("event repository not configured")
	}

	return s.eventRepo.GetAnalytics(ctx, flagKey, from, to)
}

func (s *FeatureService) GetChangeLog(ctx context.Context, flagID uuid.UUID, limit, offset int) ([]*feature.ChangeLog, int64, error) {
	if s.changeLogRepo == nil {
		return nil, 0, fmt.Errorf("change log repository not configured")
	}

	return s.changeLogRepo.GetByFlag(ctx, flagID, limit, offset)
}

func (s *FeatureService) RecordEvent(ctx context.Context, event *feature.Event) error {
	if s.eventRepo == nil {
		return nil
	}

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.Timestamp = time.Now()

	return s.eventRepo.Record(ctx, event)
}

func (s *FeatureService) CleanupExpiredData(ctx context.Context) error {
	var err error

	if s.overrideRepo != nil {
		_, err = s.overrideRepo.DeleteExpired(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete expired overrides: %w", err)
		}
	}

	if s.eventRepo != nil {
		before := time.Now().AddDate(0, 0, -30)
		_, err = s.eventRepo.DeleteOldEvents(ctx, before)
		if err != nil {
			return fmt.Errorf("failed to delete old events: %w", err)
		}
	}

	return nil
}

func (s *FeatureService) recordEvent(ctx context.Context, flagKey string, context *feature.Context, value interface{}) {
	if s.eventRepo == nil {
		return
	}

	event := &feature.Event{
		ID:        uuid.New(),
		FlagKey:   flagKey,
		Value:     value,
		Context:   *context,
		Timestamp: time.Now(),
	}

	if context.UserID != nil {
		event.UserID = context.UserID
	}
	if context.GroupID != nil {
		event.GroupID = context.GroupID
	}

	_ = s.eventRepo.Record(ctx, event)
}
