package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func newFollowService(t *testing.T) (*FollowService, *database.DB, uuid.UUID, uuid.UUID) {
	t.Helper()

	db := testdb.New(t)

	mk := func(name string) uuid.UUID {
		user, err := db.CreateUser(context.Background(), database.CreateUserParams{
			Username: name,
			Email:    name + "@example.test",
			Password: "irrelevant",
		})
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		return user.ID
	}

	return NewFollowService(db.Queries), db, mk("aino"), mk("bea")
}

func TestFollowIsIdempotent(t *testing.T) {
	svc, _, aino, bea := newFollowService(t)
	ctx := context.Background()

	if err := svc.Follow(ctx, aino, bea); err != nil {
		t.Fatalf("first follow: %v", err)
	}
	if err := svc.Follow(ctx, aino, bea); err != nil {
		t.Fatalf("second follow: %v", err)
	}

	following, err := svc.ListFollowing(ctx, aino)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(following) != 1 {
		t.Errorf("following = %d rows, want 1 - the second follow duplicated it", len(following))
	}
}

func TestFollowRejectsYourself(t *testing.T) {
	svc, _, aino, _ := newFollowService(t)

	err := svc.Follow(context.Background(), aino, aino)

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("err = %v, want *ValidationError", err)
	}
}

func TestFollowAnUnknownUserIsNotFound(t *testing.T) {
	svc, _, aino, _ := newFollowService(t)

	err := svc.Follow(context.Background(), aino, uuid.New())

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %#v, want *NotFoundError (a 500 here means the FK violation is unhandled)", err)
	}
}

func TestUnfollowWhenNotFollowing(t *testing.T) {
	svc, _, aino, bea := newFollowService(t)
	ctx := context.Background()

	var notFound *NotFoundError
	if err := svc.Unfollow(ctx, aino, bea); !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *NotFoundError", err)
	}

	if err := svc.Follow(ctx, aino, bea); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := svc.Unfollow(ctx, aino, bea); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if err := svc.Unfollow(ctx, aino, bea); !errors.As(err, &notFound) {
		t.Errorf("unfollowing twice = %v, want *NotFoundError", err)
	}
}

func TestFollowIsOneDirectional(t *testing.T) {
	svc, _, aino, bea := newFollowService(t)
	ctx := context.Background()

	if err := svc.Follow(ctx, aino, bea); err != nil {
		t.Fatalf("follow: %v", err)
	}

	beaFollowing, err := svc.ListFollowing(ctx, bea)
	if err != nil {
		t.Fatalf("listing bea's following: %v", err)
	}
	if len(beaFollowing) != 0 {
		t.Errorf("bea follows %d, want 0 - the edge went both ways", len(beaFollowing))
	}

	ainoFollowers, err := svc.ListFollowers(ctx, aino)
	if err != nil {
		t.Fatalf("listing aino's followers: %v", err)
	}
	if len(ainoFollowers) != 0 {
		t.Errorf("aino has %d followers, want 0", len(ainoFollowers))
	}

	beaFollowers, err := svc.ListFollowers(ctx, bea)
	if err != nil {
		t.Fatalf("listing bea's followers: %v", err)
	}
	if len(beaFollowers) != 1 || beaFollowers[0].Username != "aino" {
		t.Errorf("bea's followers = %+v, want [aino]", beaFollowers)
	}
}

func TestListFollowersOfAnUnknownUser(t *testing.T) {
	svc, _, _, _ := newFollowService(t)

	_, err := svc.ListFollowers(context.Background(), uuid.New())

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *NotFoundError", err)
	}
}

func TestDeletingAUserRemovesTheirFollows(t *testing.T) {
	svc, db, aino, bea := newFollowService(t)
	ctx := context.Background()

	if err := svc.Follow(ctx, aino, bea); err != nil {
		t.Fatalf("follow: %v", err)
	}

	if _, err := db.Exec("DELETE FROM users WHERE id = $1", bea); err != nil {
		t.Fatalf("deleting bea: %v", err)
	}

	following, err := svc.ListFollowing(ctx, aino)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(following) != 0 {
		t.Errorf("following = %d rows, want 0 - the follow outlived the user", len(following))
	}
}
