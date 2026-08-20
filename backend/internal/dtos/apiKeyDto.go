package dtos

import (
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

type CreateAPIKeyInput struct {
	Name string `json:"name"`
}

type APIKeyResponse struct {
	ID         int32      `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreatedAPIKeyResponse struct {
	APIKeyResponse
	Key string `json:"key"`
}

func ToAPIKeyResponse(row database.ListKeysForUserRow) APIKeyResponse {
	return APIKeyResponse{
		ID:         row.ID,
		Name:       row.Name,
		KeyPrefix:  row.KeyPrefix,
		LastUsedAt: nullTimeToPtr(row.LastUsedAt),
		RevokedAt:  nullTimeToPtr(row.RevokedAt),
		CreatedAt:  row.CreatedAt,
	}
}

func ToAPIKeyResponses(rows []database.ListKeysForUserRow) []APIKeyResponse {
	out := make([]APIKeyResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ToAPIKeyResponse(r))
	}
	return out
}

func NewCreatedAPIKeyResponse(record database.ApiKey, raw string) CreatedAPIKeyResponse {
	return CreatedAPIKeyResponse{
		APIKeyResponse: APIKeyResponse{
			ID:         record.ID,
			Name:       record.Name,
			KeyPrefix:  record.KeyPrefix,
			LastUsedAt: nullTimeToPtr(record.LastUsedAt),
			RevokedAt:  nullTimeToPtr(record.RevokedAt),
			CreatedAt:  record.CreatedAt,
		},
		Key: raw,
	}
}
