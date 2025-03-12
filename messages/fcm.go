package messages

import "gitlab.calendaria.team/services/notifications/ent/enum"

type FirebaseNotification struct {
	Type     enum.NotificationType `json:"type,omitempty"`
	Badge    *int                  `json:"badge"`
	UsersIds []int64               `json:"users_ids"`
	Title    string                `json:"title,omitempty"`
	Body     string                `json:"body,omitempty"`
	Image    string                `json:"image,omitempty"`
	Data     map[string]string     `json:"data,omitempty"`
}
