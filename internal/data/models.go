package data

import (
	"encoding/json"
	"fmt"
	"strconv"

	contacts_v1 "gitlab.calendaria.team/services/contacts/api/contacts/v1"
	events_v1 "gitlab.calendaria.team/services/events/api/events/v1"
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

	NotificationAddInfo
}

type NotificationAddInfo struct {
	Type         *string
	TaskJson     *string
	ContactJson  *string
	EventJson    *string
	MemberJson   *string
	ChatJson     *string
	MessageJson  *string
	MetadataJson *string
	PluralCount  *int64

	convertedMap map[string]interface{}
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
	if metadataString, ok := notificationData["metadata"]; ok && metadataString != "" {
		dto.MetadataJson = &metadataString
		dto.setConvertedMap("metadata", metadataString)
	}

	if dataType, ok := notificationData["type"]; ok && dataType != "" {
		dto.Type = &dataType
	}

	if plularCount, ok := notificationData["plural_count"]; ok && plularCount != "" {
		count, err := strconv.Atoi(plularCount)
		if err != nil {
			return fmt.Errorf("notificationData[plural_count]->strconv.Atoi, err: %s", err.Error())
		}
		i64Count := int64(count)
		dto.PluralCount = &i64Count
	}

	return nil
}

func (dto *NotificationDto) GetConvertedMap() map[string]interface{} {
	return dto.convertedMap
}

func (dto *NotificationDto) setConvertedMap(key string, value string) {
	var tmpVar interface{}

	err := json.Unmarshal([]byte(value), &tmpVar)
	if err != nil {
		dto.convertedMap[key] = tmpVar
	}
}
