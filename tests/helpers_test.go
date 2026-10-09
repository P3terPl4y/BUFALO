package tests

import "testing"

func TestValidateTestDatabaseChecksEffectiveDSNDatabase(t *testing.T) {
	const configured = "bufalo_validation_test"
	for _, testCase := range []struct {
		name string
		dsn  string
		want bool
	}{
		{"without DSN", "", true},
		{"same isolated database", "postgres://test:secret@localhost:5432/bufalo_validation_test?sslmode=disable", true},
		{"DSN overrides to production", "postgres://test:secret@localhost:5432/bufalo?sslmode=disable", false},
		{"DSN malformed", "not-a-postgres-dsn", false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateTestDatabase("testing", configured, configured, testCase.dsn)
			if (err == nil) != testCase.want {
				t.Fatalf("ValidateTestDatabase() error=%v, want success=%t", err, testCase.want)
			}
		})
	}
}
