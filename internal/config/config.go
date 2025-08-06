package config

import (
	"time"
)

// Config holds the application configuration
type Config struct {
	// Server settings
	Port        int
	Environment string
	Version     string
	StartTime   time.Time

	// CORS settings
	CORS CORSConfig

	// Rate limiting
	RateLimit RateLimitConfig

	// API info
	DocsURL       string
	SupportEmail  string
	StatusPageURL string

	// Metrics
	Metrics MetricsConfig
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Global  int // requests per minute
	PerIP   int
	PerUser int
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool
	Path    string
	Port    int
}
