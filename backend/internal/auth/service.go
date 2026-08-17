package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"unicode/utf8"

	"github.com/lib/pq"
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

// signupFailed labels the cause with the step that failed. It deliberately does
// NOT log: the handler logs every 5xx exactly once with the request id attached,
// and a second line here would put the detail and the id on different lines.
func signupFailed(step string, err error) error {
	return fmt.Errorf("signup: %s: %w", step, err)
}

// Signup validates input, hashes the password, creates the user (letting the
// unique constraints on email and username decide), and issues tokens.
func (s *Service) Signup(ctx context.Context, input dtos.CreateUserRequest) (SignupResponse, error) {
	if err := validateSignupInput(input); err != nil {
		return SignupResponse{}, err
	}

	// Fast path only: neither of these is the guarantee - they can both race
	// with a concurrent signup, so the unique constraints below stay the
	// authority. This just stops a doomed signup paying for a bcrypt hash.
	taken, err := s.db.UserCredentialsTaken(ctx, database.UserCredentialsTakenParams{
		Email:    input.Email,
		Username: input.Username,
	})
	if err != nil {
		return SignupResponse{}, signupFailed("look up credentials", err)
	}
	if taken.EmailTaken {
		return SignupResponse{}, &ConflictError{Message: emailTakenMessage}
	}
	if taken.UsernameTaken {
		return SignupResponse{}, &ConflictError{Message: usernameTakenMessage}
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
		if conflict := duplicateUserError(err); conflict != nil {
			return SignupResponse{}, conflict
		}
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
		return SignupResponse{}, fmt.Errorf("signup: issue token: %w", err)
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
		return LoginResult{}, fmt.Errorf("login: issue token: %w", err)
	}

	return LoginResult{
		AccessToken:  accessToken,
		RefreshToken: MakeRefreshToken(),
		User:         toUserInfo(user),
	}, nil
}

const (
	maxUsernameLength = 50
	// The users.email column is varchar(150); without this a long address is
	// a 22001 from Postgres, which classifies as a 500.
	maxEmailLength    = 150
	minPasswordLength = 8

	// bcrypt refuses anything longer and returns ErrPasswordTooLong. Rejecting
	// it here turns what would be a 500 from the hashing step into a 400.
	maxPasswordLength = 72
)

// Both the pre-check and the constraint path report a duplicate, so they share
// one message rather than drifting apart.
const (
	emailTakenMessage    = "email already in use"
	usernameTakenMessage = "username already taken"
)

func validateSignupInput(input dtos.CreateUserRequest) error {
	if input.Username == "" {
		return &ValidationError{Message: "username is required"}
	}
	if utf8.RuneCountInString(input.Username) > maxUsernameLength {
		return &ValidationError{Message: tooLong("username", maxUsernameLength)}
	}
	if input.Email == "" {
		return &ValidationError{Message: "email is required"}
	}
	if utf8.RuneCountInString(input.Email) > maxEmailLength {
		return &ValidationError{Message: tooLong("email", maxEmailLength)}
	}
	if len(input.Password) < minPasswordLength {
		return &ValidationError{
			Message: fmt.Sprintf("password must be at least %d bytes", minPasswordLength),
		}
	}
	if len(input.Password) > maxPasswordLength {
		return &ValidationError{Message: passwordTooLong(maxPasswordLength)}
	}
	return nil
}

// tooLong keeps each limit and the message describing it in one place.
func tooLong(field string, limit int) string {
	return fmt.Sprintf("%s must be %d characters or fewer", field, limit)
}

func passwordTooLong(limit int) string {
	return fmt.Sprintf("password must be %d bytes or fewer", limit)
}

// The unique indexes from migration 001. Postgres reports the constraint name
// with the error, which is what lets us tell the two apart.
const (
	usernameConstraint = "users_username_uq"
	emailConstraint    = "users_email_uq"
)

// duplicateUserError translates a unique violation into the error the handler
// maps to 409. Returns nil when err is something else.
func duplicateUserError(err error) error {
	switch {
	case isUniqueViolation(err, usernameConstraint):
		return &ConflictError{Message: usernameTakenMessage}
	case isUniqueViolation(err, emailConstraint):
		return &ConflictError{Message: emailTakenMessage}
	}
	return nil
}

// isUniqueViolation reports whether err is Postgres' "duplicate key" (23505)
// for one specific constraint.
func isUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	return pqErr.Code == "23505" && pqErr.Constraint == constraint
}

func toUserInfo(user database.User) dtos.UserInfo {
	return dtos.UserInfo{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}
}
