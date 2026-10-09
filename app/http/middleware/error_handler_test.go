package middleware

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/template/html/v3"
	"github.com/jackc/pgx/v5/pgconn"
	"goravel/app/viewhelpers"
)

func TestErrorPageCoversImportantHTTPStatuses(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 405, 408, 409, 413, 419, 422, 429, 500, 502, 503, 504} {
		page, ok := errorPage(status)
		if !ok || page.Title == "" || page.Message == "" || page.Detail == "" {
			t.Errorf("status %d has incomplete error page: %#v, found=%t", status, page, ok)
		}
	}
}

func TestCSRFRejectionsKeepForbiddenStatus(t *testing.T) {
	for _, err := range []error{csrf.ErrTokenNotFound, csrf.ErrTokenInvalid,
		csrf.ErrFetchSiteInvalid, csrf.ErrRefererNotFound, csrf.ErrRefererInvalid,
		csrf.ErrRefererNoMatch, csrf.ErrOriginInvalid, csrf.ErrOriginNoMatch} {
		if got := errorStatus(err); got != fiber.StatusForbidden {
			t.Errorf("errorStatus(%v) = %d, want %d", err, got, fiber.StatusForbidden)
		}
	}
}

func TestTransientDatabaseFailuresMapToServiceUnavailable(t *testing.T) {
	transient := []error{
		driver.ErrBadConn,
		&pgconn.PgError{Code: "57014"},
		&pgconn.PgError{Code: "55P03"},
		sql.ErrConnDone,
		context.DeadlineExceeded,
		&net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED},
	}
	for _, err := range transient {
		if got := errorStatus(err); got != fiber.StatusServiceUnavailable {
			t.Errorf("errorStatus(%T) = %d, want %d", err, got, fiber.StatusServiceUnavailable)
		}
	}
	if got := errorStatus(errors.New("duplicate key violates unique constraint")); got != fiber.StatusInternalServerError {
		t.Fatalf("database domain error mapped to %d, want 500", got)
	}
}

func TestErrorPageDoesNotExposeUnknownStatuses(t *testing.T) {
	if _, ok := errorPage(418); ok {
		t.Fatal("unconfigured status should use the safe 500 fallback")
	}
}

func TestProtectedPathMatcherKeepsPublicUnknownURLsPublic(t *testing.T) {
	for _, path := range []string{"/", "/unknown", "/login/typo", "/css/missing.css"} {
		if isProtectedRequestPath(path) {
			t.Errorf("%q should not be treated as protected", path)
		}
	}
	for _, path := range []string{"/home", "/loads/12", "/profile/edit", "/admin/users", "/facturas/export"} {
		if !isProtectedRequestPath(path) {
			t.Errorf("%q should remain protected", path)
		}
	}
}

func TestErrorHandlerRendersStatusAndKeepsHTTPCode(t *testing.T) {
	engine := html.New("../../../app/views", ".html")
	viewhelpers.Register(engine)
	app := fiber.New(fiber.Config{Views: engine})
	app.Use(ErrorHandler())
	app.Get("/missing", func(c fiber.Ctx) error { return fiber.NewError(fiber.StatusNotFound) })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/missing", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected HTTP 404, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Página no encontrada") || !strings.Contains(string(body), "404") {
		t.Fatalf("expected branded 404 page, got %q", string(body))
	}
}
