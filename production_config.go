package main

import (
	"fmt"
	"net/mail"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"goravel/app/dbresilience"
)

// validateProductionConfig rejects unsafe production settings before the
// process performs bootstrap writes or starts accepting traffic.
func validateProductionConfig(getenv func(string) string) error {
	environment, err := normalizeEnvironment(getenv("APP_ENV"))
	if err != nil {
		return err
	}
	if !productionEnvironment(environment) {
		return nil
	}
	if _, err := parseTrustedProxyIPs(getenv("TRUSTED_PROXY_IPS")); err != nil {
		return err
	}

	if raw := strings.TrimSpace(getenv("APP_DEBUG")); raw != "" {
		debug, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("APP_DEBUG must be a boolean in production")
		}
		if debug {
			return fmt.Errorf("APP_DEBUG must be false in production")
		}
	}

	appKey := strings.TrimSpace(getenv("APP_KEY"))
	if len(appKey) != 32 || containsPlaceholder(appKey) {
		return fmt.Errorf("APP_KEY must contain exactly 32 characters and must not be a placeholder in production")
	}
	rateKey := strings.TrimSpace(getenv("RATE_LIMIT_KEY_SECRET"))
	if rateKey == "" {
		rateKey = appKey
	}
	if !isConfiguredSecret(rateKey) {
		return fmt.Errorf("RATE_LIMIT_KEY_SECRET must contain at least 32 characters and must not be a placeholder in production")
	}
	appURL, err := url.Parse(strings.TrimSpace(getenv("APP_URL")))
	if err != nil || appURL.Scheme != "https" || appURL.Host == "" || appURL.User != nil || isExampleDomain(appURL.Hostname()) {
		return fmt.Errorf("APP_URL must be an absolute HTTPS URL in production")
	}
	mailHost := strings.TrimSpace(getenv("MAIL_HOST"))
	if mailHost == "" || containsPlaceholder(mailHost) || isExampleDomain(mailHost) {
		return fmt.Errorf("MAIL_HOST must be configured for email verification in production")
	}
	fromAddress := strings.TrimSpace(getenv("MAIL_FROM_ADDRESS"))
	parsedFrom, err := mail.ParseAddress(fromAddress)
	if err != nil || parsedFrom == nil {
		return fmt.Errorf("MAIL_FROM_ADDRESS must be a valid email address in production")
	}
	fromParts := strings.Split(parsedFrom.Address, "@")
	if parsedFrom.Address != fromAddress || len(fromParts) != 2 || isExampleDomain(fromParts[1]) {
		return fmt.Errorf("MAIL_FROM_ADDRESS must be a valid email address in production")
	}
	if rawPort := strings.TrimSpace(getenv("MAIL_PORT")); rawPort != "" {
		port, err := strconv.Atoi(rawPort)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("MAIL_PORT must be between 1 and 65535")
		}
	}
	dsn := strings.TrimSpace(getenv("DB_DSN"))
	if dsn != "" {
		lowerDSN := strings.ToLower(dsn)
		if containsPlaceholder(dsn) || !(strings.HasPrefix(lowerDSN, "postgres://") || strings.HasPrefix(lowerDSN, "postgresql://") || strings.HasPrefix(lowerDSN, "host=") || strings.HasPrefix(lowerDSN, "service=")) {
			return fmt.Errorf("DB_DSN must be a valid PostgreSQL URI or keyword DSN without placeholder values")
		}
	} else {
		for _, key := range []string{"DB_HOST", "DB_DATABASE", "DB_USERNAME"} {
			value := strings.TrimSpace(getenv(key))
			if value == "" || containsPlaceholder(value) {
				return fmt.Errorf("%s must be configured in production when DB_DSN is empty", key)
			}
		}
		if rawPort := strings.TrimSpace(getenv("DB_PORT")); rawPort != "" {
			port, err := strconv.Atoi(rawPort)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("DB_PORT must be between 1 and 65535")
			}
		}
	}
	if _, err := dbresilience.ParsePoolSettings(getenv); err != nil {
		return err
	}
	return nil
}

func parseTrustedProxyIPs(raw string) ([]string, error) {
	var result []string
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		ip, err := netip.ParseAddr(value)
		if err != nil || ip.IsUnspecified() || ip.IsMulticast() {
			return nil, fmt.Errorf("TRUSTED_PROXY_IPS must contain explicit unicast IP addresses, never CIDRs or wildcards")
		}
		result = append(result, ip.Unmap().String())
	}
	return result, nil
}

func isConfiguredSecret(value string) bool {
	return len(value) >= 32 && !containsPlaceholder(value)
}

func isExampleDomain(domain string) bool {
	domain = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
	return domain == "example.com" || domain == "example.net" || domain == "example.org" || domain == "example" ||
		strings.HasSuffix(domain, ".example.com") || strings.HasSuffix(domain, ".example.net") || strings.HasSuffix(domain, ".example.org") || strings.HasSuffix(domain, ".example")
}

func containsPlaceholder(value string) bool {
	lower := strings.ToLower(value)
	for _, placeholder := range []string{"replace-with", "change-me", "changeme", "your-secret", "example-secret"} {
		if strings.Contains(lower, placeholder) {
			return true
		}
	}
	return false
}

func normalizeEnvironment(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "":
		return "local", nil
	case "production", "local", "testing", "development", "staging":
		return value, nil
	default:
		return "", fmt.Errorf("unknown APP_ENV; use production, staging, local, development or testing")
	}
}

func productionEnvironment(raw string) bool {
	value := strings.ToLower(strings.TrimSpace(raw))
	return value == "production" || value == "staging"
}
