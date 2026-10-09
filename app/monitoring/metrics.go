package monitoring

import (
	"errors"
	"math"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
)

var processStartedAt = time.Now()

var latencyBuckets = [...]time.Duration{
	50 * time.Millisecond,
	100 * time.Millisecond,
	250 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
}

type counters struct {
	requests atomic.Uint64
	client   atomic.Uint64
	server   atomic.Uint64
	totalNS  atomic.Uint64
	buckets  [len(latencyBuckets) + 1]atomic.Uint64
}

var processCounters counters

// CheckTCP reports whether a dependency accepts a TCP connection. It does not
// authenticate or run a query, so callers should label it as reachability.
func CheckTCP(host string, port int) (bool, float64) {
	return checkTCP(host, port, net.DialTimeout)
}

func checkTCP(host string, port int, dial func(string, string, time.Duration) (net.Conn, error)) (bool, float64) {
	if host == "" || port <= 0 {
		return false, 0
	}
	started := time.Now()
	conn, err := dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), time.Second)
	latency := float64(time.Since(started)) / float64(time.Millisecond)
	if err != nil {
		return false, latency
	}
	_ = conn.Close()
	return true, latency
}

// Middleware mide resultados finales y conserva un historial acotado sin datos
// de entrada. Debe envolver el manejador de errores y Recover.
func Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Path() == "/healthz" || c.Path() == "/readyz" || c.Path() == "/presence/heartbeat" || c.Path() == "/admin/health" || strings.HasPrefix(c.Path(), "/admin/health/") {
			return c.Next()
		}
		started := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			status = fiber.StatusInternalServerError
			var httpError *fiber.Error
			if errors.As(err, &httpError) {
				status = httpError.Code
			}
		}
		for _, prefix := range []string{"/css/", "/js/", "/img/", "/uploads/", "/fonts/"} {
			if status < 400 && strings.HasPrefix(c.Path(), prefix) {
				return err
			}
		}
		duration := time.Since(started)
		processCounters.record(duration, status)
		route := c.Route().Path
		if route == "" || route == "/" && c.Path() != "/" || strings.Contains(route, "*") {
			route = "(ruta no registrada)"
		}
		history.record(RequestEvent{At: time.Now().UTC(), Method: strings.Clone(c.Method()), Route: strings.Clone(route), Status: status, DurationMS: float64(duration) / float64(time.Millisecond)})
		return err
	}
}

func (m *counters) record(duration time.Duration, status int) {
	if duration < 0 {
		duration = 0
	}
	m.requests.Add(1)
	m.totalNS.Add(uint64(duration))
	if status >= 500 {
		m.server.Add(1)
	} else if status >= 400 {
		m.client.Add(1)
	}
	for i, upper := range latencyBuckets {
		if duration <= upper {
			m.buckets[i].Add(1)
			return
		}
	}
	m.buckets[len(latencyBuckets)].Add(1)
}

type RuntimeSnapshot struct {
	ResidentMemoryMB  *float64 `json:"resident_memory_mb"`
	UptimeSeconds     int64    `json:"uptime_seconds"`
	Requests          uint64   `json:"requests"`
	RequestsPerMinute float64  `json:"requests_per_minute"`
	ClientErrors      uint64   `json:"client_errors_4xx"`
	ServerErrors      uint64   `json:"server_errors_5xx"`
	ErrorRatePercent  float64  `json:"error_rate_percent"`
	AverageLatencyMS  float64  `json:"average_latency_ms"`
	P95Latency        string   `json:"p95_latency"`
	HeapInUseMB       float64  `json:"heap_in_use_mb"`
	HeapAllocMB       float64  `json:"heap_alloc_mb"`
	TotalAllocMB      float64  `json:"total_alloc_mb"`
	Goroutines        int      `json:"goroutines"`
	GOMAXPROCS        int      `json:"gomaxprocs"`
	GCCycles          uint32   `json:"gc_cycles"`
}

func Snapshot() RuntimeSnapshot {
	up := time.Since(processStartedAt)
	if up <= 0 {
		up = time.Second
	}
	requests := processCounters.requests.Load()
	var average, errorRate float64
	if requests > 0 {
		average = float64(processCounters.totalNS.Load()) / float64(requests) / float64(time.Millisecond)
		errorRate = float64(processCounters.server.Load()) * 100 / float64(requests)
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return RuntimeSnapshot{
		ResidentMemoryMB:  residentMemory(),
		UptimeSeconds:     int64(up.Seconds()),
		Requests:          requests,
		RequestsPerMinute: float64(requests) / up.Minutes(),
		ClientErrors:      processCounters.client.Load(),
		ServerErrors:      processCounters.server.Load(),
		ErrorRatePercent:  errorRate,
		AverageLatencyMS:  average,
		P95Latency:        processCounters.p95Bucket(requests),
		HeapInUseMB:       bytesToMB(mem.HeapInuse),
		HeapAllocMB:       bytesToMB(mem.HeapAlloc),
		TotalAllocMB:      bytesToMB(mem.TotalAlloc),
		Goroutines:        runtime.NumGoroutine(),
		GOMAXPROCS:        runtime.GOMAXPROCS(0),
		GCCycles:          mem.NumGC,
	}
}

func residentMemory() *float64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return nil
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return nil
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return nil
	}
	value := bytesToMB(pages * uint64(os.Getpagesize()))
	return &value
}

func (m *counters) p95Bucket(requests uint64) string {
	if requests == 0 {
		return "—"
	}
	threshold := uint64(math.Ceil(float64(requests) * 0.95))
	var cumulative uint64
	for i, bucket := range latencyBuckets {
		cumulative += m.buckets[i].Load()
		if cumulative >= threshold {
			return "≤ " + bucket.String()
		}
	}
	return "> 1s"
}

func bytesToMB(value uint64) float64 {
	return math.Round(float64(value)/(1024*1024)*100) / 100
}
