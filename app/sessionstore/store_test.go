package sessionstore

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func redisFixture(t *testing.T) *redis.Client {
	t.Helper()
	path, err := exec.LookPath("redis-server")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("/tmp", "bs-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "r.sock")
	cmd := exec.Command(path, "--port", "0", "--unixsocket", socket, "--save", "", "--appendonly", "no")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	client := redis.NewClient(&redis.Options{Network: "unix", Addr: socket, MaxRetries: -1})
	t.Cleanup(func() { client.Close() })
	deadline := time.Now().Add(3 * time.Second)
	for client.Ping(context.Background()).Err() != nil {
		if time.Now().After(deadline) {
			t.Fatal("Redis fixture unavailable")
		}
		time.Sleep(10 * time.Millisecond)
	}
	return client
}
func TestSessionCapacityAtomicAndExistingSessionPreserved(t *testing.T) {
	client := redisFixture(t)
	store := New(client, 8)
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := store.Set(fmt.Sprint(i), []byte("session"), time.Minute)
			if err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, ErrCapacity) {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if accepted.Load() != 8 {
		t.Fatal(accepted.Load())
	}
	keys, err := client.ZRange(context.Background(), registry, 0, -1).Result()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set(keys[0], []byte("updated"), time.Minute); err != nil {
		t.Fatal("existing session rejected", err)
	}
	value, err := store.Get(keys[0])
	if err != nil || string(value) != "updated" {
		t.Fatal("existing session corrupted")
	}
	if err := store.Delete(keys[0]); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("replacement", []byte("ok"), time.Minute); err != nil {
		t.Fatal("deleted session did not free capacity", err)
	}
}
func TestExpiredSessionFreesCapacity(t *testing.T) {
	store := New(redisFixture(t), 1)
	if err := store.Set("short", []byte("a"), 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	if err := store.Set("new", []byte("b"), time.Minute); err != nil {
		t.Fatal(err)
	}
}
func TestWriteProbeDetectsOOMWhilePingSucceeds(t *testing.T) {
	client := redisFixture(t)
	ctx := context.Background()
	if err := client.ConfigSet(ctx, "maxmemory", "1").Err(); err != nil {
		t.Fatal(err)
	}
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatal("fixture should still answer PING", err)
	}
	if err := WriteCheck(client, "probe")(ctx); err == nil {
		t.Fatal("write probe missed OOM")
	}
}
