package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

func TestBanCacheBoundedAndNeverExtendsExpiry(t *testing.T) {
	var cache banCache
	now := time.Now()
	for n := 0; n < 10000; n++ {
		cache.put(fmt.Sprint(n), now.Add(time.Minute))
	}
	if len(cache.entries) > len(cache.order) {
		t.Fatalf("unbounded ban cache: %d", len(cache.entries))
	}
	cache.put("last", now.Add(time.Second))
	cache.put("last", now.Add(time.Hour))
	if got := cache.get("last", now); got != time.Second {
		t.Fatalf("ban was extended: %v", got)
	}
	if got := cache.get("last", now.Add(time.Second)); got != 0 {
		t.Fatalf("expired ban remains: %v", got)
	}
}

func TestCachedBanRejectsWithoutRedis(t *testing.T) {
	g := checkedGuard(t, nil, DefaultTrafficPolicy())
	g.cache.put(digestRateKey(canonicalIP("0.0.0.0")), time.Now().Add(time.Minute))
	app := fiber.New()
	app.Use(g.Handler)
	app.Get("/login", func(c fiber.Ctx) error { return c.SendStatus(200) })
	response, err := app.Test(httptest.NewRequest("GET", "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 429 {
		t.Fatalf("cached ban accessed unavailable Redis: %d", response.StatusCode)
	}
}

func trafficRedis(t *testing.T) *redis.Client {
	t.Helper()
	path, err := exec.LookPath("redis-server")
	if err != nil {
		t.Fatal("real Redis required")
	}
	dir, err := os.MkdirTemp("/tmp", "bt-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "r.sock")
	cmd := exec.Command(path, "--port", "0", "--unixsocket", socket, "--save", "", "--appendonly", "no")
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	client := redis.NewClient(&redis.Options{Network: "unix", Addr: socket, MaxRetries: -1, DialTimeout: time.Second, ReadTimeout: time.Second})
	t.Cleanup(func() { _ = client.Close() })
	deadline := time.Now().Add(3 * time.Second)
	for client.Ping(context.Background()).Err() != nil {
		if time.Now().After(deadline) {
			t.Fatal("Redis socket unavailable; run outside restricted sandbox")
		}
		time.Sleep(10 * time.Millisecond)
	}
	return client
}

func checkedGuard(t *testing.T, client redis.UniversalClient, p TrafficPolicy) *TrafficGuard {
	t.Helper()
	g, err := NewTrafficGuard(client, p)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestTrafficBucketSharedAtomicAndRefills(t *testing.T) {
	client := trafficRedis(t)
	ctx := context.Background()
	p := DefaultTrafficPolicy()
	p.Rate = 1
	guards := []*TrafficGuard{checkedGuard(t, client, p), checkedGuard(t, client, p)}
	var accepted atomic.Int64
	var wg sync.WaitGroup
	start := time.Now()
	for n := 0; n < 80; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			d, err := guards[n%2].Check(ctx, "192.0.2.1", false)
			if err != nil {
				t.Error(err)
				return
			}
			if d.Kind == 0 {
				accepted.Add(1)
			}
		}(n)
	}
	wg.Wait()
	if accepted.Load() < 40 || accepted.Load() > 40+int64(time.Since(start).Seconds()) {
		t.Fatalf("lost atomic bucket state: %d", accepted.Load())
	}
	key := trafficKeys("192.0.2.1", "dynamic")[1]
	now, err := client.Time(ctx).Result()
	if err != nil {
		t.Fatal(err)
	}
	if err := client.HSet(ctx, key, "at", now.Add(-2*time.Second).UnixMilli(), "tokens", 0).Err(); err != nil {
		t.Fatal(err)
	}
	d, err := guards[0].Check(ctx, "192.0.2.1", false)
	if err != nil || d.Kind != 0 {
		t.Fatalf("refill failed: %+v %v", d, err)
	}
	ttl := client.PTTL(ctx, key).Val()
	if ttl <= 0 || ttl > 42*time.Second {
		t.Fatalf("bucket not expiring: %v", ttl)
	}
	if strings.Contains(key, "192.0.2.1") {
		t.Fatal("IP stored in clear text")
	}
}

func TestTrafficBanEscalatesExpiresAndNeverExtends(t *testing.T) {
	client := trafficRedis(t)
	ctx := context.Background()
	p := DefaultTrafficPolicy()
	p.Rate = 1
	p.Burst = 1
	p.RejectLimit = 3
	p.ShortBan = 30 * time.Millisecond
	p.LongBan = 180 * time.Millisecond
	p.StrikeWindow = 2 * time.Second
	g := checkedGuard(t, client, p)
	ip := "192.0.2.2"
	check := func(static bool) TrafficDecision {
		t.Helper()
		d, e := g.Check(ctx, ip, static)
		if e != nil {
			t.Fatal(e)
		}
		return d
	}
	if check(false).Kind != 0 {
		t.Fatal("first request rejected")
	}
	for strike := 1; strike <= 3; strike++ {
		for n := 1; n <= 3; n++ {
			d := check(false)
			if n < 3 && d.Kind != 1 {
				t.Fatalf("early ban: %+v", d)
			}
			if n == 3 && d.Kind != 2 {
				t.Fatalf("missing ban %d: %+v", strike, d)
			}
		}
		state := trafficKeys(ip, "dynamic")[0]
		before := client.HGetAll(ctx, state).Val()
		for n := 0; n < 10; n++ {
			if check(true).Kind != 2 {
				t.Fatal("ban bypass through static lane")
			}
		}
		after := client.HGetAll(ctx, state).Val()
		if before["ban_until"] != after["ban_until"] || before["strikes"] != after["strikes"] {
			t.Fatal("banned requests extended or escalated ban")
		}
		if strike == 3 {
			if d := check(false); d.Retry < 100*time.Millisecond {
				t.Fatalf("third ban not long: %v", d.Retry)
			}
			time.Sleep(200 * time.Millisecond)
		} else {
			time.Sleep(40 * time.Millisecond)
		}
	}
	if d := check(false); d.Kind == 2 {
		t.Fatal("ban did not expire")
	}
	if d, e := g.Check(ctx, "192.0.2.3", false); e != nil || d.Kind != 0 {
		t.Fatal("ban leaked to different IP")
	}
	// An old strike window must not make the next strike a long ban.
	state := trafficKeys(ip, "dynamic")[0]
	now := client.Time(ctx).Val()
	client.HSet(ctx, state, "strike_until", now.Add(-time.Second).UnixMilli(), "rejects", 0)
	for n := 0; n < 3; n++ {
		check(false)
	}
	if got := client.HGet(ctx, state, "strikes").Val(); got != "1" {
		t.Fatalf("strike window did not reset: %s", got)
	}
}

func TestTrafficHTTPObserveFailureAndIPTrust(t *testing.T) {
	client := trafficRedis(t)
	p := DefaultTrafficPolicy()
	p.Rate = 1
	p.Burst = 1
	newApp := func(observe, trusted bool) *fiber.App {
		policy := p
		policy.Observe = observe
		g := checkedGuard(t, client, policy)
		app := fiber.New(fiber.Config{TrustProxy: trusted, ProxyHeader: "CF-Connecting-IP", TrustProxyConfig: fiber.TrustProxyConfig{Loopback: true}})
		app.Use(g.Handler)
		app.Get("/*", func(c fiber.Ctx) error { return c.SendStatus(204) })
		return app
	}
	request := func(app *fiber.App, path, header string) (int, string) {
		t.Helper()
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("CF-Connecting-IP", header)
		res, e := app.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		return res.StatusCode, res.Header.Get("Retry-After")
	}
	app := newApp(false, false)
	if s, _ := request(app, "/login", "192.0.2.4"); s != 204 {
		t.Fatal(s)
	}
	if s, retry := request(app, "/login", "192.0.2.5"); s != 429 || retry == "" {
		t.Fatalf("untrusted header evaded quota: %d %s", s, retry)
	}
	observe := newApp(true, false)
	if s, _ := request(observe, "/login", "192.0.2.6"); s != 204 {
		t.Fatalf("observe blocked: %d", s)
	}
	// Normalize IPv4-mapped IPv6 to avoid alternate identities.
	if trafficKeys("192.0.2.4", "dynamic")[0] != trafficKeys("::ffff:192.0.2.4", "dynamic")[0] {
		t.Fatal("IPv4-mapped quota bypass")
	}
	_ = client.Close()
	if s, _ := request(app, "/login", ""); s != 503 {
		t.Fatalf("Redis failure not closed: %d", s)
	}
	if s, _ := request(app, "/healthz", ""); s != 204 {
		t.Fatalf("liveness coupled to Redis: %d", s)
	}
	if s, _ := request(observe, "/login", ""); s != 204 {
		t.Fatalf("observe not available during Redis outage: %d", s)
	}
}

func TestConcurrencyGateRejectsWithoutQueueAndReleases(t *testing.T) {
	app := fiber.New()
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var wait sync.WaitGroup
	app.Use(NewConcurrencyLimit(2))
	app.Get("/work", func(c fiber.Ctx) error { entered <- struct{}{}; <-release; return c.SendStatus(204) })
	for n := 0; n < 2; n++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			res, err := app.Test(httptest.NewRequest("GET", "/work", nil), fiber.TestConfig{Timeout: 5 * time.Second})
			if err != nil {
				t.Error(err)
				return
			}
			res.Body.Close()
			if res.StatusCode != 204 {
				t.Error(res.StatusCode)
			}
		}()
	}
	for n := 0; n < 2; n++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("work not admitted")
		}
	}
	res, err := app.Test(httptest.NewRequest("GET", "/work", nil))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 503 || res.Header.Get("Retry-After") == "" {
		t.Fatalf("capacity did not reject: %d", res.StatusCode)
	}
	close(release)
	wait.Wait()
	res, err = app.Test(httptest.NewRequest("GET", "/work", nil))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 204 {
		t.Fatal("slot leaked")
	}
}

func TestTrafficIdentityCardinalityBounded(t *testing.T) {
	client := trafficRedis(t)
	p := DefaultTrafficPolicy()
	p.MaxIdentities = 2
	guard := checkedGuard(t, client, p)
	ctx := context.Background()
	for _, ip := range []string{"198.18.0.1", "198.18.0.2"} {
		d, err := guard.Check(ctx, ip, false)
		if err != nil || d.Kind != 0 {
			t.Fatal(d, err)
		}
	}
	d, err := guard.Check(ctx, "198.18.0.3", false)
	if err != nil || d.Kind != 3 {
		t.Fatal("new identity bypassed cap", d, err)
	}
	d, err = guard.Check(ctx, "198.18.0.1", false)
	if err != nil || d.Kind != 0 {
		t.Fatal("existing identity denied", d, err)
	}
	if got := client.ZCard(ctx, "bufalo:traffic:v1:identities").Val(); got != 2 {
		t.Fatal("registry unbounded", got)
	}
}
