package controllers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"goravel/app/facades"
	"goravel/app/http/middleware"
	"goravel/app/models"
	"goravel/routes"
	"goravel/tests"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
	redisstore "github.com/gofiber/storage/redis/v3"
	"github.com/redis/go-redis/v9"
)

func isolatedAuthApp(t *testing.T) *fiber.App {
	t.Helper()
	path, err := exec.LookPath("redis-server")
	if err != nil {
		t.Fatal("Redis real is required for this security test")
	}
	dir, err := os.MkdirTemp("", "ba-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := dir + "/auth.sock"
	cmd := exec.Command(path, "--port", "0", "--unixsocket", socket, "--unixsocketperm", "700", "--save", "", "--appendonly", "no")
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	client := redis.NewClient(&redis.Options{Network: "unix", Addr: socket, MaxRetries: -1, DialTimeout: time.Second})
	t.Cleanup(func() { _ = client.Close() })
	deadline := time.Now().Add(3 * time.Second)
	for client.Ping(context.Background()).Err() != nil {
		if time.Now().After(deadline) {
			t.Fatal("isolated Redis unavailable")
		}
		time.Sleep(10 * time.Millisecond)
	}
	storage := redisstore.NewFromConnection(client)
	app := fiber.New(fiber.Config{Views: testApp.Config().Views, ErrorHandler: middleware.RenderError})
	app.Use(middleware.ErrorHandler())
	sessionHandler, sessionStore := session.NewWithStore(session.Config{Storage: storage, Extractor: extractors.FromCookie("session_id"), CookieHTTPOnly: true, CookieSameSite: "Lax", IdleTimeout: 30 * time.Minute, AbsoluteTimeout: 24 * time.Hour})
	app.Use(sessionHandler)
	app.Use(csrf.New(csrf.Config{Session: sessionStore, Extractor: extractors.FromForm("_csrf"), CookieHTTPOnly: true, ErrorHandler: middleware.RenderError}))
	routes.SetupWebRoutesWithEmailSender(app, middleware.NewRedisRateCounter(client), func(string, string) error { return nil }, func(c fiber.Ctx) error { return c.Next() })
	return app
}

func authRequest(t *testing.T, app *fiber.App, method, path string, form url.Values, cookies []*http.Cookie) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(form.Encode()))
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	res, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return res, string(body)
}

var csrfPattern = regexp.MustCompile(`name="_csrf"\s+value="([^"]+)"`)

func loginForm(t *testing.T, app *fiber.App) (string, []*http.Cookie) {
	t.Helper()
	res, body := authRequest(t, app, "GET", "/login", nil, nil)
	match := csrfPattern.FindStringSubmatch(body)
	if res.StatusCode != 200 || len(match) != 2 {
		t.Fatalf("missing login CSRF: status=%d", res.StatusCode)
	}
	return match[1], res.Cookies()
}

func seedSecurityUser(t *testing.T, email string, active bool) {
	t.Helper()
	user := models.User{Name: "Security user", Email: email, Role: "admin", IsActive: true}
	if err := user.SetPassword("correct-password"); err != nil {
		t.Fatal(err)
	}
	if err := facades.Orm().Query().Create(&user); err != nil {
		t.Fatal(err)
	}
	if !active {
		if _, err := facades.Orm().Query().Model(&models.User{}).Where("id = ?", user.ID).Update("is_active", false); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoginRotatesRedisSessionAndRejectsCSRF(t *testing.T) {
	tests.ResetDB(t)
	seedSecurityUser(t, "login@test.com", true)
	app := isolatedAuthApp(t)
	token, cookies := loginForm(t, app)
	form := url.Values{"email": {" LOGIN@test.com "}, "password": {"correct-password"}}
	res, _ := authRequest(t, app, "POST", "/login", form, cookies)
	if res.StatusCode != 403 {
		t.Fatalf("missing CSRF accepted: %d", res.StatusCode)
	}
	form.Set("_csrf", "wrong")
	res, _ = authRequest(t, app, "POST", "/login", form, cookies)
	if res.StatusCode != 403 {
		t.Fatalf("invalid CSRF accepted: %d", res.StatusCode)
	}
	form.Set("_csrf", token)
	res, _ = authRequest(t, app, "POST", "/login", form, cookies)
	if res.StatusCode != 303 || res.Header.Get("Location") != "/home" {
		t.Fatalf("normalized login failed: %d", res.StatusCode)
	}
	value := func(cookies []*http.Cookie) string {
		for _, c := range cookies {
			if c.Name == "session_id" {
				return c.Value
			}
		}
		return ""
	}
	if value(cookies) == "" || value(res.Cookies()) == "" || value(cookies) == value(res.Cookies()) {
		t.Fatal("session ID not rotated")
	}
	old, _ := authRequest(t, app, "GET", "/admin/health/data", nil, cookies)
	if old.StatusCode != 303 {
		t.Fatalf("old session retained access: %d", old.StatusCode)
	}
	current, body := authRequest(t, app, "GET", "/admin/health/data", nil, res.Cookies())
	if current.StatusCode != 200 || !strings.Contains(body, `"runtime"`) {
		t.Fatalf("new session not authorized: %d", current.StatusCode)
	}
}

func TestLoginUsesGenericResponseForUnknownWrongAndDisabledAccounts(t *testing.T) {
	tests.ResetDB(t)
	seedSecurityUser(t, "active@test.com", true)
	seedSecurityUser(t, "disabled@test.com", false)
	app := isolatedAuthApp(t)
	for _, tc := range []struct{ email, password string }{{"unknown@test.com", "correct-password"}, {"active@test.com", "incorrect"}, {"disabled@test.com", "correct-password"}, {"' OR 1=1 --", "correct-password"}, {"active@test.com", strings.Repeat("x", 73)}} {
		token, cookies := loginForm(t, app)
		res, body := authRequest(t, app, "POST", "/login", url.Values{"email": {tc.email}, "password": {tc.password}, "_csrf": {token}}, cookies)
		if res.StatusCode != 200 || strings.Contains(body, "Usuario deshabilitado") || res.Header.Get("Location") == "/home" {
			t.Fatalf("unsafe authentication failure for %q: status=%d", tc.email, res.StatusCode)
		}
		if len(tc.password) <= 72 && !strings.Contains(body, "Credenciales incorrectas") {
			t.Fatalf("non-generic credentials response for %q", tc.email)
		}
	}
}

func TestConcurrentLoginDoesNotMixRedisSessions(t *testing.T) {
	tests.ResetDB(t)
	seedSecurityUser(t, "parallel@test.com", true)
	app := isolatedAuthApp(t)
	var wg sync.WaitGroup
	// Four concurrent operations are the authentication admission budget.
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, cookies := loginForm(t, app)
			res, _ := authRequest(t, app, "POST", "/login", url.Values{"email": {"parallel@test.com"}, "password": {"correct-password"}, "_csrf": {token}}, cookies)
			if res.StatusCode != 303 {
				t.Errorf("concurrent login status=%d", res.StatusCode)
				return
			}
			res, _ = authRequest(t, app, "GET", "/admin/health/data", nil, res.Cookies())
			if res.StatusCode != 200 {
				t.Errorf("concurrent session lost: status=%d", res.StatusCode)
			}
		}()
	}
	wg.Wait()
}

func TestRegisterAndConfirmRequireCSRFAndRejectPrivilegeEscalation(t *testing.T) {
	tests.ResetDB(t)
	app := isolatedAuthApp(t)
	for _, path := range []string{"/register", "/register/confirm"} {
		res, _ := authRequest(t, app, "POST", path, url.Values{"token": {strings.Repeat("a", 43)}}, nil)
		if res.StatusCode != 403 {
			t.Fatalf("CSRF bypass on %s: %d", path, res.StatusCode)
		}
	}
	token, cookies := loginForm(t, app)
	res, body := authRequest(t, app, "POST", "/register", url.Values{"role": {"admin"}, "_csrf": {token}}, cookies)
	if res.StatusCode != 200 || !strings.Contains(body, "tipo de cuenta válido") || tests.CountUsers(t) != 0 {
		t.Fatal("public admin registration was not rejected")
	}
	res, _ = authRequest(t, app, "POST", "/register/confirm", url.Values{"token": {strings.Repeat("a", 43)}, "_csrf": {token}}, cookies)
	if res.StatusCode != 200 || tests.CountUsers(t) != 0 {
		t.Fatal("forged confirmation created a user")
	}
}
