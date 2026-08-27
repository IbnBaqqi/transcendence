package notify

import "context"

type Sender interface {
	Send(ctx context.Context, m Message) error
}
