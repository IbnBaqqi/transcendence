package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func newFollowService(t *testing.T) (*FollowService, *database.DB, uuid.UUID, uuid.UUID) {
	t.Helper()

	db := testdb.New(t)

	mk := func(name string) uuid.UUID {
		user, err := db.CreateUser(context.Background(), database.CreateUserParams{
			ID:       database.NewID(),
			Username: name,
			Email:    name + "@example.test",
			Password: sql.NullString{String: "irrelevant", Valid: true},
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

func TestFollowAnUnknownUserIsNotFound(t *testing.T) {
	svc, _, aino, _ := newFollowService(t)

	err := svc.Follow(context.Background(), aino, uuid.New())

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %#v, want *NotFoundError (a 500 here means the FK violation is unhandled)", err)
	}
}

func TestFollowWithADeletedFollowerIsNotFound(t *testing.T) {
	svc, db, aino, bea := newFollowService(t)
	ctx := context.Background()

	if _, err := db.Exec("DELETE FROM users WHERE id = $1", aino); err != nil {
		t.Fatalf("deleting aino: %v", err)
	}

	err := svc.Follow(ctx, aino, bea)

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %#v, want *NotFoundError (a 500 here means the follower FK is unhandled)", err)
	}
}

func TestSelfFollowIsImpossibleInTheDatabase(t *testing.T) {
	_, db, aino, _ := newFollowService(t)

	_, err := db.Exec("INSERT INTO follows (follower_id, followee_id) VALUES ($1, $1)", aino)
	if err == nil {
		t.Fatal("the database accepted a self-follow - is follows_no_self_follow still there?")
	}

	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		t.Fatalf("err = %v, want a *pq.Error", err)
	}
	if pqErr.Code != "23514" || pqErr.Constraint != "follows_no_self_follow" {
		t.Errorf("code = %s constraint = %q, want 23514 follows_no_self_follow", pqErr.Code, pqErr.Constraint)
	}
}

func TestUnfollowIsIdempotent(t *testing.T) {
	svc, _, aino, bea := newFollowService(t)
	ctx := context.Background()

	if err := svc.Unfollow(ctx, aino, bea); err != nil {
		t.Fatalf("unfollowing without following = %v, want nil", err)
	}

	if err := svc.Follow(ctx, aino, bea); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := svc.Unfollow(ctx, aino, bea); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if err := svc.Unfollow(ctx, aino, bea); err != nil {
		t.Errorf("unfollowing twice = %v, want nil", err)
	}

	following, err := svc.ListFollowing(ctx, aino)
	if err != nil {
		t.Fatal(err)
	}
	if len(following) != 0 {
		t.Errorf("aino follows %+v, want nobody", following)
	}
}

func TestUnfollowAnUnknownUser(t *testing.T) {
	svc, _, aino, _ := newFollowService(t)

	if err := svc.Unfollow(context.Background(), aino, uuid.New()); err != nil {
		t.Errorf("err = %v, want nil", err)
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

func TestListFollowingOfAnUnknownUser(t *testing.T) {
	svc, _, _, _ := newFollowService(t)

	_, err := svc.ListFollowing(context.Background(), uuid.New())

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *NotFoundError", err)
	}
}

func TestListFollowingOfADeletedCaller(t *testing.T) {
	svc, db, aino, _ := newFollowService(t)

	if _, err := db.Exec(`DELETE FROM users WHERE id = $1`, aino); err != nil {
		t.Fatal(err)
	}

	_, err := svc.ListFollowing(context.Background(), aino)

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *NotFoundError", err)
	}
}

func TestDeletingAUserRemovesTheirFollows(t *testing.T) {
	tests := []struct {
		name   string
		delete string
	}{
		{"the followee is deleted", "followee"},
		{"the follower is deleted", "follower"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, db, aino, bea := newFollowService(t)
			ctx := context.Background()

			if err := svc.Follow(ctx, aino, bea); err != nil {
				t.Fatalf("follow: %v", err)
			}

			gone := bea
			if tt.delete == "follower" {
				gone = aino
			}
			if _, err := db.Exec("DELETE FROM users WHERE id = $1", gone); err != nil {
				t.Fatalf("deleting the user: %v", err)
			}

			var rows int
			if err := db.QueryRow("SELECT count(*) FROM follows").Scan(&rows); err != nil {
				t.Fatalf("counting: %v", err)
			}
			if rows != 0 {
				t.Errorf("follows = %d, want 0 - the edge outlived the user", rows)
			}
		})
	}
}
