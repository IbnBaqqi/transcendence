package dtos

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

func TestNewOrderResponseJSON(t *testing.T) {
	at := sql.NullTime{Time: time.Unix(0, 0).UTC(), Valid: true}
	row := database.Order{
		ID:           7,
		ListingID:    1,
		ListingTitle: "Golden Chanterelles",
		BuyerID:      uuid.MustParse("3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34"),
		SellerID:     uuid.MustParse("9c2b1d40-5e6f-4a7b-8c9d-0e1f2a3b4c5d"),
		Quantity:     4,
		UnitPrice:    "18.50",
		TotalPrice:   "74.00",
		Status:       "pending",
		CreatedAt:    at,
		UpdatedAt:    at,
	}

	b, err := json.Marshal(NewOrderResponse(row))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	got := string(b)

	want := `{"id":7,"listing_id":1,"listing_title":"Golden Chanterelles",` +
		`"buyer_id":"3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",` +
		`"seller_id":"9c2b1d40-5e6f-4a7b-8c9d-0e1f2a3b4c5d",` +
		`"quantity":4,"unit_price":18.5,"total_price":74,"status":"pending",` +
		`"seller_handed_over_at":null,"buyer_received_at":null,` +
		`"created_at":"1970-01-01T00:00:00Z","updated_at":"1970-01-01T00:00:00Z"}`

	if got != want {
		t.Errorf("JSON shape changed\n got: %s\nwant: %s", got, want)
	}
}

func TestAnUnparseableNumericIsLogged(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(previous)

	if got := numericToFloat("not a number"); got != 0 {
		t.Errorf("value = %v, want 0", got)
	}
	if !strings.Contains(buf.String(), "could not parse a NUMERIC value") {
		t.Errorf("nothing was logged: %s", buf.String())
	}

	buf.Reset()
	if got := numericToFloat("18.50"); got != 18.5 {
		t.Errorf("value = %v, want 18.5", got)
	}
	if buf.Len() != 0 {
		t.Errorf("a good value logged something: %s", buf.String())
	}
}

func TestOrderPricesAreUnquotedNumbers(t *testing.T) {
	tests := []struct {
		stored string
		want   string
	}{
		{"18.50", "18.5"},
		{"0.10", "0.1"},
		{"74.00", "74"},
		{"99999999.99", "99999999.99"},
		{"0.01", "0.01"},
	}

	for _, tt := range tests {
		t.Run(tt.stored, func(t *testing.T) {
			b, err := json.Marshal(NewOrderResponse(database.Order{
				UnitPrice:  tt.stored,
				TotalPrice: tt.stored,
			}))
			if err != nil {
				t.Fatal(err)
			}
			got := string(b)

			if want := `"unit_price":` + tt.want + `,`; !strings.Contains(got, want) {
				t.Errorf("got %s, want it to contain %s", got, want)
			}
			if strings.Contains(got, `"unit_price":"`) {
				t.Errorf("unit_price is quoted: %s", got)
			}
		})
	}
}
