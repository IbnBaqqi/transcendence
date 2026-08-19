package dtos

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

func sampleEvent() database.OrderEvent {
	return database.OrderEvent{
		ID:         1,
		OrderID:    7,
		ActorID:    uuid.NullUUID{UUID: uuid.MustParse("33333333-3333-3333-3333-333333333333"), Valid: true},
		FromStatus: sql.NullString{String: "confirmed", Valid: true},
		ToStatus:   "completed",
		Note:       sql.NullString{String: "buyer_receipt", Valid: true},
		CreatedAt:  time.Date(2026, 8, 17, 9, 30, 0, 0, time.UTC),
	}
}

func TestOrderEventJSONKeys(t *testing.T) {
	body, err := json.Marshal(ToOrderEventResponse(sampleEvent()))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	for _, key := range []string{
		`"id":`, `"order_id":`, `"actor_id":`, `"from_status":`, `"to_status":`, `"note":`, `"created_at":`,
	} {
		if !strings.Contains(string(body), key) {
			t.Errorf("missing key %s:\n%s", key, body)
		}
	}
	if !strings.Contains(string(body), `"created_at":"2026-08-17T09:30:00Z"`) {
		t.Errorf("created_at is not RFC 3339:\n%s", body)
	}
}

func TestOrderEventNullables(t *testing.T) {
	tests := []struct {
		name  string
		event database.OrderEvent
		want  []string
	}{
		{
			name: "the creation event has no previous status",
			event: database.OrderEvent{
				ID: 1, OrderID: 7, ToStatus: "pending",
				ActorID: uuid.NullUUID{UUID: uuid.New(), Valid: true},
			},
			want: []string{`"from_status":null`, `"note":null`},
		},
		{
			name: "a deleted actor leaves the event behind",
			event: database.OrderEvent{
				ID: 2, OrderID: 7, ToStatus: "cancelled",
				FromStatus: sql.NullString{String: "pending", Valid: true},
			},
			want: []string{`"actor_id":null`, `"to_status":"cancelled"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(ToOrderEventResponse(tt.event))
			if err != nil {
				t.Fatalf("marshalling: %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(string(body), want) {
					t.Errorf("missing %s:\n%s", want, body)
				}
			}
		})
	}
}

func TestOrderEventsIsAlwaysAnArray(t *testing.T) {
	body, err := json.Marshal(ToOrderEventResponses(nil))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	if string(body) != "[]" {
		t.Errorf("empty timelien = %s, want []", body)
	}
}
