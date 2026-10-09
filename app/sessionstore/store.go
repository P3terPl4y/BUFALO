// Package sessionstore preserves existing sessions while bounding new state.
package sessionstore

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"goravel/app/monitoring"
	"time"
)

const registry = "bufalo:sessions:registry:v1"

var ErrCapacity = errors.New("session capacity reached")
var ErrUnavailable = errors.New("session store unavailable")

type Store struct {
	Client redis.UniversalClient
	Max    int64
}

func New(client redis.UniversalClient, max int64) *Store {
	if max < 1 {
		panic("invalid session capacity")
	}
	return &Store{client, max}
}

var setScript = redis.NewScript(`
local clock=redis.call('TIME')
local now=clock[1]*1000+math.floor(clock[2]/1000)
local expired=redis.call('ZRANGEBYSCORE',KEYS[2],'-inf',now,'LIMIT',0,256)
if #expired>0 then redis.call('ZREM',KEYS[2],unpack(expired)) end
local existing=redis.call('EXISTS',KEYS[1])
local indexed=redis.call('ZSCORE',KEYS[2],KEYS[1])
if existing==0 and not indexed and redis.call('ZCARD',KEYS[2])>=tonumber(ARGV[3]) then return 0 end
redis.call('SET',KEYS[1],ARGV[1],'PX',ARGV[2])
if indexed or redis.call('ZCARD',KEYS[2])<tonumber(ARGV[3]) then redis.call('ZADD',KEYS[2],now+tonumber(ARGV[2]),KEYS[1]) end
redis.call('PEXPIRE',KEYS[2],86401000)
return 1
`)

func (s *Store) Get(key string) ([]byte, error) { return s.GetWithContext(context.Background(), key) }
func (s *Store) GetWithContext(parent context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, nil
	}
	if len(key) > 256 {
		return nil, errors.New("invalid session identity")
	}
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	value, err := s.Client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return value, nil
}
func (s *Store) Set(key string, value []byte, expiration time.Duration) error {
	return s.SetWithContext(context.Background(), key, value, expiration)
}
func (s *Store) SetWithContext(parent context.Context, key string, value []byte, expiration time.Duration) error {
	if key == "" || len(key) > 256 || len(value) > 4<<10 {
		return errors.New("invalid session state")
	}
	if expiration <= 0 || expiration > 24*time.Hour {
		expiration = 24 * time.Hour
	}
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	admitted, err := setScript.Run(ctx, s.Client, []string{key, registry}, value, max(1, expiration.Milliseconds()), s.Max).Int()
	if err != nil {
		monitoring.RecordTraffic("store_error")
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if admitted != 1 {
		monitoring.RecordTraffic("session_capacity")
		return ErrCapacity
	}
	return nil
}
func (s *Store) Delete(key string) error { return s.DeleteWithContext(context.Background(), key) }
func (s *Store) DeleteWithContext(parent context.Context, key string) error {
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	err := s.Client.Eval(ctx, `redis.call('DEL',KEYS[1]); redis.call('ZREM',KEYS[2],KEYS[1]); return 1`, []string{key, registry}).Err()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}
func (s *Store) ResetWithContext(context.Context) error {
	return errors.New("bulk session deletion is disabled")
}
func (s *Store) Reset() error { return errors.New("bulk session deletion is disabled") }
func (s *Store) Close() error { return nil } // The application owns the client.
func WriteCheck(client redis.UniversalClient, key string) func(context.Context) error {
	return func(ctx context.Context) error { return client.Set(ctx, key, "ready", 5*time.Second).Err() }
}
