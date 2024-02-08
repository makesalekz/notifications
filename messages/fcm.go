package messages

import "gitlab.calendaria.team/services/notifications/ent/enum"

type FirebaseNotification struct {
	Type     enum.NotificationType
	UsersIds []int64
	Title    string
	Body     string
	Image    string
	Data     map[string]string
}
