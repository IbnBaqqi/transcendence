// Package notify sends transactional email: welcome, order and chat events,
// and the password-reset link.
//
// # Delivery is at-most-once, deliberately
//
// Notify hands a message to a background dispatcher and returns immediately.
// The send is not part of the caller's database transaction, and nothing is
// retried or persisted, so a message can be lost: the queue is full, the
// process stops, or the relay is down. That is the trade. Making it
// at-least-once means an outbox table and a worker, which is a lot of
// machinery for "you have a new message" - and the alternative, sending inside
// the transaction, is worse: the mail goes out and the transaction then rolls
// back, telling someone about an order that does not exist.
//
// The consequence for callers: never make a notification a precondition for
// anything. A failed send is logged, never returned, and never rolls back the
// work that triggered it.
//
// # Configuration
//
// An empty MAIL_USERNAME means an unauthenticated relay - that is what Mailpit
// wants locally, and it is why the username is checked rather than assumed. A
// real relay needs both username and password.
//
// Disabled{} is the no-op notifier used when mail is switched off, so callers
// never nil-check.
package notify

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/IbnBaqqi/transcendence/internal/config"
)

func New(cfg config.MailConfig) Notifier {
	if !cfg.Configured() {
		slog.Info("mail is not configured, notifications will be logged only")
		return Disabled{}
	}
	return NewDispatcher(NewSMTP(cfg))
}

type Kind string

const (
	KindWelcome         Kind = "welcome"
	KindOrderPlaced     Kind = "order_placed"
	KindOrderHandedOver Kind = "order_handed_over"
	KindOrderCancelled  Kind = "order_cancelled"
	KindChatRequest     Kind = "chat_request"
)

type Message struct {
	Kind    Kind
	To      string
	Subject string
	Body    string
}

type Notifier interface {
	Notify(ctx context.Context, m Message)
	Close()
}

type Disabled struct{}

func (Disabled) Close() {}

func (Disabled) Notify(_ context.Context, m Message) {
	slog.Debug("notification not sent, mail is not configured",
		"kind", m.Kind, "to", m.To)
}

func Welcome(to, username string) Message {
	return Message{
		Kind:    KindWelcome,
		To:      to,
		Subject: "Welcome to the forager's market",
		Body: fmt.Sprintf(
			"Hei %s,\n\nYour account is ready. Happy foraging.\n", username),
	}
}

func OrderPlaced(to, listingTitle string, quantity int32, unit string) Message {
	return Message{
		Kind:    KindOrderPlaced,
		To:      to,
		Subject: "You have a new order",
		Body: fmt.Sprintf(
			"Someone ordered %d %s of %s.\n\nConfirm or cancel it from your orders page.\n",
			quantity, unit, listingTitle),
	}
}

func OrderHandedOver(to, listingTitle string) Message {
	return Message{
		Kind:    KindOrderHandedOver,
		To:      to,
		Subject: "Your order is on its way",
		Body: fmt.Sprintf(
			"The seller has handed over your order of %s.\n", listingTitle),
	}
}

func OrderCancelled(to, listingTitle string) Message {
	return Message{
		Kind:    KindOrderCancelled,
		To:      to,
		Subject: "An order was cancelled",
		// Neutral wording: this goes to whichever side did not cancel, so it
		// cannot talk about released stock (seller-facing) or a refund (there
		// is no real money in this flow).
		Body: fmt.Sprintf(
			"The order for %s has been cancelled.\n",
			listingTitle),
	}
}

func ChatRequest(to, listingTitle string) Message {
	return Message{
		Kind:    KindChatRequest,
		To:      to,
		Subject: "Someone is asking about your listing",
		Body: fmt.Sprintf(
			"You have a new chat request about %s. Accept it to reply.\n",
			listingTitle),
	}
}
