package feature

import "errors"

var (
	ErrFlagNotFound         = errors.New("feature flag not found")
	ErrFlagAlreadyExists    = errors.New("feature flag already exists")
	ErrInvalidFlagType      = errors.New("invalid flag type")
	ErrInvalidFlagValue     = errors.New("invalid flag value")
	ErrFlagDisabled         = errors.New("feature flag is disabled")
	ErrInvalidContext       = errors.New("invalid evaluation context")
	ErrRuleNotFound         = errors.New("rule not found")
	ErrInvalidRule          = errors.New("invalid rule configuration")
	ErrOverrideNotFound     = errors.New("override not found")
	ErrSegmentNotFound      = errors.New("segment not found")
	ErrSegmentAlreadyExists = errors.New("segment already exists")
	ErrExperimentNotFound   = errors.New("experiment not found")
	ErrExperimentInactive   = errors.New("experiment is not active")
	ErrInvalidVariant       = errors.New("invalid variant configuration")
	ErrEvaluationFailed     = errors.New("flag evaluation failed")
	ErrCacheMiss            = errors.New("cache miss")
	ErrInvalidOperator      = errors.New("invalid condition operator")
	ErrInvalidProperty      = errors.New("invalid context property")
	ErrCircularDependency   = errors.New("circular dependency detected")
)
