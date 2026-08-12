package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

type Service struct {
	db  *database.DB
	jwt *JwtService
}

func NewService(db *database.DB, jwt *JwtService) *Service {
	return &Service{
		db:  db,
		jwt: jwt,
	}
}

// SignupResponse holds the outputs of a successful signup.
type SignupResponse struct {
	AccessToken  string
	RefreshToken string
	User         dtos.UserInfo
}

// LoginResult holds the outputs of a successful login.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	User         dtos.UserInfo
}

// signupFailed logs why signup failed and returns the message the client sees.
func signupFailed(step string, err error) error {
	slog.Error("signup failed", "step", step, "error", err)
	return errors.New("could not create user")
}

// Signup validates input, checks for duplicate email, hashes the
// password, creates the user, and issues tokens.
func (s *Service) Signup(ctx context.Context, input dtos.CreateUserRequest) (SignupResponse, error) {
	if err := validateSignupInput(input); err != nil {
		return SignupResponse{}, err
	}

	// Check for duplicate email
	_, err := s.db.GetUserByEmail(ctx, input.Email)
	if err == nil {
		return SignupResponse{}, &ConflictError{Message: "email already in use"}
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return SignupResponse{}, signupFailed("look up email", err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return SignupResponse{}, signupFailed("hash password", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SignupResponse{}, signupFailed("begin transaction", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("signup transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	user, err := qtx.CreateUser(ctx, database.CreateUserParams{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashed),
	})
	if err != nil {
		return SignupResponse{}, signupFailed("create user", err)
	}

	if err := qtx.EnsureProfile(ctx, user.ID); err != nil {
		return SignupResponse{}, signupFailed("create profile", err)
	}

	if err := tx.Commit(); err != nil {
		return SignupResponse{}, signupFailed("commit", err)
	}

	accessToken, err := s.jwt.IssueAccessToken(user)
	if err != nil {
		slog.Error("signup failed", "step", "issue token", "error", err)
		return SignupResponse{}, errors.New("could not issue token")
	}

	return SignupResponse{
		AccessToken:  accessToken,
		RefreshToken: MakeRefreshToken(),
		User:         toUserInfo(user),
	}, nil
}

// Login verifies credentials and issues tokens.
func (s *Service) Login(ctx context.Context, input dtos.LoginRequest) (LoginResult, error) {
	if input.Email == "" || input.Password == "" {
		return LoginResult{}, &ValidationError{Message: "email and password are required"}
	}

	user, err := s.db.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return LoginResult{}, &AuthError{Message: "invalid email or password"}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return LoginResult{}, &AuthError{Message: "invalid email or password"}
	}

	accessToken, err := s.jwt.IssueAccessToken(user)
	if err != nil {
		slog.Error("login failed", "step", "issue token", "error", err)
		return LoginResult{}, errors.New("could not issue token")
	}

	return LoginResult{
		AccessToken:  accessToken,
		RefreshToken: MakeRefreshToken(),
		User:         toUserInfo(user),
	}, nil
}

func validateSignupInput(input dtos.CreateUserRequest) error {
	if input.Username == "" {
		return &ValidationError{Message: "username is required"}
	}
	if len(input.Username) > 50 {
		return &ValidationError{Message: "username must be 50 characters or fewer"}
	}
	if input.Email == "" {
		return &ValidationError{Message: "email is required"}
	}
	if len(input.Password) < 8 {
		return &ValidationError{Message: "password must be at least 8 characters"}
	}
	return nil
}

func toUserInfo(user database.User) dtos.UserInfo {
	return dtos.UserInfo{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}
}
