package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func adminHandler(t *testing.T) (*Handler, *database.DB) {
	t.Helper()

	db := testdb.New(t)

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	return New(Deps{
		DB:        db,
		AdminUser: service.NewAdminUserService(db, files),
	}), db
}

func mkAdminUser(t *testing.T, db *database.DB, name, role string) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	user, err := db.CreateUser(ctx, database.CreateUserParams{
		ID:       database.NewID(),
		Username: name,
		Email:    name + "@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating %s: %v", name, err)
	}

	if role == auth.RoleAdmin {
		if _, err := db.ExecContext(ctx, `UPDATE users SET role = 'ADMIN' WHERE id = $1`, user.ID); err != nil {
			t.Fatalf("promoting %s: %v", name, err)
		}
	}

	return user.ID
}

// callAdmin drives one handler directly, standing in for the router: chi's URL
// params and the authenticated caller both live in the request context, so a
// handler test has to put them there itself.
func callAdmin(
	t *testing.T,
	h func(http.ResponseWriter, *http.Request),
	caller uuid.UUID,
	idParam, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", idParam)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = auth.WithUser(ctx, auth.User{ID: caller, Role: auth.RoleAdmin})

	rec := httptest.NewRecorder()
	h(rec, req.WithContext(ctx))
	return rec
}

func TestAnUnknownFieldIsRejectedRatherThanIgnored(t *testing.T) {
	h, db := adminHandler(t)
	admin := mkAdminUser(t, db, "admin", auth.RoleAdmin)
	target := mkAdminUser(t, db, "target", auth.RoleUser)

	// The reason itself is valid, so only DisallowUnknownFields can reject
	// this. A test whose body is invalid anyway would pass either way.
	rec := callAdmin(t, h.SuspendUser, admin, target.String(),
		`{"reason":"listing things they do not have","notify":true}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("an unknown field: got %d, want 400", rec.Code)
	}

	// The danger is a 200 that silently drops the field the caller thought
	// they were setting - here, that nobody is told.
	user, err := db.GetUser(context.Background(), target)
	if err != nil {
		t.Fatalf("re-reading the target: %v", err)
	}
	if user.SuspendedAt.Valid {
		t.Fatal("a rejected body suspended the account anyway")
	}
}

func TestAnUnparseableIDIsA400NotA500(t *testing.T) {
	h, db := adminHandler(t)
	admin := mkAdminUser(t, db, "admin", auth.RoleAdmin)

	rec := callAdmin(t, h.SuspendUser, admin, "not-a-uuid", `{"reason":"spam"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a malformed id: got %d, want 400", rec.Code)
	}
	if msg := decodeError(t, rec); msg != "Invalid user id" {
		t.Fatalf("message: got %q", msg)
	}
}

func TestASuspensionRespondsWithTheNewState(t *testing.T) {
	h, db := adminHandler(t)
	admin := mkAdminUser(t, db, "admin", auth.RoleAdmin)
	target := mkAdminUser(t, db, "target", auth.RoleUser)

	rec := callAdmin(t, h.SuspendUser, admin, target.String(), `{"reason":"spam"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspending: got %d, want 200: %s", rec.Code, rec.Body)
	}

	// The client updates its row from this body rather than refetching, so
	// the derived status and the reason both have to be in it.
	var got struct {
		Email            string `json:"email"`
		Status           string `json:"status"`
		SuspensionReason string `json:"suspension_reason"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding the response: %v", err)
	}

	if got.Status != "suspended" {
		t.Errorf("status: got %q, want \"suspended\"", got.Status)
	}
	if got.SuspensionReason != "spam" {
		t.Errorf("reason: got %q, want \"spam\"", got.SuspensionReason)
	}
	if got.Email != "target@example.test" {
		t.Errorf("email: got %q - admins need it to identify an account", got.Email)
	}
}

func TestAnUnactionedAccountHasAnEmptyHistoryNotNull(t *testing.T) {
	h, db := adminHandler(t)
	admin := mkAdminUser(t, db, "admin", auth.RoleAdmin)
	target := mkAdminUser(t, db, "target", auth.RoleUser)

	rec := callAdmin(t, h.GetUserHistory, admin, target.String(), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("fetching the history: got %d, want 200", rec.Code)
	}

	// A nil slice marshals to null, which a client has to guard before it can
	// iterate. ToUserActionResponses allocates so it never can.
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Fatalf("an empty history: got %s, want []", body)
	}
}

// The response carries the account in its new state, so a console can render
// the row it just changed without refetching.
func TestARoleChangeRespondsWithTheNewState(t *testing.T) {
	h, db := adminHandler(t)
	admin := mkAdminUser(t, db, "admin", auth.RoleAdmin)
	target := mkAdminUser(t, db, "target", auth.RoleUser)

	rec := callAdmin(t, h.SetUserRole, admin, target.String(), `{"role":"ADMIN","note":"trusted"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d with body %s, want 200", rec.Code, rec.Body.String())
	}

	var got dtos.AdminUserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if got.Role != auth.RoleAdmin {
		t.Errorf("role = %q, want the new one", got.Role)
	}
}

// The role is a value the request carries, so a value outside the two is a
// client mistake rather than a server state - 400, not 500.
func TestAnUnknownRoleIsA400(t *testing.T) {
	h, db := adminHandler(t)
	admin := mkAdminUser(t, db, "admin", auth.RoleAdmin)
	target := mkAdminUser(t, db, "target", auth.RoleUser)

	rec := callAdmin(t, h.SetUserRole, admin, target.String(), `{"role":"MODERATOR"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d with body %s, want 400", rec.Code, rec.Body.String())
	}
}

// The client's idea of how many admins exist is a stale page of a paginated
// list, so the refusal has to be the server's and has to arrive as a message
// the console can show.
func TestDemotingTheLastAdminIsA409WithAMessage(t *testing.T) {
	h, db := adminHandler(t)
	admin := mkAdminUser(t, db, "admin", auth.RoleAdmin)
	other := mkAdminUser(t, db, "other", auth.RoleAdmin)

	if rec := callAdmin(t, h.SetUserRole, admin, other.String(), `{"role":"USER"}`); rec.Code != http.StatusOK {
		t.Fatalf("demoting the second admin: %d %s", rec.Code, rec.Body.String())
	}

	rec := callAdmin(t, h.SetUserRole, other, admin.String(), `{"role":"USER"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("got %d with body %s, want 409", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "last active admin") {
		t.Errorf("body = %s, want the reason a console can render", rec.Body.String())
	}
}

func TestChangingYourOwnRoleIsA403(t *testing.T) {
	h, db := adminHandler(t)
	admin := mkAdminUser(t, db, "admin", auth.RoleAdmin)

	rec := callAdmin(t, h.SetUserRole, admin, admin.String(), `{"role":"USER"}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("got %d with body %s, want 403", rec.Code, rec.Body.String())
	}
}
