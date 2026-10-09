// Package ingress bounds network and body work before Fiber buffers a request.
package ingress

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"goravel/app/http/middleware"
	"goravel/app/monitoring"
)

const RequestCapacity = 64
const UploadCapacity = 2
const PerIPCapacity = 8
const FormBodyLimit = 256 << 10
const UploadBodyLimit = 6 << 20

type Guard interface {
	AllowHTTP(http.ResponseWriter, *http.Request, string, bool) bool
}
type Handler struct {
	next           http.Handler
	guard          Guard
	slots, uploads chan struct{}
	mu             sync.Mutex
	active         map[string]int
	proxies        map[netip.Addr]bool
	tokens         float64
	at             time.Time
}

func New(next http.Handler, guard Guard, proxies []string) *Handler {
	h := &Handler{next: next, guard: guard, slots: make(chan struct{}, RequestCapacity), uploads: make(chan struct{}, UploadCapacity), active: make(map[string]int), proxies: make(map[netip.Addr]bool), tokens: 400, at: time.Now()}
	for _, raw := range proxies {
		if ip, err := netip.ParseAddr(raw); err == nil {
			h.proxies[ip.Unmap()] = true
		}
	}
	return h
}
func (h *Handler) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "unknown"
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return "unknown"
	}
	peer = peer.Unmap()
	if peer.IsLoopback() || h.proxies[peer] {
		if values := r.Header.Values("CF-Connecting-IP"); len(values) == 1 {
			if ip, err := netip.ParseAddr(values[0]); err == nil && !ip.IsUnspecified() && !ip.IsMulticast() {
				return ip.Unmap().String()
			}
		}
	}
	return peer.String()
}
func reject(w http.ResponseWriter, r *http.Request, status int, outcome string) {
	monitoring.RecordTraffic(outcome)
	monitoring.RecordAttack(outcome, status, r.Header.Get("CF-Connecting-IP"))
	w.Header().Set("Connection", "close")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", "1")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.Error(w, http.StatusText(status), status)
}
func (h *Handler) acquire(ip string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	h.tokens = min(400, h.tokens+now.Sub(h.at).Seconds()*200)
	h.at = now
	if h.tokens < 1 || h.active[ip] >= PerIPCapacity {
		return false
	}
	h.tokens--
	h.active[ip]++
	return true
}
func (h *Handler) release(ip string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.active[ip]--
	if h.active[ip] == 0 {
		delete(h.active, ip)
	}
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := middleware.CanonicalPath(r.URL.Path)
	ip := h.clientIP(r)
	// Freeze one validated identity for both the ingress and trusted Fiber proxy.
	// For untrusted peers Fiber ignores this header and uses the same peer IP.
	r.Header.Set("CF-Connecting-IP", ip)
	if (path == "/healthz" || path == "/readyz") && (r.Method == "GET" || r.Method == "HEAD") {
		if r.ContentLength > 0 || len(r.TransferEncoding) > 0 || r.ContentLength < 0 {
			reject(w, r, 413, "body_rejected")
			return
		}
		r.URL.Path = path
		h.next.ServeHTTP(w, r)
		return
	}

	// Known bans must not spend tokens needed by independent identities.
	if cached, ok := h.guard.(interface {
		RejectCachedHTTP(http.ResponseWriter, *http.Request, string) bool
	}); ok && cached.RejectCachedHTTP(w, r, ip) {
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		reject(w, r, 503, "capacity")
		return
	}
	if !h.acquire(ip) {
		reject(w, r, 429, "capacity")
		return
	}
	defer h.release(ip)
	static := (r.Method == "GET" || r.Method == "HEAD") && (strings.HasPrefix(path, "/css/") || strings.HasPrefix(path, "/js/") || strings.HasPrefix(path, "/img/") || strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/fonts/") || path == "/favicon.ico" || path == "/robots.txt")
	if h.guard != nil && !h.guard.AllowHTTP(w, r, ip, static) {
		w.Header().Set("Connection", "close")
		return
	}
	limit := int64(FormBodyLimit)
	readBudget := 5 * time.Second
	if path == "/login" || path == "/register" || path == "/register/confirm" {
		limit = 16 << 10
	}
	if path == "/profile/photo" && r.Method == "POST" {
		limit = UploadBodyLimit
		readBudget = 20 * time.Second
		select {
		case h.uploads <- struct{}{}:
			defer func() { <-h.uploads }()
		default:
			reject(w, r, 503, "capacity")
			return
		}
	}
	if r.ContentLength > limit {
		reject(w, r, 413, "body_rejected")
		return
	}
	if r.Body != nil && r.Body != http.NoBody {
		controller := http.NewResponseController(w)
		_ = controller.SetReadDeadline(time.Now().Add(readBudget))
		body := http.MaxBytesReader(w, r.Body, limit)
		data, err := io.ReadAll(body)
		_ = body.Close()
		if err != nil {
			var large *http.MaxBytesError
			var timeout net.Error
			switch {
			case errors.As(err, &large):
				reject(w, r, 413, "body_rejected")
			case errors.As(err, &timeout) && timeout.Timeout():
				reject(w, r, 408, "read_timeout")
			default:
				reject(w, r, 400, "body_rejected")
			}
			return
		}
		_ = controller.SetReadDeadline(time.Time{})
		r.Body = io.NopCloser(bytes.NewReader(data))
		r.ContentLength = int64(len(data))
		r.Header.Del("Content-Length")
		r.TransferEncoding = nil
	}
	h.next.ServeHTTP(w, r)
}

// LimitListener never queues accepted connections in userspace. Header reads
// have their own deadline; data-plane capacity cannot consume the health port.
func LimitListener(listener net.Listener, max int) net.Listener {
	return &limitedListener{Listener: listener, max: int64(max)}
}

type limitedListener struct {
	net.Listener
	active atomic.Int64
	max    int64
}
type limitedConn struct {
	net.Conn
	once  sync.Once
	owner *limitedListener
}

func (c *limitedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.owner.active.Add(-1) })
	return err
}
func (l *limitedListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if l.active.Add(1) <= l.max {
			return &limitedConn{Conn: conn, owner: l}, nil
		}
		l.active.Add(-1)
		monitoring.RecordTraffic("transport_rejected")
		host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		monitoring.RecordAttack("transport_rejected", 503, host)
		_ = conn.Close()
	}
}
