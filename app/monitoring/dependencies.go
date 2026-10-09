package monitoring

import (
	"context"
	"sync"
	"time"
)

// DependencyResult comunica disponibilidad y latencia sin revelar errores internos.
type DependencyResult struct {
	Status    string  `json:"status"`
	LatencyMS float64 `json:"latency_ms"`
}

var dependencyChecks struct {
	sync.RWMutex
	postgres, redis ReadinessCheck
}

// SetDependencyChecks configura las mismas comprobaciones autenticadas de /readyz.
func SetDependencyChecks(postgres, redis ReadinessCheck) {
	dependencyChecks.Lock()
	defer dependencyChecks.Unlock()
	dependencyChecks.postgres, dependencyChecks.redis = postgres, redis
}

// CheckDependencies ejecuta SELECT 1 y PING en paralelo bajo un plazo de dos segundos.
func CheckDependencies(parent context.Context) map[string]DependencyResult {
	dependencyChecks.RLock()
	pg, rd := dependencyChecks.postgres, dependencyChecks.redis
	dependencyChecks.RUnlock()
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	type result struct {
		name  string
		value DependencyResult
	}
	done := make(chan result, 2)
	for name, check := range map[string]ReadinessCheck{"postgres": pg, "redis": rd} {
		go func(name string, check ReadinessCheck) {
			started := time.Now()
			status := "unavailable"
			if check != nil && check(ctx) == nil {
				status = "available"
			}
			done <- result{name, DependencyResult{Status: status, LatencyMS: float64(time.Since(started)) / float64(time.Millisecond)}}
		}(name, check)
	}
	out := map[string]DependencyResult{"postgres": {Status: "unavailable"}, "redis": {Status: "unavailable"}}
	for n := 0; n < 2; n++ {
		select {
		case r := <-done:
			out[r.name] = r.value
		case <-ctx.Done():
			return out
		}
	}
	return out
}
