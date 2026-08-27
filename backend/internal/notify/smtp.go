package notify

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/config"
)

const sendTimeout = 10 * time.Second

type SMTP struct {
	addr string
	from string
	auth smtp.Auth
}

func NewSMTP(cfg config.MailConfig) *SMTP {
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	return &SMTP{
		addr: net.JoinHostPort(cfg.Host, cfg.Port),
		from: cfg.From,
		auth: auth,
	}
}

func (s *SMTP) Send(ctx context.Context, m Message) error {
	// One budget for the whole exchange. Giving the dialer its own timeout and
	// then starting the deadline after dialing lets a slow relay cost both.
	deadline := time.Now().Add(sendTimeout)

	dialer := &net.Dialer{Deadline: deadline}
	conn, err := dialer.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", s.addr, err)
	}
	_ = conn.SetDeadline(deadline)

	host, _, _ := net.SplitHostPort(s.addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer func() { _ = client.Close() }()

	if s.auth != nil {
		if err := client.Auth(s.auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp from: %w", err)
	}
	if err := client.Rcpt(m.To); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(s.compose(m)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close body: %w", err)
	}

	return client.Quit()
}

func (s *SMTP) compose(m Message) []byte {
	var b strings.Builder

	// Every header value is sanitised, not just the subject. Subjects are
	// compile-time constants; To is user-chosen. Today stdlib's Rcpt would
	// reject a CRLF address before Data() is ever called, so this is defence
	// in depth - but that safety is a property of stdlib's call ordering, not
	// of anything here.
	fmt.Fprintf(&b, "From: %s\r\n", sanitizeHeader(s.from))
	fmt.Fprintf(&b, "To: %s\r\n", sanitizeHeader(m.To))
	fmt.Fprintf(&b, "Subject: %s\r\n", sanitizeHeader(m.Subject))
	// Mailpit synthesises both, which is why local testing never noticed them
	// missing. Real MTAs and spam filters penalise a message with no Date.
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@%s>\r\n", uuid.NewString(), s.domain())
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(m.Body)

	return []byte(b.String())
}

// domain is the right-hand side of the From address, for Message-ID. It
// tolerates a "Name <user@host>" From, and falls back rather than emitting a
// malformed header.
func (s *SMTP) domain() string {
	_, host, found := strings.Cut(s.from, "@")
	host = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(host), ">"))
	if !found || host == "" {
		return "localhost"
	}
	return sanitizeHeader(host)
}

func sanitizeHeader(v string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(v)
}
