package notify

import (
	"strings"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/config"
)

func TestComposeStripsNewlinesFromTheSubject(t *testing.T) {
	s := NewSMTP(config.MailConfig{Host: "localhost", Port: "1025", From: "no-reply@test"})

	out := string(s.compose(Message{
		To:      "victim@example.test",
		Subject: "Order placed\r\nBcc: attacker@evil.test",
		Body:    "body",
	}))

	headers, body, found := strings.Cut(out, "\r\n\r\n")
	if !found {
		t.Fatal("no blank line between headers and body")
	}
	if body != "body" {
		t.Errorf("body = %q, want %q", body, "body")
	}

	for _, line := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "bcc:") {
			t.Errorf("a newline in the subject forged a header:\n%s", headers)
		}
	}

	subjectLines := 0
	for _, line := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(line, "Subject:") {
			subjectLines++
		}
	}
	if subjectLines != 1 {
		t.Errorf("found %d Subject lines, want exactly 1:\n%s", subjectLines, headers)
	}
}
