package dtos

import (
	"database/sql"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

type OrderEventResponse struct {
	ID         uuid.UUID `json:"id"`
	OrderID    uuid.UUID `json:"order_id"`
	ActorID    *string   `json:"actor_id"`
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Note       *string   `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

func ToOrderEventResponse(e database.OrderEvent) OrderEventResponse {
	return OrderEventResponse{
		ID:         e.ID,
		OrderID:    e.OrderID,
		ActorID:    actorOrNil(e.ActorID),
		FromStatus: textOrNil(e.FromStatus),
		ToStatus:   e.ToStatus,
		Note:       textOrNil(e.Note),
		CreatedAt:  e.CreatedAt,
	}
}

func ToOrderEventResponses(rows []database.OrderEvent) []OrderEventResponse {
	out := make([]OrderEventResponse, 0, len(rows))
	for _, e := range rows {
		out = append(out, ToOrderEventResponse(e))
	}
	return out
}

func textOrNil(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func actorOrNil(v uuid.NullUUID) *string {
	if !v.Valid {
		return nil
	}
	s := v.UUID.String()
	return &s
}
