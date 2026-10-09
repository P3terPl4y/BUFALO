package monitoring

import (
	"sync"
	"time"
)

const RequestHistoryLimit = 500
const ErrorHistoryLimit = 200
const TrendMinutes = 60

// RequestEvent guarda solo el patrón de ruta y el resultado HTTP, nunca la URL,
// parámetros, cuerpos, IP, identidad de usuario ni errores internos.
type RequestEvent struct {
	ID         uint64    `json:"id"`
	At         time.Time `json:"at"`
	Method     string    `json:"method"`
	Route      string    `json:"route"`
	Status     int       `json:"status"`
	DurationMS float64   `json:"duration_ms"`
}

// TrendPoint agrupa el tráfico de aplicación por minuto.
type TrendPoint struct {
	At           time.Time `json:"at"`
	Requests     uint64    `json:"requests"`
	ClientErrors uint64    `json:"client_errors"`
	ServerErrors uint64    `json:"server_errors"`
}

// HistorySnapshot es una copia independiente de la ventana acotada del proceso.
type HistorySnapshot struct {
	Requests     []RequestEvent `json:"requests"`
	Errors       []RequestEvent `json:"errors"`
	Trend        []TrendPoint   `json:"trend"`
	RequestLimit int            `json:"request_limit"`
	ErrorLimit   int            `json:"error_limit"`
	StartedAt    time.Time      `json:"started_at"`
}

type eventHistory struct {
	mu                       sync.RWMutex
	nextID                   uint64
	requests                 [RequestHistoryLimit]RequestEvent
	errors                   [ErrorHistoryLimit]RequestEvent
	requestCount, errorCount int
	trend                    [TrendMinutes]TrendPoint
}

var history eventHistory

func (h *eventHistory) record(event RequestEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	event.ID = h.nextID
	h.requests[(h.nextID-1)%RequestHistoryLimit] = event
	if h.requestCount < RequestHistoryLimit {
		h.requestCount++
	}
	if event.Status >= 400 {
		h.errors[h.errorCount%ErrorHistoryLimit] = event
		h.errorCount++
	}
	minute := event.At.UTC().Truncate(time.Minute)
	i := int(minute.Unix()/60) % TrendMinutes
	if !h.trend[i].At.Equal(minute) {
		h.trend[i] = TrendPoint{At: minute}
	}
	h.trend[i].Requests++
	if event.Status >= 500 {
		h.trend[i].ServerErrors++
	} else if event.Status >= 400 {
		h.trend[i].ClientErrors++
	}
}

func (h *eventHistory) snapshot(now time.Time) HistorySnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s := HistorySnapshot{Requests: make([]RequestEvent, 0, h.requestCount), Errors: make([]RequestEvent, 0, ErrorHistoryLimit), Trend: make([]TrendPoint, 0, TrendMinutes), RequestLimit: RequestHistoryLimit, ErrorLimit: ErrorHistoryLimit, StartedAt: processStartedAt}
	for n := 0; n < h.requestCount; n++ {
		s.Requests = append(s.Requests, h.requests[(h.nextID-1-uint64(n))%RequestHistoryLimit])
	}
	count := min(h.errorCount, ErrorHistoryLimit)
	for n := 0; n < count; n++ {
		s.Errors = append(s.Errors, h.errors[(h.errorCount-1-n)%ErrorHistoryLimit])
	}
	last := now.UTC().Truncate(time.Minute)
	for n := TrendMinutes - 1; n >= 0; n-- {
		at := last.Add(-time.Duration(n) * time.Minute)
		p := h.trend[int(at.Unix()/60)%TrendMinutes]
		if !p.At.Equal(at) {
			p = TrendPoint{At: at}
		}
		s.Trend = append(s.Trend, p)
	}
	return s
}

// RecentHistory devuelve las últimas 500 peticiones y 200 errores del proceso.
// La retención es en memoria y se reinicia con el servidor.
func RecentHistory() HistorySnapshot { return history.snapshot(time.Now()) }
