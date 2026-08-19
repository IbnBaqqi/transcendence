// Package auth issues and verifies the tokens the API authenticates with.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	// "fmt"
	"net/http"
	"strings"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CustomClaims is what the access token carries beyond the registered claims.
type CustomClaims struct {
	Name string `json:"name"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// JwtService signs and verifies access tokens.
type JwtService struct {
	JwtSecret      string
	AccessTokenTTL time.Duration
}

// User is the authenticated caller, as carried in the request context.
type User struct {
	ID   uuid.UUID
	Role string
	Name string
}

type contextKey struct{}

var userKey = contextKey{}

type expiredKey struct{}

var tokenExpiredKey = expiredKey{}

type tokenType string

const tokenTypeAccess tokenType = "access"

// Errors callers match with errors.Is.
var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrExpiredToken         = errors.New("expired token")
	ErrEmptyBearerToken     = errors.New("bearer token is empty")
	ErrInvalidBearerToken   = errors.New("bearer token is incorrect")
	ErrNoAuthHeaderIncluded = errors.New("no auth header included in request")
)

// WithUser puts the authenticated caller in the context.
func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// UserFromContext returns the caller Authenticate attached, if any.
func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userKey).(User)
	return user, ok
}

func WithExpiredToken(ctx context.Context) context.Context {
	return context.WithValue(ctx, tokenExpiredKey, true)
}

func TokenExpired(ctx context.Context) bool {
	expired, _ := ctx.Value(tokenExpiredKey).(bool)
	return expired
}

// NewJwtService builds a service signing with secret.
func NewJwtService(secret string, accessTokenTTL time.Duration) *JwtService {
	return &JwtService{
		JwtSecret:      secret,
		AccessTokenTTL: accessTokenTTL,
	}
}

// IssueAccessToken signs a short-lived token for the user.
func (s *JwtService) IssueAccessToken(user database.User) (string, error) {

	claims := CustomClaims{
		Name: user.Username,
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.AccessTokenTTL)),
			Issuer:    string(tokenTypeAccess),
		},
	}
	signingKey := []byte(s.JwtSecret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtToken, err := token.SignedString(signingKey)
	if err != nil {
		return "", err
	}
	return jwtToken, nil
}

// VerifyAccessToken checks the signature and returns the claims.
func (s *JwtService) VerifyAccessToken(tokenStr string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&CustomClaims{},
		func(token *jwt.Token) (any, error) {
			// Reject anything not signed with HMAC: accepting the token's own
			// choice of algorithm is how signatures get bypassed.
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return []byte(s.JwtSecret), nil
		},
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetBearerToken pulls the token out of an Authorization header.
func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthHeaderIncluded
	}
	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok {
		return "", ErrInvalidBearerToken
	}
	if token == "" {
		return "", ErrEmptyBearerToken
	}
	return token, nil
}

// MakeRefreshToken returns 256 bits of CSPRNG output, hex encoded.
func MakeRefreshToken() string {
	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	return hex.EncodeToString(tokenBytes)
}
