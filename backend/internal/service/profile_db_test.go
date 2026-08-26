package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// newProfileService returns the service plus a user that already has the empty
// profile row every account gets at signup.
func newProfileService(t *testing.T) (*ProfileService, uuid.UUID) {
	t.Helper()

	db := testdb.New(t)

	user, err := db.CreateUser(context.Background(), database.CreateUserParams{
		ID:       database.NewID(),
		Username: "aino",
		Email:    "aino@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating a user: %v", err)
	}
	if err := db.EnsureProfile(context.Background(), user.ID); err != nil {
		t.Fatalf("creating the profile: %v", err)
	}

	return NewProfileService(db), user.ID
}

func TestProfileUpdateKeepsFieldsThePatchOmits(t *testing.T) {
	svc, userID := newProfileService(t)
	ctx := context.Background()

	if _, err := svc.Update(ctx, userID, dtos.UpdateProfileInput{
		Firstname:   dtos.SetString("Aino"),
		Bio:         dtos.SetString("picks chanterelles"),
		DateOfBirth: dtos.SetString("2001-05-14"),
		Location:    dtos.SetString("Espoo"),
	}); err != nil {
		t.Fatalf("first update: %v", err)
	}

	got, err := svc.Update(ctx, userID, dtos.UpdateProfileInput{Lastname: dtos.SetString("Virtanen")})
	if err != nil {
		t.Fatalf("second update: %v", err)
	}

	if got.Profile.Firstname.String != "Aino" {
		t.Errorf("firstname = %q, want %q", got.Profile.Firstname.String, "Aino")
	}
	if got.Profile.Bio.String != "picks chanterelles" {
		t.Errorf("bio = %q, want it unchanged", got.Profile.Bio.String)
	}
	if !got.Profile.DateOfBirth.Valid {
		t.Error("date_of_birth was cleared by a patch that did not mention it")
	}
	if got.Location.String != "Espoo" {
		t.Errorf("location = %q, want it unchanged", got.Location.String)
	}
}

func TestProfileUpdateClearsWithAnEmptyString(t *testing.T) {
	svc, userID := newProfileService(t)
	ctx := context.Background()

	if _, err := svc.Update(ctx, userID, dtos.UpdateProfileInput{Bio: dtos.SetString("picks chanterelles")}); err != nil {
		t.Fatalf("setting the bio: %v", err)
	}

	got, err := svc.Update(ctx, userID, dtos.UpdateProfileInput{Bio: dtos.SetString("")})
	if err != nil {
		t.Fatalf("clearing the bio: %v", err)
	}

	if got.Profile.Bio.Valid {
		t.Errorf("bio = %q, want NULL", got.Profile.Bio.String)
	}
}

// location lives in addresses, which has no row until the first write.
func TestProfileUpdateWritesLocationToAddresses(t *testing.T) {
	svc, userID := newProfileService(t)
	ctx := context.Background()

	before, err := svc.Get(ctx, userID)
	if err != nil {
		t.Fatalf("reading a profile with no address: %v", err)
	}
	if before.Location.Valid {
		t.Errorf("location = %q, want none before anything is written", before.Location.String)
	}

	if _, err := svc.Update(ctx, userID, dtos.UpdateProfileInput{Location: dtos.SetString("Espoo")}); err != nil {
		t.Fatalf("inserting the address: %v", err)
	}

	got, err := svc.Update(ctx, userID, dtos.UpdateProfileInput{Location: dtos.SetString("Tampere")})
	if err != nil {
		t.Fatalf("updating the address: %v", err)
	}
	if got.Location.String != "Tampere" {
		t.Errorf("location = %q, want %q", got.Location.String, "Tampere")
	}

	reread, err := svc.Get(ctx, userID)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}
	if reread.Location.String != "Tampere" {
		t.Errorf("location did not survive the read: %q", reread.Location.String)
	}
}

func TestProfileGetUnknownUser(t *testing.T) {
	svc, _ := newProfileService(t)

	_, err := svc.Get(context.Background(), uuid.New())

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *NotFoundError", err)
	}
}
