package config_test

import (
	"testing"

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/config"
)

// allKeys are every variable Load reads; tests set the full set (using "" for absent) so a case is
// deterministic regardless of the ambient environment.
var allKeys = []string{
	"CALENDAR_ADDR",
	"CALENDAR_DATABASE_URL",
	"CALENDAR_CORE_JWKS_URL",
	"CALENDAR_CORE_ISSUER",
	"CALENDAR_EXTENSION_ID",
}

func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for _, k := range allKeys {
		t.Setenv(k, env[k])
	}
}

func requiredEnv() map[string]string {
	return map[string]string{
		"CALENDAR_DATABASE_URL":  "postgres://localhost/calendar",
		"CALENDAR_CORE_JWKS_URL": "https://core.test/.well-known/jwks.json",
		"CALENDAR_CORE_ISSUER":   "https://core.test",
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	setEnv(t, requiredEnv())

	c, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.Addr != ":9090" {
		t.Errorf("Addr = %q, want :9090", c.Addr)
	}
	if c.ExtensionID != "net.flytegration.calendar" {
		t.Errorf("ExtensionID = %q, want net.flytegration.calendar", c.ExtensionID)
	}
	if c.DatabaseURL != "postgres://localhost/calendar" {
		t.Errorf("DatabaseURL = %q", c.DatabaseURL)
	}
}

func TestLoadHonorsOverrides(t *testing.T) {
	env := requiredEnv()
	env["CALENDAR_ADDR"] = ":8080"
	env["CALENDAR_EXTENSION_ID"] = "net.flytegration.calendar.staging"
	setEnv(t, env)

	c, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", c.Addr)
	}
	if c.ExtensionID != "net.flytegration.calendar.staging" {
		t.Errorf("ExtensionID = %q", c.ExtensionID)
	}
}

func TestLoadFailsFastOnMissingRequired(t *testing.T) {
	tests := []struct {
		name string
		drop string
	}{
		{name: "missing database url", drop: "CALENDAR_DATABASE_URL"},
		{name: "missing jwks url", drop: "CALENDAR_CORE_JWKS_URL"},
		{name: "missing issuer", drop: "CALENDAR_CORE_ISSUER"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := requiredEnv()
			delete(env, tt.drop)
			setEnv(t, env)

			if _, err := config.Load(); err == nil {
				t.Fatalf("Load succeeded with %s absent, want error", tt.drop)
			}
		})
	}
}
