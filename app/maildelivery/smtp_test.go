package maildelivery

import (
	"bufio"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSilentSMTPIsCancelled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(io.Discard, conn)
	}()
	host, raw, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(raw)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := Send(ctx, Config{Host: host, Port: port}, "from@test.com", "to@test.com", "test", "<p>test</p>", "test"); err == nil {
		t.Fatal("silent SMTP succeeded")
	}
	if time.Since(started) > time.Second {
		t.Fatal("SMTP deadline did not close network connection")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SMTP connection leaked")
	}
}
func TestSMTPDeliversMessage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	message := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		reader := bufio.NewReader(conn)
		_, _ = io.WriteString(conn, "220 localhost ready\r\n")
		var data strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				_, _ = io.WriteString(conn, "250 localhost\r\n")
			case strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
				_, _ = io.WriteString(conn, "250 OK\r\n")
			case strings.HasPrefix(line, "DATA"):
				_, _ = io.WriteString(conn, "354 data\r\n")
				for {
					line, err = reader.ReadString('\n')
					if err != nil {
						return
					}
					if line == ".\r\n" {
						break
					}
					data.WriteString(line)
				}
				_, _ = io.WriteString(conn, "250 stored\r\n")
				message <- data.String()
			case strings.HasPrefix(line, "QUIT"):
				_, _ = io.WriteString(conn, "221 bye\r\n")
				return
			default:
				return
			}
		}
	}()
	host, raw, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(raw)
	if err := Send(context.Background(), Config{Host: host, Port: port}, "BUFALO <from@test.com>", "to@test.com", "Confirmación", "<p>confirm</p>", "confirm"); err != nil {
		t.Fatal(err)
	}
	select {
	case data := <-message:
		if !strings.Contains(data, "multipart/alternative") || !strings.Contains(data, "confirm") {
			t.Fatal("message content lost")
		}
	case <-time.After(time.Second):
		t.Fatal("missing message")
	}
}
