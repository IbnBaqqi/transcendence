package dtos

import "github.com/IbnBaqqi/transcendence/internal/database"

type UpdateSettingsInput struct {
	ShowOnlineStatus *bool `json:"show_online_status"`
}

type UserSettingsResponse struct {
	ShowOnlineStatus bool `json:"show_online_status"`
}

func ToUserSettingsResponse(u database.User) UserSettingsResponse {
	return UserSettingsResponse{ShowOnlineStatus: u.ShowOnlineStatus}
}
