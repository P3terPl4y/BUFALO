package middleware

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"goravel/app/monitoring"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

type TrafficPolicy struct {
	MaxIdentities                                 int64
	Rate, Burst, StaticRate, StaticBurst          int64
	RejectLimit, StrikeLimit                      int64
	RejectWindow, StrikeWindow, ShortBan, LongBan time.Duration
	Observe                                       bool
}

func DefaultTrafficPolicy() TrafficPolicy {
	return TrafficPolicy{MaxIdentities: 20000, Rate: 20, Burst: 40, StaticRate: 100, StaticBurst: 200,
		RejectLimit: 100, StrikeLimit: 3, RejectWindow: time.Minute,
		StrikeWindow: 30 * time.Minute, ShortBan: time.Minute, LongBan: 20 * time.Minute}
}

func (p TrafficPolicy) Validate() error {
	if p.MaxIdentities <= 0 || p.Rate <= 0 || p.Burst <= 0 || p.StaticRate <= 0 || p.StaticBurst <= 0 ||
		p.RejectLimit <= 0 || p.StrikeLimit <= 0 || p.RejectWindow < time.Millisecond ||
		p.StrikeWindow < time.Millisecond || p.ShortBan < time.Millisecond || p.LongBan < time.Millisecond {
		return fmt.Errorf("invalid traffic protection policy")
	}
	return nil
}

// Redis TIME avoids clock differences between application instances. Ban
// checks perform no writes, so traffic during a ban never extends its expiry.
// Per-IP keys are HMACed. The global registry requires standalone Redis,
// matching the configured NewClient transport (not Redis Cluster).
var trafficScript = redis.NewScript(`
local clock = redis.call('TIME')
local now = clock[1]*1000 + math.floor(clock[2]/1000)
local ban = tonumber(redis.call('HGET',KEYS[1],'ban_until') or '0')
if ban > now then return {2,ban-now} end
local registry=KEYS[3]
local expired=redis.call('ZRANGEBYSCORE',registry,'-inf',now,'LIMIT',0,128)
if #expired>0 then redis.call('ZREM',registry,unpack(expired)) end
if not redis.call('ZSCORE',registry,KEYS[1]) then
 if redis.call('ZCARD',registry)>=tonumber(ARGV[9]) then return {3,1000} end
 redis.call('ZADD',registry,now+tonumber(ARGV[10]),KEYS[1])
elseif tonumber(redis.call('ZSCORE',registry,KEYS[1]))<now+tonumber(ARGV[10])/2 then
 redis.call('ZADD',registry,now+tonumber(ARGV[10]),KEYS[1])
end
redis.call('PEXPIRE',registry,tonumber(ARGV[10])+1000)

local rate,burst = tonumber(ARGV[1]),tonumber(ARGV[2])
local bucket = redis.call('HMGET',KEYS[2],'tokens','at')
local tokens = tonumber(bucket[1]) or burst
local at = tonumber(bucket[2]) or now
tokens = math.min(burst,tokens+math.max(0,now-at)*rate/1000)
local allowed = tokens >= 1
if allowed then tokens = tokens-1 end
redis.call('HSET',KEYS[2],'tokens',tokens,'at',now)
redis.call('PEXPIRE',KEYS[2],math.ceil(burst/rate*1000)+1000)
if allowed then return {0,0} end
local rejectLimit,rejectWindow = tonumber(ARGV[3]),tonumber(ARGV[4])
local strikeLimit,strikeWindow = tonumber(ARGV[5]),tonumber(ARGV[6])
local shortBan,longBan = tonumber(ARGV[7]),tonumber(ARGV[8])
local state = redis.call('HMGET',KEYS[1],'rejects','reject_until','strikes','strike_until')
local rejects,rejectUntil = tonumber(state[1]) or 0,tonumber(state[2]) or 0
local strikes,strikeUntil = tonumber(state[3]) or 0,tonumber(state[4]) or 0
if rejectUntil <= now then rejects=0; rejectUntil=now+rejectWindow end
if strikeUntil <= now then strikes=0; strikeUntil=0 end
rejects=rejects+1
if rejects >= rejectLimit then
  if strikes == 0 then strikeUntil=now+strikeWindow end
  strikes=strikes+1
  local duration=shortBan
  if strikes >= strikeLimit then duration=longBan end
  ban=now+duration
  rejects=0; rejectUntil=now+rejectWindow
  redis.call('HSET',KEYS[1],'ban_until',ban)
end
redis.call('HSET',KEYS[1],'rejects',rejects,'reject_until',rejectUntil,'strikes',strikes,'strike_until',strikeUntil)
redis.call('PEXPIRE',KEYS[1],math.max(rejectUntil,strikeUntil,ban)-now)
if ban > now then return {2,ban-now} end
return {1,math.max(1,math.ceil((1-tokens)/rate*1000))}
`)

type TrafficDecision struct {
	Kind  int64
	Retry time.Duration
}
type TrafficGuard struct {
	client redis.UniversalClient
	policy TrafficPolicy
	cache  banCache
}

// Cache only positive bans, never an allow decision. A fixed-size ring bounds
// memory under many identities; eviction falls back to authoritative Redis.
type banCache struct {
	mu      sync.Mutex
	entries map[string]time.Time
	order   [4096]string
	next    int
}

func (b *banCache) get(key string, now time.Time) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	until, ok := b.entries[key]
	if !ok {
		return 0
	}
	if !until.After(now) {
		delete(b.entries, key)
		return 0
	}
	return until.Sub(now)
}

func (b *banCache) put(key string, until time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.entries == nil {
		b.entries = make(map[string]time.Time)
	}
	if old, ok := b.entries[key]; ok {
		if until.Before(old) {
			b.entries[key] = until
		}
		return
	}
	delete(b.entries, b.order[b.next])
	b.order[b.next] = key
	b.next = (b.next + 1) % len(b.order)
	b.entries[key] = until
}

func NewTrafficGuard(client redis.UniversalClient, policy TrafficPolicy) (*TrafficGuard, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &TrafficGuard{client: client, policy: policy}, nil
}

func canonicalIP(raw string) string {
	if ip, err := netip.ParseAddr(raw); err == nil {
		return ip.Unmap().String()
	}
	return "unknown" // never create arbitrary cardinality from invalid headers
}

func trafficKeys(ip, lane string) []string {
	return trafficDigestKeys(digestRateKey(canonicalIP(ip)), lane)
}

func trafficDigestKeys(digest, lane string) []string {
	base := "bufalo:traffic:v1:{" + digest + "}:"
	return []string{base + "abuse", base + lane}
}

func (g *TrafficGuard) Check(ctx context.Context, ip string, static bool) (TrafficDecision, error) {
	return g.checkDigest(ctx, digestRateKey(canonicalIP(ip)), static)
}

func (g *TrafficGuard) checkDigest(ctx context.Context, digest string, static bool) (TrafficDecision, error) {
	if g.client == nil {
		return TrafficDecision{}, fmt.Errorf("traffic store unavailable")
	}
	p := g.policy
	rate, burst, lane := p.Rate, p.Burst, "dynamic"
	if static {
		rate, burst, lane = p.StaticRate, p.StaticBurst, "static"
	}
	keys := append(trafficDigestKeys(digest, lane), "bufalo:traffic:v1:identities")
	if p.Observe {
		for i := range keys {
			keys[i] = strings.Replace(keys[i], "traffic:v1:", "traffic:observe:v1:", 1)
		}
	}
	values, err := trafficScript.Run(ctx, g.client, keys, rate, burst, p.RejectLimit, p.RejectWindow.Milliseconds(), p.StrikeLimit, p.StrikeWindow.Milliseconds(), p.ShortBan.Milliseconds(), p.LongBan.Milliseconds(), p.MaxIdentities, max(p.StrikeWindow, p.LongBan, p.RejectWindow).Milliseconds()+2000).Slice()
	if err != nil {
		return TrafficDecision{}, err
	}
	if len(values) != 2 {
		return TrafficDecision{}, fmt.Errorf("invalid traffic store response")
	}
	kind, ok := redisInt64(values[0])
	retry, okRetry := redisInt64(values[1])
	if !ok || !okRetry || kind < 0 || kind > 3 || retry < 0 {
		return TrafficDecision{}, fmt.Errorf("invalid traffic decision")
	}
	return TrafficDecision{Kind: kind, Retry: time.Duration(retry) * time.Millisecond}, nil
}

func staticTraffic(c fiber.Ctx) bool {
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		return false
	}
	for _, prefix := range []string{"/css/", "/js/", "/img/", "/uploads/", "/fonts/"} {
		if strings.HasPrefix(c.Path(), prefix) {
			return true
		}
	}
	return c.Path() == "/favicon.ico" || c.Path() == "/robots.txt"
}

func (g *TrafficGuard) Handler(c fiber.Ctx) error {
	if c.Path() == "/healthz" && (c.Method() == "GET" || c.Method() == "HEAD") {
		return c.Next()
	}
	decision, err := g.Decide(c.Context(), c.IP(), staticTraffic(c))
	if err != nil {
		monitoring.RecordTraffic("store_error")
		if g.policy.Observe {
			return c.Next()
		}
		c.Set(fiber.HeaderRetryAfter, "1")
		return c.Status(503).SendString("Servicio temporalmente no disponible.")
	}
	if decision.Kind == 0 {
		return c.Next()
	}
	if g.policy.Observe {
		monitoring.RecordTraffic("observed")
		return c.Next()
	}
	if decision.Kind == 3 {
		c.Set(fiber.HeaderRetryAfter, "1")
		monitoring.RecordTraffic("capacity")
		return c.SendStatus(503)
	}
	if decision.Kind == 2 {
		monitoring.RecordTraffic("banned")
	} else {
		monitoring.RecordTraffic("limited")
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set(fiber.HeaderRetryAfter, fmt.Sprint(int64(math.Max(1, math.Ceil(decision.Retry.Seconds())))))
	return c.Status(429).SendString("Demasiadas solicitudes. Inténtalo más tarde.")
}

// A non-blocking gate prevents an unbounded queue of expensive work. Each
// process owns its capacity; rate and ban state remain shared in Redis.
func NewConcurrencyLimit(max int) fiber.Handler {
	if max < 1 {
		panic("concurrency limit must be positive")
	}
	slots := make(chan struct{}, max)
	return func(c fiber.Ctx) error {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
			return c.Next()
		default:
			monitoring.RecordTraffic("capacity")
			c.Set(fiber.HeaderRetryAfter, "1")
			return c.Status(503).SendString("Servicio ocupado. Inténtalo más tarde.")
		}
	}
}

// Decide shares the positive-ban cache across HTTP transports.
func (g *TrafficGuard) Decide(parent context.Context, ip string, static bool) (TrafficDecision, error) {
	digest := digestRateKey(canonicalIP(ip))
	started := time.Now()
	if retry := g.cache.get(digest, started); retry > 0 {
		return TrafficDecision{Kind: 2, Retry: retry}, nil
	}
	ctx, cancel := context.WithTimeout(parent, 500*time.Millisecond)
	defer cancel()
	decision, err := g.checkDigest(ctx, digest, static)
	if err == nil && decision.Kind == 2 {
		g.cache.put(digest, started.Add(decision.Retry))
	}
	return decision, err
}
func (g *TrafficGuard) AllowHTTP(w http.ResponseWriter, r *http.Request, ip string, static bool) bool {
	decision, err := g.Decide(r.Context(), ip, static)
	if err != nil {
		monitoring.RecordTraffic("store_error")
		if g.policy.Observe {
			return true
		}
		w.Header().Set("Connection", "close")
		w.Header().Set("Retry-After", "1")
		w.Header().Set("Cache-Control", "no-store")
		monitoring.RecordAttack("store_error", 503, ip)
		http.Error(w, "Servicio temporalmente no disponible.", 503)
		return false
	}
	if decision.Kind == 0 {
		return true
	}
	if g.policy.Observe {
		monitoring.RecordTraffic("observed")
		return true
	}
	if decision.Kind == 2 {
		monitoring.RecordTraffic("banned")
	} else if decision.Kind == 3 {
		monitoring.RecordTraffic("capacity")
	} else {
		monitoring.RecordTraffic("limited")
	}
	w.Header().Set("Connection", "close")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", fmt.Sprint(int64(math.Max(1, math.Ceil(decision.Retry.Seconds())))))
	status := 429
	if decision.Kind == 3 {
		status = 503
	}
	outcome := "limited"
	if decision.Kind == 2 {
		outcome = "banned"
	}
	if decision.Kind == 3 {
		outcome = "identity_capacity"
	}
	monitoring.RecordAttack(outcome, status, ip)
	http.Error(w, "Servicio temporalmente limitado.", status)
	return false
}

// RejectCachedHTTP rejects known bans without Redis work or consuming the
// global admission budget. A cache miss still uses the bounded normal path.
func (g *TrafficGuard) RejectCachedHTTP(w http.ResponseWriter, r *http.Request, ip string) bool {
	if g.policy.Observe {
		return false
	}
	retry := g.cache.get(digestRateKey(canonicalIP(ip)), time.Now())
	if retry <= 0 {
		return false
	}
	w.Header().Set("Connection", "close")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", fmt.Sprint(int64(math.Max(1, math.Ceil(retry.Seconds())))))
	monitoring.RecordTraffic("banned")
	monitoring.RecordAttack("banned", 429, ip)
	http.Error(w, "Servicio temporalmente limitado.", 429)
	return true
}
