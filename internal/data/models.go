package data

import (
	"encoding/json"
	"fmt"
	"strconv"

	contacts_v1 "gitlab.calendaria.team/services/contacts/api/contacts/v1"
	events_v1 "gitlab.calendaria.team/services/events/api/events/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	projects_v1 "gitlab.calendaria.team/services/pms/projects/api/projects/v1"
	tasks_v1 "gitlab.calendaria.team/services/pms/tasks/api/tasks/v1"
)

type FilterNotificationsDto struct {
	UserId int64
	Type   string
}

type ReadNotificationDto struct {
	UserId         int64
	Type           string
	NotificationId int64
}

type Counter struct {
	UserId int    `json:"user_id"`
	Type   string `json:"type"`
	Count  int    `json:"count"`
}

type NotificationDto struct {
	UserId    int64
	Title     string
	Text      string
	EventId   int64
	ContactId int64
	TaskId    int64
	ProjectId int64

	NotificationAddInfo
}

type NotificationAddInfo struct {
	Type         *string
	TaskJson     *string
	ProjectJson  *string
	ContactJson  *string
	EventJson    *string
	MemberJson   *string
	ChatJson     *string
	MessageJson  *string
	UserJson     *string
	MetadataJson *string
	PluralCount  *int64

	convertedMap map[string]interface{}
}

func FromEnt(n_ent *ent.Notification) *NotificationDto {
	dto := NotificationDto{
		UserId: n_ent.UserID,
		Title:  n_ent.Title,
		Text:   n_ent.Text,
	}

	if n_ent.EventID != nil {
		dto.EventId = *n_ent.EventID
	}
	if n_ent.ContactID != nil {
		dto.ContactId = *n_ent.ContactID
	}
	if n_ent.TaskID != nil {
		dto.TaskId = *n_ent.TaskID
	}
	if n_ent.ProjectID != nil {
		dto.ProjectId = *n_ent.ProjectID
	}

	if n_ent.Edges.NotificationData != nil {
		n_data := n_ent.Edges.NotificationData
		dto.NotificationAddInfo = NotificationAddInfo{
			Type:         n_data.Type,
			TaskJson:     n_data.Task,
			ContactJson:  n_data.Contact,
			EventJson:    n_data.Event,
			MemberJson:   n_data.Member,
			ChatJson:     n_data.Chat,
			MessageJson:  n_data.Message,
			UserJson:     n_data.User,
			MetadataJson: n_data.Metadata,
			PluralCount:  n_data.PluralCount,
			convertedMap: make(map[string]interface{}),
		}

		dto.generateConvertedMap()
	}

	return &dto
}

func (dto *NotificationDto) generateConvertedMap() {
	if dto.NotificationAddInfo.EventJson != nil {
		dto.setConvertedMap("event", *dto.NotificationAddInfo.EventJson)
	}
	if dto.NotificationAddInfo.ContactJson != nil {
		dto.setConvertedMap("contact", *dto.NotificationAddInfo.ContactJson)
	}
	if dto.NotificationAddInfo.TaskJson != nil {
		dto.setConvertedMap("task", *dto.NotificationAddInfo.TaskJson)
	}
	if dto.NotificationAddInfo.MemberJson != nil {
		dto.setConvertedMap("member", *dto.NotificationAddInfo.MemberJson)
	}
	if dto.NotificationAddInfo.ChatJson != nil {
		dto.setConvertedMap("chat", *dto.NotificationAddInfo.ChatJson)
	}
	if dto.NotificationAddInfo.MessageJson != nil {
		dto.setConvertedMap("message", *dto.NotificationAddInfo.MessageJson)
	}
	if dto.NotificationAddInfo.UserJson != nil {
		dto.setConvertedMap("user", *dto.NotificationAddInfo.UserJson)
	}
	if dto.NotificationAddInfo.PluralCount != nil {
		dto.setConvertedMap("plural_count", strconv.FormatInt(*dto.NotificationAddInfo.PluralCount, 10))
	}
	if dto.NotificationAddInfo.MetadataJson != nil {
		dto.setConvertedMap("metadata", *dto.NotificationAddInfo.MetadataJson)
	}
}

func (dto *NotificationDto) setConvertedMap(key string, value string) {
	if dto.convertedMap == nil {
		dto.convertedMap = make(map[string]interface{})
	}

	var tmpVar interface{}

	err := json.Unmarshal([]byte(value), &tmpVar)
	if err == nil {
		dto.convertedMap[key] = tmpVar
	}
}

func (dto *NotificationDto) ParseAndSetNotificationData(notificationData map[string]string) error {
	if eventJson, ok := notificationData["event"]; ok && eventJson != "" {
		var event *events_v1.Event
		err := json.Unmarshal([]byte(eventJson), &event)
		if err != nil {
			return fmt.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		}

		dto.EventId = event.Id
		dto.EventJson = &eventJson
		dto.setConvertedMap("event", eventJson)
	}

	if contactJson, ok := notificationData["contact"]; ok && contactJson != "" {
		var contact *contacts_v1.Contact
		err := json.Unmarshal([]byte(contactJson), &contact)
		if err != nil {
			return fmt.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		}

		dto.ContactId = contact.Id
		dto.ContactJson = &contactJson

		dto.setConvertedMap("contact", contactJson)
	}

	if taskJson, ok := notificationData["task"]; ok && taskJson != "" {
		var task *tasks_v1.Task
		err := json.Unmarshal([]byte(taskJson), &task)
		if err != nil {
			return fmt.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		}

		dto.TaskId = task.Id
		dto.TaskJson = &taskJson

		dto.setConvertedMap("task", taskJson)
	}

	if projectJson, ok := notificationData["project"]; ok && projectJson != "" {
		var project *projects_v1.Project
		err := json.Unmarshal([]byte(projectJson), &project)
		if err != nil {
			return fmt.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		}

		dto.ProjectId = project.Id
		dto.ProjectJson = &projectJson

		dto.setConvertedMap("project", projectJson)
	}

	if memberJson, ok := notificationData["member"]; ok && memberJson != "" {
		dto.MemberJson = &memberJson
		dto.setConvertedMap("member", memberJson)
	}
	if chatJson, ok := notificationData["chat"]; ok && chatJson != "" {
		dto.ChatJson = &chatJson
		dto.setConvertedMap("chat", chatJson)
	}
	if messageJson, ok := notificationData["message"]; ok && messageJson != "" {
		dto.MessageJson = &messageJson
		dto.setConvertedMap("message", messageJson)
	}
	if userJson, ok := notificationData["user"]; ok && userJson != "" {
		dto.UserJson = &userJson
		dto.setConvertedMap("user", userJson)
	}
	if metadataString, ok := notificationData["metadata"]; ok && metadataString != "" {
		dto.MetadataJson = &metadataString
		dto.setConvertedMap("metadata", metadataString)
	}

	if dataType, ok := notificationData["type"]; ok && dataType != "" {
		dto.Type = &dataType
	}

	if pluralCount, ok := notificationData["plural_count"]; ok && pluralCount != "" {
		count, err := strconv.Atoi(pluralCount)
		if err != nil {
			return fmt.Errorf("notificationData[plural_count]->strconv.Atoi, err: %s", err.Error())
		}
		i64Count := int64(count)
		dto.PluralCount = &i64Count
		dto.setConvertedMap("plural_count", pluralCount)
	}

	return nil
}

func (dto *NotificationDto) GetConvertedMap() map[string]interface{} {
	return dto.convertedMap
}
