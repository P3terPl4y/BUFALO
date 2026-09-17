package feature

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/bootstrap"
	"goravel/routes"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v3"
)

var (
	testApp    *fiber.App
	testServer *httptest.Server
)

func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func findViewsDir() string {
	root := projectRoot()
	candidates := []string{
		filepath.Join(root, "app", "views"),
		filepath.Join(root, "resources", "views"),
		filepath.Join(root, "resources", "templates"),
		filepath.Join(root, "views"),
		filepath.Join(root, "templates"),
	}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			var found string
			_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
					found = path
					return io.EOF
				}
				return nil
			})
			if found != "" {
				return dir
			}
		}
	}
	return candidates[0]
}

func TestMain(m *testing.M) {
	os.Setenv("APP_ENV", "testing")
	os.Setenv("MAIL_MAILER", "log")
	os.Setenv("SESSION_DRIVER", "memory")
	os.Setenv("APP_KEY", "base64:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")

	app := bootstrap.Boot()
	app.Boot()

	viewsDir := findViewsDir()
	println(">>> viewsDir:", viewsDir)

	engine := html.New(viewsDir, ".html")
	engine.Reload(true)

	testApp = fiber.New(fiber.Config{Views: engine})

	// ── MIDDLEWARES GLOBALES (Fiber v3 API) ──
	testApp.Use(recover.New())
	testApp.Use(logger.New())
	testApp.Use(session.New(session.Config{
		Extractor:      extractors.FromCookie("session_id"), // ← v3: reemplaza KeyLookup
		CookieSecure:   false,                               // httptest usa http://
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		IdleTimeout:    30 * time.Minute,                    // ← v3: reemplaza Expiration
		AbsoluteTimeout: 24 * time.Hour,                     // debe ser >= IdleTimeout
	}))
	// ────────────────────────────────────────

	routes.SetupWebRoutes(testApp)

	testServer = httptest.NewServer(adaptor.FiberApp(testApp))
	defer testServer.Close()

	os.Exit(m.Run())
}

// ── Cliente HTTP con cookie jar ──

func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func seedUser(t *testing.T, name, email, password, role string) *models.User {
	t.Helper()
	svc := services.NewUserService()
	u := &models.User{
		Name:     name,
		Email:    email,
		Role:     role,
		IsActive: true,
	}
	if err := u.SetPassword(password); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if err := svc.Create(u); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	return u
}

func login(t *testing.T, email, password string) *http.Client {
	t.Helper()
	client := newClient()
	form := url.Values{}
	form.Set("email", email)
	form.Set("password", password)

	resp, err := client.Post(
		testServer.URL+"/login",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("login failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	return client
}

func postForm(t *testing.T, client *http.Client, path string, form map[string]string) *http.Response {
	t.Helper()
	values := url.Values{}
	for k, v := range form {
		values.Set(k, v)
	}
	resp, err := client.Post(
		testServer.URL+path,
		"application/x-www-form-urlencoded",
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	return resp
}

func get(t *testing.T, client *http.Client, path string) *http.Response {
	t.Helper()
	resp, err := client.Get(testServer.URL + path)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	return string(b)
}
