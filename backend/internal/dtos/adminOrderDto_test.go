package dtos

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

func TestAdminOrderFlattensTheOrderItWraps(t *testing.T) {
	got, err := json.Marshal(ToAdminOrderResponse(database.Order{Status: "confirmed"}, true))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	body := string(got)

	if strings.Contains(body, "OrderResponse") {
		t.Fatalf("the order is nested instead of flattened:\n%s", body)
	}
	if !strings.Contains(body, `"status":"confirmed"`) {
		t.Errorf("the wrapped order's fields are missing:\n%s", body)
	}
	if !strings.Contains(body, `"stuck":true`) {
		t.Errorf("stuck is missing:\n%s", body)
	}
}
