package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

func TestAuthBodyRejectedFromHeadersBeforeReadingPayload(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{BodyLimit: 6 << 20, ReadTimeout: time.Second})
	app.Use(func(c fiber.Ctx) error { return c.SendStatus(204) })
	_ = app.Handler()
	server := app.Server()
	server.HeaderReceived = AuthBodyConfig
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Shutdown() })
	for _, path := range []string{"/login", "/LOGIN", "/login/", "/register", "/REGISTER", "/register/", "/register/confirm?token=test", "/%6cogin"} {
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		_, err = fmt.Fprintf(conn, "POST %s HTTP/1.1\r\nHost: localhost\r\nContent-Length: 1048576\r\nConnection: close\r\n\r\n", path)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			conn.Close()
			t.Fatal(err)
		}
		res.Body.Close()
		conn.Close()
		if res.StatusCode != 413 {
			t.Fatalf("%s buffered/waited for oversized auth body: %d", path, res.StatusCode)
		}
	}
	var header fasthttp.RequestHeader
	header.SetMethod("POST")
	header.SetRequestURI("/profile/photo")
	if AuthBodyConfig(&header).MaxRequestBodySize != 0 {
		t.Fatal("upload limit changed")
	}
}
