package service

import (
	"database/sql"
	"testing"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

func TestCheckOrderActor(t *testing.T) {
	seller := uuid.New()
	buyer := uuid.New()
	stranger := uuid.New()

	order := database.Order{SellerID: seller, BuyerID: buyer}

	tests := []struct {
		name    string
		userID  uuid.UUID
		action  orderAction
		wantErr bool
	}{
		{"seller may confirm", seller, actionConfirm, false},
		{"buyer may not confirm", buyer, actionConfirm, true},
		{"stranger may not confirm", stranger, actionConfirm, true},
		{"either side may cancel", buyer, actionCancel, false},
		{"stranger may not cancel", stranger, actionCancel, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := checkOrderActor(order, tt.userID, tt.action)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkOrderAction() error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBothSidesMarked(t *testing.T) {
	now := sql.NullTime{Time: time.Now(), Valid: true}
	null := sql.NullTime{}

	if bothSidesMarked(database.Order{SellerHandedOverAt: now, BuyerReceivedAt: null}) {
		t.Error("seller only: want false")
	}
	if bothSidesMarked(database.Order{SellerHandedOverAt: null, BuyerReceivedAt: now}) {
		t.Error("buyer only: want false")
	}
	if !bothSidesMarked(database.Order{SellerHandedOverAt: now, BuyerReceivedAt: now}) {
		t.Error("both marked: want true")
	}
}

func TestCheckHandshakeLock(t *testing.T) {
	marked := sql.NullTime{Time: time.Now(), Valid: true}

	tests := []struct {
		name string
		order database.Order
		action orderAction
		wantErr bool
	}{
		{"cancel allowed when nothing marked", database.Order{}, actionCancel, false},
		{"cancel blocked after seller marked", database.Order{SellerHandedOverAt: marked}, actionCancel, true},
		{"cancel blocked after buyer marked", database.Order{BuyerReceivedAt: marked}, actionCancel, true},
		{"confirm unaffected by marks", database.Order{SellerHandedOverAt: marked}, actionConfirm, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := checkHandshakeLock(tt.order, tt.action)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkHandshakeLock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}