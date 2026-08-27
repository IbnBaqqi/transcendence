package notify

import (
	"strings"
	"testing"
	"time"

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

func TestComposeStripsNewlinesFromTheRecipient(t *testing.T) {
	s := NewSMTP(config.MailConfig{Host: "localhost", Port: "1025", From: "no-reply@test"})

	out := string(s.compose(Message{
		To:      "victim@example.test\r\nBcc: attacker@evil.test",
		Subject: "Order placed",
		Body:    "body",
	}))

	headers, _, _ := strings.Cut(out, "\r\n\r\n")
	for _, line := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "bcc:") {
			t.Errorf("a newline in the recipient forged a header:\n%s", headers)
		}
	}
}

func TestComposeSetsDateAndMessageID(t *testing.T) {
	s := NewSMTP(config.MailConfig{Host: "localhost", Port: "1025", From: "no-reply@forage.test"})

	out := string(s.compose(Message{To: "buyer@example.test", Subject: "Order placed", Body: "body"}))

	headers, _, _ := strings.Cut(out, "\r\n\r\n")

	var date, messageID string
	for _, line := range strings.Split(headers, "\r\n") {
		switch {
		case strings.HasPrefix(line, "Date: "):
			date = strings.TrimPrefix(line, "Date: ")
		case strings.HasPrefix(line, "Message-ID: "):
			messageID = strings.TrimPrefix(line, "Message-ID: ")
		}
	}

	if date == "" {
		t.Error("no Date header - Mailpit synthesises one, real MTAs penalise its absence")
	} else if _, err := time.Parse(time.RFC1123Z, date); err != nil {
		t.Errorf("Date %q does not parse as RFC 1123Z: %v", date, err)
	}

	if !strings.HasPrefix(messageID, "<") || !strings.HasSuffix(messageID, ">") {
		t.Errorf("Message-ID = %q, want it wrapped in angle brackets", messageID)
	}
	if !strings.HasSuffix(messageID, "@forage.test>") {
		t.Errorf("Message-ID = %q, want the From address's domain", messageID)
	}
}

func TestMessageIDsAreUnique(t *testing.T) {
	s := NewSMTP(config.MailConfig{Host: "localhost", Port: "1025", From: "no-reply@forage.test"})

	m := Message{To: "buyer@example.test", Subject: "Order placed", Body: "body"}
	first := string(s.compose(m))
	second := string(s.compose(m))

	if idOf(t, first) == idOf(t, second) {
		t.Error("two messages share a Message-ID")
	}
}

func idOf(t *testing.T, raw string) string {
	t.Helper()

	headers, _, _ := strings.Cut(raw, "\r\n\r\n")
	for _, line := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(line, "Message-ID: ") {
			return strings.TrimPrefix(line, "Message-ID: ")
		}
	}
	t.Fatal("no Message-ID header")
	return ""
}

func TestTheDrainOutlivesASend(t *testing.T) {
	if drainTimeout <= sendTimeout {
		t.Errorf("drainTimeout %v <= sendTimeout %v - a shutdown cannot flush even one in-flight message",
			drainTimeout, sendTimeout)
	}
}
