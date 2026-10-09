package main

import (
	"strings"
	"testing"
)

func TestValidateProductionConfig(t *testing.T) {
	valid := map[string]string{
		"APP_ENV":               "production",
		"APP_DEBUG":             "false",
		"APP_KEY":               "0123456789abcdef0123456789abcdef",
		"APP_URL":               "https://bufalo.test",
		"MAIL_HOST":             "smtp.bufalo.test",
		"MAIL_FROM_ADDRESS":     "no-reply@bufalo.test",
		"MAIL_PORT":             "587",
		"RATE_LIMIT_KEY_SECRET": "abcdef0123456789abcdef0123456789",
		"DB_HOST":               "postgres.internal",
		"DB_PORT":               "5432",
		"DB_DATABASE":           "bufalo",
		"DB_USERNAME":           "bufalo_app",
	}
	get := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}

	if err := validateProductionConfig(get(valid)); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
	haConfig := make(map[string]string, len(valid))
	for key, value := range valid {
		haConfig[key] = value
	}
	haConfig["DB_DSN"] = "postgres://bufalo:secret@pg-a:5432,pg-b:5432/bufalo?sslmode=require&connect_timeout=5&target_session_attrs=read-write"
	if err := validateProductionConfig(get(haConfig)); err != nil {
		t.Fatalf("valid multi-host DSN rejected: %v", err)
	}

	invalidValues := []struct{ key, value string }{
		{"TRUSTED_PROXY_IPS", "0.0.0.0/0"},
		{"TRUSTED_PROXY_IPS", "*"},
		{"TRUSTED_PROXY_IPS", "0.0.0.0"},
		{"APP_DEBUG", "true"},
		{"APP_KEY", "short"},
		{"APP_KEY", "0123456789abcdef0123456789abcdef-too-long"},
		{"APP_KEY", "replace-with-a-unique-random-secret"},
		{"RATE_LIMIT_KEY_SECRET", "short"},
		{"DB_POOL_MAX_OPEN_CONNS", "0"},
		{"DB_HOST", "replace-with-postgres-host"},
		{"DB_PORT", "70000"},
		{"DB_DSN", "replace-with-postgres-dsn"},
		{"APP_URL", "http://bufalo.example.com"},
		{"MAIL_HOST", "replace-with-smtp-host"},
		{"MAIL_FROM_ADDRESS", "not-an-email"},
		{"MAIL_PORT", "70000"},
		{"APP_URL", "https://bufalo.example.com"},
		{"MAIL_HOST", "smtp.example.com"},
		{"MAIL_FROM_ADDRESS", "no-reply@example.com"},
	}
	for _, testCase := range invalidValues {
		config := make(map[string]string, len(valid))
		for existing, existingValue := range valid {
			config[existing] = existingValue
		}
		config[testCase.key] = testCase.value
		if err := validateProductionConfig(get(config)); err == nil {
			t.Errorf("%s=%q should be rejected", testCase.key, testCase.value)
		}
	}

	local := map[string]string{"APP_ENV": "local", "APP_DEBUG": "true"}
	if err := validateProductionConfig(get(local)); err != nil {
		t.Fatalf("local development config should remain unrestricted: %v", err)
	}

	invalidDebug := map[string]string{"APP_ENV": "production", "APP_DEBUG": "sometimes", "APP_KEY": valid["APP_KEY"]}
	if err := validateProductionConfig(get(invalidDebug)); err == nil || !strings.Contains(err.Error(), "APP_DEBUG") {
		t.Fatalf("invalid APP_DEBUG should produce a specific error, got %v", err)
	}
}

func TestEnvironmentVariantsRemainSecure(t *testing.T) {
	for _, raw := range []string{"production", "Production", " PRODUCTION ", "staging"} {
		normalized, err := normalizeEnvironment(raw)
		if err != nil || !productionEnvironment(normalized) {
			t.Fatalf("unsafe variant %q", raw)
		}
		if err := validateProductionConfig(func(k string) string {
			if k == "APP_ENV" {
				return raw
			}
			return ""
		}); err == nil {
			t.Fatalf("%q bypasses production validation", raw)
		}
	}
	if _, err := normalizeEnvironment("prodution"); err == nil {
		t.Fatal("unknown environment accepted")
	}
}
