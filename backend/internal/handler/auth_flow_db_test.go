package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func authRouter(t *testing.T) http.Handler {
	t.Helper()

	db := testdb.New(t)
	jwt := auth.NewJwtService("test-secret", time.Hour)
	h := New(db, auth.NewService(db, jwt, notify.Disabled{}, "http://frontend.test"), nil, nil, nil, nil,
		service.NewUserService(db.Queries), nil, nil, nil, nil, nil, 0, true, nil, "")

	r := chi.NewRouter()
	r.Use(mw.Authenticate(jwt, nil))
	r.Post("/auth/signup", h.Signup)
	r.Post("/auth/refresh", h.Refresh)
	r.Post("/auth/logout", h.Logout)
	r.Group(func(r chi.Router) {
		r.Use(mw.RequiredAuth)
		r.Get("/auth/me", h.Me)
	})

	return r
}

type reply struct {
	code    int
	body    string
	cookie  *http.Cookie
	headers http.Header
}

func send(t *testing.T, r http.Handler, method, path, body string, cookie *http.Cookie, bearer string) reply {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	out := reply{code: rec.Code, body: rec.Body.String(), headers: rec.Header()}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "refresh_token" {
			out.cookie = c
		}
	}
	return out
}

func signupVia(t *testing.T, r http.Handler) (accessToken string, cookie *http.Cookie) {
	t.Helper()

	res := send(t, r, http.MethodPost, "/auth/signup",
		`{"username":"aino","email":"aino@example.test","password":"password123"}`, nil, "")
	if res.code != http.StatusCreated {
		t.Fatalf("signup = %d: %s", res.code, res.body)
	}
	if res.cookie == nil {
		t.Fatal("signup set no refresh cookie")
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(res.body), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.AccessToken, res.cookie
}

func TestRefreshRotatesTheCookieAndReturnsTheUser(t *testing.T) {
	r := authRouter(t)
	_, cookie := signupVia(t, r)

	res := send(t, r, http.MethodPost, "/auth/refresh", "", cookie, "")
	if res.code != http.StatusOK {
		t.Fatalf("refresh = %d: %s", res.code, res.body)
	}
	if res.cookie == nil || res.cookie.Value == cookie.Value {
		t.Error("the refresh cookie was not rotated")
	}
	if !res.cookie.HttpOnly {
		t.Error("the rotated cookie is not HttpOnly")
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		User        struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal([]byte(res.body), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.AccessToken == "" || payload.User.Username != "aino" {
		t.Errorf("body = %s, want an access token and the user", res.body)
	}
}

func TestRefreshWithoutACookieIs401(t *testing.T) {
	r := authRouter(t)

	res := send(t, r, http.MethodPost, "/auth/refresh", "", nil, "")
	if res.code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401: %s", res.code, res.body)
	}
}

func TestLogoutMakesTheCookieUnusable(t *testing.T) {
	r := authRouter(t)
	_, cookie := signupVia(t, r)

	res := send(t, r, http.MethodPost, "/auth/logout", "", cookie, "")
	if res.code != http.StatusNoContent {
		t.Fatalf("logout = %d", res.code)
	}
	if res.cookie == nil || res.cookie.MaxAge != -1 {
		t.Error("logout did not clear the cookie")
	}

	after := send(t, r, http.MethodPost, "/auth/refresh", "", cookie, "")
	if after.code != http.StatusUnauthorized {
		t.Errorf("refresh after logout = %d, want 401: %s", after.code, after.body)
	}
}

func TestMeReturnsTheSessionsIdentity(t *testing.T) {
	r := authRouter(t)
	accessToken, _ := signupVia(t, r)

	res := send(t, r, http.MethodGet, "/auth/me", "", nil, accessToken)
	if res.code != http.StatusOK {
		t.Fatalf("me = %d: %s", res.code, res.body)
	}

	for _, want := range []string{`"username":"aino"`, `"email":"aino@example.test"`, `"role":"USER"`, `"id":"`} {
		if !strings.Contains(res.body, want) {
			t.Errorf("body is missing %s:\n%s", want, res.body)
		}
	}
}

func TestMeWithoutATokenIs401(t *testing.T) {
	r := authRouter(t)

	res := send(t, r, http.MethodGet, "/auth/me", "", nil, "")
	if res.code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", res.code)
	}
}
