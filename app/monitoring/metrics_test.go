package monitoring

import (
	"net"
	"testing"
	"time"
)

func TestCountersAggregateLatencyAndHTTPStatus(t *testing.T) {
	var got counters
	for i := 0; i < 18; i++ {
		got.record(40*time.Millisecond, 200)
	}
	got.record(75*time.Millisecond, 404)
	got.record(800*time.Millisecond, 500)

	if got.requests.Load() != 20 {
		t.Fatalf("requests = %d, want 20", got.requests.Load())
	}
	if got.client.Load() != 1 || got.server.Load() != 1 {
		t.Fatalf("error counters = 4xx:%d 5xx:%d, want 1 each", got.client.Load(), got.server.Load())
	}
	if got.p95Bucket(got.requests.Load()) != "≤ 100ms" {
		t.Fatalf("p95 bucket = %q, want ≤ 100ms", got.p95Bucket(got.requests.Load()))
	}
}

func TestCheckTCPReportsDependencyReachability(t *testing.T) {
	var address string
	var timeout time.Duration
	left, right := net.Pipe()
	defer right.Close()
	available, latency := checkTCP("127.0.0.1", 5432, func(network, gotAddress string, gotTimeout time.Duration) (net.Conn, error) {
		address, timeout = gotAddress, gotTimeout
		if network != "tcp" {
			t.Fatalf("network = %q, want tcp", network)
		}
		return left, nil
	})
	if !available || latency < 0 {
		t.Fatalf("CheckTCP() = (%v, %v), want reachable and non-negative latency", available, latency)
	}
	if address != "127.0.0.1:5432" || timeout != time.Second {
		t.Fatalf("dial target = %q timeout = %s, want 127.0.0.1:5432 and 1s", address, timeout)
	}
	invalid, invalidLatency := CheckTCP("", 5432)
	if invalid || invalidLatency != 0 {
		t.Fatalf("invalid check = (%v, %v), want (false, 0)", invalid, invalidLatency)
	}
}
