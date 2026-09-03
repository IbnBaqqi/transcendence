package dtos

import (
	"database/sql"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

func ToFollowingResponses(rows []database.ListFollowingRow) []ChatUserResponse {
	out := make([]ChatUserResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, chatUser(r.ID, r.Username, r.AvatarFilename, r.LastSeenAt, r.ShowOnlineStatus))
	}
	return out
}

func ToFollowerResponses(rows []database.ListFollowersRow) []ChatUserResponse {
	out := make([]ChatUserResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, chatUser(r.ID, r.Username, r.AvatarFilename, r.LastSeenAt, r.ShowOnlineStatus))
	}
	return out
}

// chatUser is where the actual work happens, including applying the
// show_online_status rule via toPresence.
func chatUser(
	id uuid.UUID,
	username string,
	avatarFilename sql.NullString,
	lastSeen sql.NullTime,
	showOnlineStatus bool,
) ChatUserResponse {
	return ChatUserResponse{
		ID:        id,
		Username:  username,
		AvatarURL: avatarURL(avatarFilename),
		Presence:  toPresence(lastSeen, showOnlineStatus),
	}
}
