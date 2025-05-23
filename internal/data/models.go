package data

import (
	"encoding/json"
	"fmt"
	"strconv"

	chats_v1 "gitlab.calendaria.team/services/chats/api/chats/v1"
	contacts_v1 "gitlab.calendaria.team/services/contacts/api/contacts/v1"
	events_v1 "gitlab.calendaria.team/services/events/api/events/v1"
	iam_v1 "gitlab.calendaria.team/services/iam/api/iam/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	projects_v1 "gitlab.calendaria.team/services/pms/projects/api/projects/v1"
	tasks_v1 "gitlab.calendaria.team/services/pms/tasks/api/tasks/v1"
)

type FilterNotificationsDto struct {
	UserID int64
	Type   string
}

type ReadNotificationDto struct {
	UserID         int64
	Type           string
	NotificationID int64
}

type Counter struct {
	UserID int    `json:"user_id"`
	Type   string `json:"type"`
	Count  int    `json:"count"`
}

type NotificationDto struct {
	UserID       int64
	Title        string
	Text         string
	EventID      int64
	ContactID    int64
	TaskID       int64
	ProjectID    int64
	TargetUserID int64

	NotificationAddInfo
}

type NotificationAddInfo struct {
	Type         *string
	TaskJSON     *string
	ProjectJSON  *string
	ContactJSON  *string
	EventJSON    *string
	MemberJSON   *string
	ChatJSON     *string
	MessageJSON  *string
	UserJSON     *string
	MetadataJSON *string
	PluralCount  *int64

	convertedMap map[string]interface{}
}

func getUserJSON(mapUsers map[int64]*iam_v1.UserShort, userID *int64) *string {
	deletedAccount := "{\"username\":\"Deleted Account\",\"name\":\"Deleted Account\"}"

	if userID == nil {
		return &deletedAccount
	}

	user, ok := mapUsers[*userID]
	if !ok {
		return &deletedAccount
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return &deletedAccount
	}

	userString := string(userJSON)
	return &userString
}

func FromEnt(nEnt *ent.Notification, mapUsers map[int64]*iam_v1.UserShort) *NotificationDto {
	dto := NotificationDto{
		UserID: nEnt.UserID,
		Title:  nEnt.Title,
		Text:   nEnt.Text,
	}

	if nEnt.EventID != nil {
		dto.EventID = *nEnt.EventID
	}
	if nEnt.ContactID != nil {
		dto.ContactID = *nEnt.ContactID
	}
	if nEnt.TaskID != nil {
		dto.TaskID = *nEnt.TaskID
	}
	if nEnt.ProjectID != nil {
		dto.ProjectID = *nEnt.ProjectID
	}

	nData := nEnt.Edges.NotificationData
	if nData != nil {
		dto.NotificationAddInfo = NotificationAddInfo{
			Type:         nData.Type,
			TaskJSON:     nData.Task,
			ContactJSON:  nData.Contact,
			EventJSON:    nData.Event,
			MemberJSON:   nData.Member,
			ChatJSON:     nData.Chat,
			MessageJSON:  nData.Message,
			UserJSON:     getUserJSON(mapUsers, nData.TargetUserID),
			MetadataJSON: nData.Metadata,
			PluralCount:  nData.PluralCount,
			convertedMap: make(map[string]interface{}),
		}

		dto.generateConvertedMap()
	}

	return &dto
}

func (dto *NotificationDto) generateConvertedMap() {
	if dto.NotificationAddInfo.EventJSON != nil {
		dto.setConvertedMap("event", *dto.NotificationAddInfo.EventJSON)
	}
	if dto.NotificationAddInfo.ContactJSON != nil {
		dto.setConvertedMap("contact", *dto.NotificationAddInfo.ContactJSON)
	}
	if dto.NotificationAddInfo.TaskJSON != nil {
		dto.setConvertedMap("task", *dto.NotificationAddInfo.TaskJSON)
	}
	if dto.NotificationAddInfo.MemberJSON != nil {
		dto.setConvertedMap("member", *dto.NotificationAddInfo.MemberJSON)
	}
	if dto.NotificationAddInfo.ChatJSON != nil {
		dto.setConvertedMap("chat", *dto.NotificationAddInfo.ChatJSON)
	}
	if dto.NotificationAddInfo.MessageJSON != nil {
		dto.setConvertedMap("message", *dto.NotificationAddInfo.MessageJSON)
	}
	if dto.NotificationAddInfo.UserJSON != nil {
		dto.setConvertedMap("user", *dto.NotificationAddInfo.UserJSON)
	}
	if dto.NotificationAddInfo.PluralCount != nil {
		dto.setConvertedMap("plural_count", strconv.FormatInt(*dto.NotificationAddInfo.PluralCount, 10))
	}
	if dto.NotificationAddInfo.MetadataJSON != nil {
		dto.setConvertedMap("metadata", *dto.NotificationAddInfo.MetadataJSON)
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

//nolint: funlen, gocognit // it's a DTO
func (dto *NotificationDto) ParseAndSetNotificationData(notificationData map[string]string) error {
	if eventJSON, ok := notificationData["event"]; ok && eventJSON != "" {
		var event *events_v1.Event
		err := json.Unmarshal([]byte(eventJSON), &event)
		if err != nil {
			return fmt.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		}

		dto.EventID = event.GetId()
		dto.EventJSON = &eventJSON
		dto.setConvertedMap("event", eventJSON)
	}

	if contactJSON, ok := notificationData["contact"]; ok && contactJSON != "" {
		var contact *contacts_v1.Contact
		err := json.Unmarshal([]byte(contactJSON), &contact)
		if err != nil {
			return fmt.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		}

		dto.ContactID = contact.GetId()
		dto.ContactJSON = &contactJSON
		dto.setConvertedMap("contact", contactJSON)
	}

	if taskJSON, ok := notificationData["task"]; ok && taskJSON != "" {
		var task *tasks_v1.Task
		err := json.Unmarshal([]byte(taskJSON), &task)
		if err != nil {
			return fmt.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		}

		dto.TaskID = task.GetId()
		dto.TaskJSON = &taskJSON
		dto.setConvertedMap("task", taskJSON)
	}

	if projectJSON, ok := notificationData["project"]; ok && projectJSON != "" {
		var project *projects_v1.Project
		err := json.Unmarshal([]byte(projectJSON), &project)
		if err != nil {
			return fmt.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		}

		dto.ProjectID = project.GetId()
		dto.ProjectJSON = &projectJSON
		dto.setConvertedMap("project", projectJSON)
	}

	if memberJSON, ok := notificationData["member"]; ok && memberJSON != "" {
		dto.MemberJSON = &memberJSON
		dto.setConvertedMap("member", memberJSON)
	}
	if chatJSON, ok := notificationData["chat"]; ok && chatJSON != "" {
		dto.ChatJSON = &chatJSON
		dto.setConvertedMap("chat", chatJSON)
	}
	if messageJSON, ok := notificationData["message"]; ok && messageJSON != "" {
		dto.MessageJSON = &messageJSON
		dto.setConvertedMap("message", messageJSON)
	}
	if userJSON, ok := notificationData["user"]; ok && userJSON != "" {
		var user *iam_v1.UserShort
		err := json.Unmarshal([]byte(userJSON), &user)
		if err != nil {
			return fmt.Errorf("sendNotifications: json.Unmarshal: %s", err.Error())
		}

		dto.TargetUserID = user.GetId()
		dto.UserJSON = &userJSON
		dto.setConvertedMap("user", userJSON)
	}
	if metadataString, ok := notificationData["metadata"]; ok && metadataString != "" {
		dto.MetadataJSON = &metadataString
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

		if count > 0 {
			i64Count := int64(count)

			dto.PluralCount = &i64Count
			dto.setConvertedMap("plural_count", pluralCount)
		}
	}

	return nil
}

func (dto *NotificationDto) GetConvertedMap() map[string]interface{} {
	return dto.convertedMap
}

func (dto *NotificationDto) GetChat() *chats_v1.Chat {
	if dto.ChatJSON == nil {
		return nil
	}

	var chat chats_v1.Chat

	err := json.Unmarshal([]byte(*dto.ChatJSON), &chat)
	if err != nil {
		return nil
	}

	return &chat
}
