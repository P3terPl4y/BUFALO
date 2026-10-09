// Package maildelivery sends SMTP with real network deadlines and cancellation.
package maildelivery

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"time"

	frameworkmail "github.com/goravel/framework/mail"
)

type Config struct {
	Host, Username, Password string
	Port                     int
}

func Send(ctx context.Context, cfg Config, from, to, subject, html, text string) error {
	if cfg.Host == "" || cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("SMTP is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)))
	if err != nil {
		return err
	}
	defer conn.Close()
	rawConn := conn
	stop := context.AfterFunc(ctx, func() { _ = rawConn.Close() })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return err
		}
	}
	tlsConfig := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	if cfg.Port == 465 {
		secure := tls.Client(conn, tlsConfig)
		if err := secure.HandshakeContext(ctx); err != nil {
			return err
		}
		conn = secure
	}
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if cfg.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return err
			}
		} else if cfg.Port == 587 {
			return fmt.Errorf("SMTP requires STARTTLS")
		}
	}
	if cfg.Username != "" {
		if _, secure := client.TLSConnectionState(); !secure {
			ip := net.ParseIP(cfg.Host)
			if cfg.Host != "localhost" && (ip == nil || !ip.IsLoopback()) {
				return fmt.Errorf("SMTP authentication requires TLS")
			}
		}

		var auth smtp.Auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if ok, mechanisms := client.Extension("AUTH"); ok && !containsPlain(mechanisms) {
			auth = frameworkmail.LoginAuth(cfg.Username, cfg.Password)
		}
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	message := frameworkmail.NewEmail()
	message.From, message.To, message.Subject = from, []string{to}, subject
	message.HTML, message.Text = []byte(html), []byte(text)
	data, err := message.Bytes()
	if err != nil {
		return err
	}
	// Parse the envelope independently from the display-name header.
	sender, err := envelopeAddress(from)
	if err != nil {
		return err
	}
	receiver, err := envelopeAddress(to)
	if err != nil {
		return err
	}
	if err := client.Mail(sender); err != nil {
		return err
	}
	if err := client.Rcpt(receiver); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
