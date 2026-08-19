// Package auth provides JWT authentication and authorization.
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

type CustomClaims struct {
	Name string `json:"name"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type JwtService struct {
	JwtSecret      string
	AccessTokenTTL time.Duration
}

type User struct {
	ID   uuid.UUID
	Role string
	Name string
}

const (
	RoleUser  = "USER"
	RoleAdmin = "ADMIN"
)

type contextKey struct{}

var userKey = contextKey{}

type tokenType string

const tokenTypeAccess tokenType = "access"

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrExpiredToken         = errors.New("expired token")
	ErrEmptyBearerToken     = errors.New("bearer token is empty")
	ErrInvalidBearerToken   = errors.New("bearer token is incorrect")
	ErrNoAuthHeaderIncluded = errors.New("no auth header included in request")
)

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userKey).(User)
	return user, ok
}

func NewJwtService(secret string) *JwtService {
	return &JwtService{
		JwtSecret:      secret,
		AccessTokenTTL: 7 * 24 * time.Hour,
	}
}

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

func (s *JwtService) VerifyAccessToken(tokenStr string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&CustomClaims{},
		func(token *jwt.Token) (any, error) {
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

func MakeRefreshToken() string {
	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	return hex.EncodeToString(tokenBytes)
}
