package analytics

import "errors"

var (
	// ErrEventNotFound is returned when an event is not found
	ErrEventNotFound = errors.New("event not found")

	// ErrMetricNotFound is returned when a metric is not found
	ErrMetricNotFound = errors.New("metric not found")

	// ErrInvalidEventType is returned when event type is invalid
	ErrInvalidEventType = errors.New("invalid event type")

	// ErrInvalidMetricType is returned when metric type is invalid
	ErrInvalidMetricType = errors.New("invalid metric type")

	// ErrInvalidTimeRange is returned when time range is invalid
	ErrInvalidTimeRange = errors.New("invalid time range")

	// ErrInvalidFilter is returned when filter parameters are invalid
	ErrInvalidFilter = errors.New("invalid filter parameters")

	// ErrEventRequired is returned when event is nil
	ErrEventRequired = errors.New("event is required")

	// ErrMetricRequired is returned when metric is nil
	ErrMetricRequired = errors.New("metric is required")

	// ErrInvalidPeriod is returned when period is invalid
	ErrInvalidPeriod = errors.New("invalid period")

	// ErrExportFailed is returned when data export fails
	ErrExportFailed = errors.New("data export failed")

	// ErrRepositoryUnavailable is returned when repository is unavailable
	ErrRepositoryUnavailable = errors.New("repository unavailable")
)
