package enum

type NotificationType string

const (
	Common  NotificationType = "COMMON"
	Event   NotificationType = "EVENT"
	Contact NotificationType = "CONTACT"
	Tasks   NotificationType = "TASKS"
)

func notificationTypeValues() []NotificationType {
	return []NotificationType{Common, Event, Contact, Tasks}
}

func (NotificationType) Values() (kinds []string) {
	for _, value := range notificationTypeValues() {
		kinds = append(kinds, string(value))
	}
	return
}

func (m NotificationType) Value() string {
	return string(m)
}

func (m NotificationType) IsValid() bool {
	for _, value := range notificationTypeValues() {
		if m == value {
			return true
		}
	}
	return false
}
