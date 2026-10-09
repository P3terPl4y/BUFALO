package dbresilience

import (
	"github.com/jackc/pgx/v5"
	"testing"
)

func TestBoundedDSNPreservesCredentialsAndFailover(t *testing.T) {
	for _, raw := range []string{"", "postgres://u:p@a:5432,b:5432/db?sslmode=require&target_session_attrs=read-write", "host=a,b user=u password='a b' dbname=db sslmode=require"} {
		values := map[string]string{"DB_DSN": raw, "DB_HOST": "127.0.0.1", "DB_PORT": "5432", "DB_USERNAME": "u", "DB_PASSWORD": "a b'@", "DB_DATABASE": "db"}
		dsn, err := BoundedDSN(func(k string) string { return values[k] }, false)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := pgx.ParseConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		if parsed.RuntimeParams["statement_timeout"] != "5000" || parsed.RuntimeParams["idle_in_transaction_session_timeout"] != "10000" {
			t.Fatal("timeouts missing")
		}
		if raw == "" && parsed.Password != values["DB_PASSWORD"] {
			t.Fatal("password changed")
		}
		if raw != "" && len(parsed.Fallbacks) == 0 {
			t.Fatal("failover hosts removed")
		}
	}
}
