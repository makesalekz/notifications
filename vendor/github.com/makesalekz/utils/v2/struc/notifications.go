package struc

import "slices"

// ------------------------------- Structs -----------------------------

type FirebaseNotification struct {
	Type     NotificationType  `json:"type,omitempty"`
	UsersIds []int64           `json:"users_ids"`
	Title    string            `json:"title,omitempty"`
	Body     string            `json:"body,omitempty"`
	Image    string            `json:"image,omitempty"`
	Data     map[string]string `json:"data,omitempty"`
}

type EmailDetails struct {
	Language string            `json:"language,omitempty"`
	Type     string            `json:"type,omitempty"`
	Emails   []string          `json:"emails,omitempty"`
	Data     map[string]string `json:"data,omitempty"`
}

// ------------------------------- Enums -----------------------------

type NotificationType string

const (
	Common   NotificationType = "COMMON"
	Calendar NotificationType = "CALENDAR"
	Event    NotificationType = "EVENT"
	Contact  NotificationType = "CONTACT"
	Tasks    NotificationType = "TASKS"
	Projects NotificationType = "PROJECTS"
	Chat     NotificationType = "CHAT"
)

func notificationTypeValues() []NotificationType {
	return []NotificationType{Common, Calendar, Event, Contact, Tasks, Projects, Chat}
}

func (NotificationType) Values() (kinds []string) {
	for _, value := range notificationTypeValues() {
		kinds = append(kinds, string(value))
	}
	return
}

func (t NotificationType) Value() string {
	return string(t)
}

func (t NotificationType) IsValid() bool {
	return slices.Contains(notificationTypeValues(), t)
}
