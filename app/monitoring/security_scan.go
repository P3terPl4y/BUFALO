package monitoring

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

type SecurityScanSnapshot struct {
	Mode             string  `json:"mode"`
	Status           string  `json:"status"`
	StartedAt        string  `json:"started_at"`
	FinishedAt       string  `json:"finished_at"`
	MemoryPeakBytes  uint64  `json:"memory_peak_bytes"`
	MemoryLimitBytes uint64  `json:"memory_limit_bytes"`
	CPUSeconds       float64 `json:"cpu_seconds"`
	DurationSeconds  float64 `json:"duration_seconds"`
	ExitCode         *int    `json:"exit_code"`
	Result           string  `json:"result"`
}

var securityScanCache struct {
	sync.Mutex
	at    time.Time
	value SecurityScanSnapshot
}

func SecurityScan() SecurityScanSnapshot {
	securityScanCache.Lock()
	defer securityScanCache.Unlock()
	if time.Since(securityScanCache.at) < time.Second {
		return securityScanCache.value
	}
	value := SecurityScanSnapshot{Status: "not_run"}
	file, err := os.Open("storage/security-scan/latest.json")
	if err == nil {
		defer file.Close()
		if json.NewDecoder(io.LimitReader(file, 65536)).Decode(&value) != nil {
			value = SecurityScanSnapshot{Status: "unavailable"}
		}
	}
	securityScanCache.at = time.Now()
	securityScanCache.value = value
	return value
}
