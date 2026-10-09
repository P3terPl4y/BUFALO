package ingress

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClientIdentityTrust(t *testing.T) {
	h := New(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), nil, []string{"172.30.113.1"})
	for _, test := range []struct{ peer, header, want string }{{"203.0.113.1:123", "198.51.100.2", "203.0.113.1"}, {"127.0.0.1:123", "198.51.100.2", "198.51.100.2"}, {"172.30.113.1:123", "::ffff:198.51.100.2", "198.51.100.2"}, {"127.0.0.1:123", "invalid", "127.0.0.1"}} {
		r := httptest.NewRequest("GET", "/login", nil)
		r.RemoteAddr = test.peer
		r.Header.Set("CF-Connecting-IP", test.header)
		if got := h.clientIP(r); got != test.want {
			t.Fatalf("identity %q != %q", got, test.want)
		}
	}
}
func TestOversizedAliasesAndChunkedRejected(t *testing.T) {
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), nil, nil)
	for _, path := range []string{"/login", "/LOGIN", "/login/", "/register/", "/register/confirm/"} {
		for _, chunked := range []bool{false, true} {
			r := httptest.NewRequest("POST", path, strings.NewReader(strings.Repeat("x", 16385)))
			if chunked {
				r.ContentLength = -1
				r.TransferEncoding = []string{"chunked"}
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 413 {
				t.Fatalf("path %s chunked %v: %d", path, chunked, w.Code)
			}
		}
	}
}
func TestSlowBodiesDoNotBlockHealthOrAnotherIP(t *testing.T) {
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }), nil, nil)
	server := httptest.NewUnstartedServer(h)
	server.Config.ReadHeaderTimeout = time.Second
	server.Config.ReadTimeout = 6 * time.Second
	server.Start()
	defer server.Close()
	var connections []net.Conn
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
	}()
	for i := 0; i < 64; i++ {
		conn, err := net.Dial("tcp", server.Listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		connections = append(connections, conn)
		_, _ = fmt.Fprint(conn, "POST /login HTTP/1.1\r\nHost: localhost\r\nCF-Connecting-IP: 198.18.1.1\r\nContent-Length: 1\r\n\r\n")
	}
	time.Sleep(100 * time.Millisecond)
	for _, path := range []string{"/healthz", "/login"} {
		r, _ := http.NewRequest("GET", server.URL+path, nil)
		r.Header.Set("CF-Connecting-IP", "198.18.1.2")
		client := http.Client{Timeout: time.Second}
		res, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("slow clients blocked %s: %d", path, res.StatusCode)
		}
	}
}
func TestGlobalGateDoesNotQueue(t *testing.T) {
	entered := make(chan struct{}, RequestCapacity)
	release := make(chan struct{})
	defer close(release)
	h := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-release; w.WriteHeader(200) }), nil, nil)
	var wg sync.WaitGroup
	for i := 0; i < RequestCapacity; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/login", nil)
			r.RemoteAddr = fmt.Sprintf("198.18.1.%d:123", i+1)
			h.ServeHTTP(httptest.NewRecorder(), r)
		}(i)
	}
	for i := 0; i < RequestCapacity; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("capacity not filled")
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/login", nil))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
	// Release and drain explicitly before waiting on workers.
	for i := 0; i < RequestCapacity; i++ {
		release <- struct{}{}
	}
	wg.Wait()
}
func TestOversizedFromHeadersDoesNotWaitForBody(t *testing.T) {
	server := httptest.NewServer(New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }), nil, nil))
	defer server.Close()
	conn, err := net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	_, _ = io.WriteString(conn, "POST /LOGIN/ HTTP/1.1\r\nHost: localhost\r\nContent-Length: 1048576\r\n\r\n")
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 413 {
		t.Fatal(response.StatusCode)
	}
}

func TestHealthAliasesNeverReadBodiesOrCreateApplicationSessions(t *testing.T) {
	calls := 0
	handler := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/healthz" {
			t.Error(r.URL.Path)
		}
		w.WriteHeader(200)
	}), nil, nil)
	for _, path := range []string{"/HEALTHZ/", "/healthz"} {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	for _, chunked := range []bool{false, true} {
		r := httptest.NewRequest("GET", "/READYZ/", nil)
		if chunked {
			r.TransferEncoding = []string{"chunked"}
			r.ContentLength = -1
		} else {
			r.ContentLength = 1 << 30
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 413 {
			t.Fatal(w.Code)
		}
	}
	if calls != 2 {
		t.Fatal("health body reached application", calls)
	}
}

type cachedAbuseGuard struct{}

func (cachedAbuseGuard) RejectCachedHTTP(w http.ResponseWriter, _ *http.Request, ip string) bool {
	if ip == "198.18.1.1" {
		w.WriteHeader(429)
		return true
	}
	return false
}
func (g cachedAbuseGuard) AllowHTTP(w http.ResponseWriter, r *http.Request, ip string, _ bool) bool {
	return !g.RejectCachedHTTP(w, r, ip)
}
func TestCachedAbuseCannotSpendAdmissionBudget(t *testing.T) {
	h := New(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }), cachedAbuseGuard{}, nil)
	for i := 0; i < 2000; i++ {
		req := httptest.NewRequest("GET", "/login", nil)
		req.RemoteAddr = "198.18.1.1:123"
		out := httptest.NewRecorder()
		h.ServeHTTP(out, req)
		if out.Code != 429 {
			t.Fatal("known ban not rejected")
		}
	}
	req := httptest.NewRequest("GET", "/login", nil)
	req.RemoteAddr = "198.18.1.2:123"
	out := httptest.NewRecorder()
	h.ServeHTTP(out, req)
	if out.Code != 200 {
		t.Fatalf("banned traffic starved independent client: %d", out.Code)
	}
}
