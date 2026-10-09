package dbresilience

import "testing"

func TestParsePoolSettingsUsesBoundedRecoveryDefaults(t *testing.T) {
	got, err := ParsePoolSettings(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	want := DefaultPoolSettings()
	if got != want {
		t.Fatalf("settings=%+v, want %+v", got, want)
	}
}

func TestParsePoolSettingsAllowsSafeTuning(t *testing.T) {
	values := map[string]string{
		"DB_POOL_MAX_IDLE_CONNS":            "4",
		"DB_POOL_MAX_OPEN_CONNS":            "12",
		"DB_POOL_CONN_MAX_IDLE_SECONDS":     "120",
		"DB_POOL_CONN_MAX_LIFETIME_SECONDS": "900",
	}
	got, err := ParsePoolSettings(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	want := PoolSettings{MaxIdleConns: 4, MaxOpenConns: 12, ConnMaxIdleSeconds: 120, ConnMaxLifeSeconds: 900}
	if got != want {
		t.Fatalf("settings=%+v, want %+v", got, want)
	}
}

func TestParsePoolSettingsRejectsDangerousAndMalformedValues(t *testing.T) {
	for _, values := range []map[string]string{
		{"DB_POOL_MAX_OPEN_CONNS": "0"},
		{"DB_POOL_MAX_OPEN_CONNS": "501"},
		{"DB_POOL_MAX_IDLE_CONNS": "26"},
		{"DB_POOL_MAX_OPEN_CONNS": "not-a-number"},
		{"DB_POOL_CONN_MAX_LIFETIME_SECONDS": "1"},
	} {
		if _, err := ParsePoolSettings(func(key string) string { return values[key] }); err == nil {
			t.Errorf("ParsePoolSettings(%v) succeeded, want validation error", values)
		}
	}
}
