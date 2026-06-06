// Package config loads the calendar backend's environment configuration (12-factor).
package config

import (
	"fmt"
	"os"
)

// Config is the calendar backend's runtime configuration.
type Config struct {
	Addr         string // listen address, e.g. ":9090"
	DatabaseURL  string // Postgres connection string (the extension's OWN database)
	CoreJWKSURL  string // core's JWKS endpoint used to verify delegated tokens
	CoreIssuer   string // expected `iss` claim — core's public URL
	ExtensionID  string // expected `aud` claim — this extension's id
	NatsURL      string // optional: core's NATS URL for the opt-in fast path (GDPR teardown). Empty disables it.
	OTLPEndpoint string // optional: OTLP/HTTP endpoint for traces+metrics. Empty makes telemetry a no-op.
}

// Load reads the configuration from the environment and fails fast on missing required values.
func Load() (Config, error) {
	c := Config{
		Addr:         getenv("CALENDAR_ADDR", ":9090"),
		DatabaseURL:  os.Getenv("CALENDAR_DATABASE_URL"),
		CoreJWKSURL:  os.Getenv("CALENDAR_CORE_JWKS_URL"),
		CoreIssuer:   os.Getenv("CALENDAR_CORE_ISSUER"),
		ExtensionID:  getenv("CALENDAR_EXTENSION_ID", "net.flytegration.calendar"),
		NatsURL:      os.Getenv("CALENDAR_NATS_URL"),
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("CALENDAR_DATABASE_URL is required")
	}
	if c.CoreJWKSURL == "" || c.CoreIssuer == "" {
		return Config{}, fmt.Errorf("CALENDAR_CORE_JWKS_URL and CALENDAR_CORE_ISSUER are required")
	}
	return c, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
