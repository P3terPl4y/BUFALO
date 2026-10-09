package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

// RateCounter provides an atomic increment with expiry. Redis is required in
// production so counters are shared by all application instances.
type RateCounter interface {
	Increment(context.Context, string, time.Duration) (int64, time.Duration, error)
}

type RedisRateCounter struct{ client redis.UniversalClient }

func NewRedisRateCounter(client redis.UniversalClient) *RedisRateCounter {
	return &RedisRateCounter{client: client}
}

const incrementWithExpiryScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
return {count, redis.call('PTTL', KEYS[1])}
`

func (r *RedisRateCounter) Increment(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	if r == nil || r.client == nil {
		return 0, 0, fmt.Errorf("rate limit counter is unavailable")
	}
	values, err := r.client.Eval(ctx, incrementWithExpiryScript, []string{key}, window.Milliseconds()).Slice()
	if err != nil {
		return 0, 0, err
	}
	if len(values) != 2 {
		return 0, 0, fmt.Errorf("unexpected Redis rate counter response")
	}
	count, okCount := redisInt64(values[0])
	ttlMillis, okTTL := redisInt64(values[1])
	if !okCount || !okTTL {
		return 0, 0, fmt.Errorf("invalid Redis rate counter response")
	}
	return count, time.Duration(ttlMillis) * time.Millisecond, nil
}

func redisInt64(value any) (int64, bool) {
	switch n := value.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case string:
		var parsed int64
		_, err := fmt.Sscan(n, &parsed)
		return parsed, err == nil
	default:
		return 0, false
	}
}

// MemoryRateCounter is only for local/test use. It is process-local and must
// not be used by production application instances.
type MemoryRateCounter struct {
	mu      sync.Mutex
	entries map[string]memoryRateEntry
}

type memoryRateEntry struct {
	count int64
	until time.Time
}

func NewMemoryRateCounter() *MemoryRateCounter {
	return &MemoryRateCounter{entries: make(map[string]memoryRateEntry)}
}

func (m *MemoryRateCounter) Increment(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	entry := m.entries[key]
	if !entry.until.After(now) {
		entry = memoryRateEntry{until: now.Add(window)}
	}
	entry.count++
	m.entries[key] = entry
	return entry.count, time.Until(entry.until), nil
}

type rateKeyFunc func(fiber.Ctx) string

func NewRateLimit(counter RateCounter, namespace string, max int64, window time.Duration, keyFn rateKeyFunc) fiber.Handler {
	return func(c fiber.Ctx) error {
		key := "bufalo:rate:v1:" + namespace + ":" + digestRateKey(keyFn(c))
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		count, ttl, err := counter.Increment(ctx, key, window)
		cancel()
		if err != nil {
			// Fail closed: accepting unmetered requests while Redis is unavailable
			// would silently remove the brute-force/abuse control.
			return c.Status(fiber.StatusServiceUnavailable).SendString("Servicio temporalmente no disponible. Inténtalo más tarde.")
		}
		if count > max {
			seconds := int64(math.Ceil(ttl.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			c.Set(fiber.HeaderRetryAfter, fmt.Sprintf("%d", seconds))
			return c.Status(fiber.StatusTooManyRequests).SendString("Demasiadas solicitudes. Inténtalo más tarde.")
		}
		return c.Next()
	}
}

func digestRateKey(value string) string {
	secret := os.Getenv("RATE_LIMIT_KEY_SECRET")
	if secret == "" {
		secret = os.Getenv("APP_KEY")
	}
	if secret == "" {
		secret = "bufalo-local-rate-limit-key"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strings.ToLower(strings.TrimSpace(value))))
	return hex.EncodeToString(mac.Sum(nil))
}

func clientIPKey(c fiber.Ctx) string { return c.IP() }

func emailKey(c fiber.Ctx) string {
	return strings.ToLower(strings.TrimSpace(string(c.Request().PostArgs().Peek("email"))))
}

// LoginEmailRateLimiter acota intentos distribuidos entre varias IP por cuenta.
func LoginEmailRateLimiter(counter RateCounter) fiber.Handler {
	return NewRateLimit(counter, "login:email", 20, 15*time.Minute, emailKey)
}

// ConfirmationIPRateLimiter limita consultas públicas que toman bloqueos SQL.
func ConfirmationIPRateLimiter(counter RateCounter) fiber.Handler {
	return NewRateLimit(counter, "confirm:ip", 30, time.Minute, clientIPKey)
}

func LoginIPRateLimiter(counter RateCounter) fiber.Handler {
	return NewRateLimit(counter, "login:ip", 10, time.Minute, clientIPKey)
}

func RegisterIPRateLimiter(counter RateCounter) fiber.Handler {
	return NewRateLimit(counter, "register:ip", 5, 15*time.Minute, clientIPKey)
}

func RegisterEmailRateLimiter(counter RateCounter) fiber.Handler {
	return NewRateLimit(counter, "register:email", 2, time.Hour, emailKey)
}

func InterestAccountRateLimiter(counter RateCounter) fiber.Handler {
	return NewRateLimit(counter, "interest:account", 10, time.Hour, func(c fiber.Ctx) string { return fmt.Sprint(c.Locals("user_id")) })
}

func CompanyChatReadRateLimiter(counter RateCounter) fiber.Handler {
	return NewRateLimit(counter, "company-chat:read", 60, time.Minute, func(c fiber.Ctx) string { return fmt.Sprint(c.Locals("user_id")) })
}

func CompanyChatWriteRateLimiter(counter RateCounter) fiber.Handler {
	return NewRateLimit(counter, "company-chat:write", 20, time.Minute, func(c fiber.Ctx) string { return fmt.Sprint(c.Locals("user_id")) })
}

func ProfileAccountRateLimiter(counter RateCounter) fiber.Handler {
	return NewRateLimit(counter, "profile:account", 10, time.Minute, func(c fiber.Ctx) string { return fmt.Sprint(c.Locals("user_id")) })
}

// LoginRateLimiter preserves the existing single-handler API for legacy tests
// and local callers. Production routes use the explicit layered limiters.
func LoginRateLimiter() fiber.Handler {
	return LoginIPRateLimiter(NewMemoryRateCounter())
}
