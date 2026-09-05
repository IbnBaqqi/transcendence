package dtos

import (
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

type AdminUserQuery struct {
	Role   string
	Status string
	Page   string
	Limit  string
}

type SuspendRequest struct {
	Reason string `json:"reason"`
}

type ReinstateRequest struct {
	Note string `json:"note"`
}

// The request carries the value it wants, so a third role later adds a
// constant rather than a second pair of endpoints.
type SetRoleRequest struct {
	Role string `json:"role"`
	Note string `json:"note"`
}

type AdminDeleteRequest struct {
	Username string `json:"username"`
	Reason   string `json:"reason"`
}

type AdminUserResponse struct {
	ID               uuid.UUID  `json:"id"`
	Username         string     `json:"username"`
	Email            string     `json:"email"`
	Role             string     `json:"role"`
	Status           string     `json:"status"`
	SuspensionReason *string    `json:"suspension_reason,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	LastSeenAt       *time.Time `json:"last_seen_at,omitempty"`
}

type PaginatedAdminUsers struct {
	Items      []AdminUserResponse `json:"items"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

func adminUserStatus(deletedAt, suspendedAt sql.NullTime) string {
	switch {
	case deletedAt.Valid:
		return "deleted"
	case suspendedAt.Valid:
		return "suspended"
	default:
		return "active"
	}
}

func ToAdminUserResponse(u database.User) AdminUserResponse {
	return AdminUserResponse{
		ID:               u.ID,
		Username:         u.Username,
		Email:            u.Email,
		Role:             u.Role,
		Status:           adminUserStatus(u.DeletedAt, u.SuspendedAt),
		SuspensionReason: nullStringPtr(u.SuspensionReason),
		CreatedAt:        u.CreatedAt.Time,
		LastSeenAt:       nullTimePtr(u.LastSeenAt),
	}
}

func ToAdminUserResponses(rows []database.ListUsersForAdminRow) []AdminUserResponse {
	out := make([]AdminUserResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, AdminUserResponse{
			ID:               r.ID,
			Username:         r.Username,
			Email:            r.Email,
			Role:             r.Role,
			Status:           adminUserStatus(r.DeletedAt, r.SuspendedAt),
			SuspensionReason: nullStringPtr(r.SuspensionReason),
			CreatedAt:        r.CreatedAt.Time,
			LastSeenAt:       nullTimePtr(r.LastSeenAt),
		})
	}
	return out
}

type UserActionResponse struct {
	ID          uuid.UUID  `json:"id"`
	Action      string     `json:"action"`
	Note        string     `json:"note"`
	ModeratorID *uuid.UUID `json:"moderator_id"`
	CreatedAt   time.Time  `json:"created_at"`
}

func ToUserActionResponses(rows []database.UserAction) []UserActionResponse {
	out := make([]UserActionResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, UserActionResponse{
			ID:          r.ID,
			Action:      r.Action,
			Note:        r.Note.String,
			ModeratorID: nullUUIDPtr(r.ModeratorID),
			CreatedAt:   r.CreatedAt,
		})
	}
	return out
}
