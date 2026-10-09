package dbresilience

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// BoundedDSN preserves multi-host/provider DSNs while applying server-side
// deadlines to every connection, including queries made through legacy ORM APIs.
func BoundedDSN(get func(string) string, migration bool) (string, error) {
	timeout := "5000"
	if migration {
		timeout = "120000"
	}
	raw := strings.TrimSpace(get("DB_DSN"))
	if raw == "" {
		host := get("DB_HOST")
		if host == "" {
			return "", nil
		}
		port := get("DB_PORT")
		if port == "" {
			port = "5432"
		}
		if _, err := strconv.Atoi(port); err != nil {
			return "", fmt.Errorf("invalid database port")
		}
		ssl := get("DB_SSLMODE")
		if ssl == "" {
			ssl = "disable"
		}
		schema := get("DB_SCHEMA")
		if schema == "" {
			schema = "public"
		}
		u := url.URL{Scheme: "postgres", User: url.UserPassword(get("DB_USERNAME"), get("DB_PASSWORD")), Host: net.JoinHostPort(host, port), Path: "/" + get("DB_DATABASE")}
		query := url.Values{"sslmode": {ssl}, "search_path": {schema}}
		u.RawQuery = query.Encode()
		raw = u.String()
	}
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("invalid database DSN")
		}
		q := u.Query()
		q.Set("statement_timeout", timeout)
		q.Set("idle_in_transaction_session_timeout", "10000")
		q.Set("connect_timeout", "5")
		u.RawQuery = q.Encode()
		return u.String(), nil
	}
	return raw + " statement_timeout=" + timeout + " idle_in_transaction_session_timeout=10000 connect_timeout=5", nil
}
