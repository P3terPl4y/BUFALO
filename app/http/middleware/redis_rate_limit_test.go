package middleware

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisRateCounterIsAtomicAcrossConcurrentRequests(t *testing.T) {
	redisServer, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server not installed; Redis counter integration test skipped")
	}
	dir, err := os.MkdirTemp("", "br-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := dir + "/redis.sock"
	cmd := exec.Command(redisServer, "--port", "0", "--unixsocket", socket, "--unixsocketperm", "700", "--save", "", "--appendonly", "no")
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start isolated redis-server: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })

	client := redis.NewClient(&redis.Options{Network: "unix", Addr: socket, DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second, MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := client.Ping(context.Background()).Err(); err == nil {
			break
		} else if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("sandbox blocks local Redis sockets; rerun outside sandbox: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated Redis did not become ready")
		}
		time.Sleep(20 * time.Millisecond)
	}

	counter := NewRedisRateCounter(client)
	const requests = 256
	counts := make(chan int64, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, _, err := counter.Increment(context.Background(), "test:atomic", time.Minute)
			if err != nil {
				t.Errorf("atomic Redis increment: %v", err)
				return
			}
			counts <- count
		}()
	}
	wg.Wait()
	close(counts)
	got := make([]int, 0, requests)
	for count := range counts {
		got = append(got, int(count))
	}
	sort.Ints(got)
	if len(got) != requests {
		t.Fatalf("got %d counter results, want %d", len(got), requests)
	}
	for i, count := range got {
		if count != i+1 {
			t.Fatalf("counter values are not atomic/unique at position %d: got %d", i, count)
		}
	}

	first, ttl, err := counter.Increment(context.Background(), "test:expiry", 150*time.Millisecond)
	if err != nil || first != 1 || ttl <= 0 {
		t.Fatalf("first increment: count=%d ttl=%s err=%v", first, ttl, err)
	}
	time.Sleep(200 * time.Millisecond)
	second, _, err := counter.Increment(context.Background(), "test:expiry", time.Minute)
	if err != nil || second != 1 {
		t.Fatalf("expired counter should reset: count=%d err=%v", second, err)
	}
	if _, err := client.Get(context.Background(), "test:atomic").Result(); err != nil {
		t.Fatal(fmt.Errorf("counter key missing after increments: %w", err))
	}
}
