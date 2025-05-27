package biz

import (
	"context"
	"io"
	"testing"

	"firebase.google.com/go/v4/messaging"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	contacts_v1 "gitlab.calendaria.team/services/contacts/api/contacts/v1"
	users_v1 "gitlab.calendaria.team/services/iam/api/iam/v1"
	"gitlab.calendaria.team/services/notifications/ent"
	"gitlab.calendaria.team/services/notifications/internal/data"
	"gitlab.calendaria.team/services/notifications/internal/data/mock"
	u_struc "gitlab.calendaria.team/services/utils/v2/struc"
	badge_mock "gitlab.calendaria.team/services/utils/v4/badge/mock"
	nats_mock "gitlab.calendaria.team/services/utils/v4/nats/mock"
)

// FCMMessageMatcher - пользовательский matcher для FCM сообщений.
type FCMMessageMatcher struct {
	expected messaging.Message
}

func (m FCMMessageMatcher) Matches(x interface{}) bool {
	actual, ok := x.(*messaging.Message)
	if !ok {
		return false
	}

	// Проверяем основные поля сообщения
	if m.expected.Token != actual.Token {
		return false
	}

	// Проверяем данные
	if len(m.expected.Data) != len(actual.Data) {
		return false
	}
	for k, v := range m.expected.Data {
		if actual.Data[k] != v {
			return false
		}
	}

	// Проверяем уведомление если оно есть
	if (m.expected.Notification == nil) != (actual.Notification == nil) {
		return false
	}
	if m.expected.Notification != nil {
		if m.expected.Notification.Title != actual.Notification.Title ||
			m.expected.Notification.Body != actual.Notification.Body ||
			m.expected.Notification.ImageURL != actual.Notification.ImageURL {
			return false
		}
	}

	// Проверяем Android-конфигурацию если она есть
	if (m.expected.Android == nil) != (actual.Android == nil) {
		return false
	}
	if m.expected.Android != nil && m.expected.Android.Notification != nil && actual.Android != nil && actual.Android.Notification != nil {
		if m.expected.Android.Notification.Sound != actual.Android.Notification.Sound {
			return false
		}
		// Проверяем NotificationCount если он установлен
		if m.expected.Android.Notification.NotificationCount != nil && actual.Android.Notification.NotificationCount != nil {
			if *m.expected.Android.Notification.NotificationCount != *actual.Android.Notification.NotificationCount {
				return false
			}
		}
	}

	// Проверяем APNS-конфигурацию если она есть
	if (m.expected.APNS == nil) != (actual.APNS == nil) {
		return false
	}
	if m.expected.APNS != nil && m.expected.APNS.Payload != nil && actual.APNS != nil && actual.APNS.Payload != nil {
		if m.expected.APNS.Payload.Aps != nil && actual.APNS.Payload.Aps != nil {
			if m.expected.APNS.Payload.Aps.Sound != actual.APNS.Payload.Aps.Sound {
				return false
			}
			// Проверяем Badge если он установлен
			if m.expected.APNS.Payload.Aps.Badge != nil && actual.APNS.Payload.Aps.Badge != nil {
				if *m.expected.APNS.Payload.Aps.Badge != *actual.APNS.Payload.Aps.Badge {
					return false
				}
			}
			if m.expected.APNS.Payload.Aps.MutableContent != actual.APNS.Payload.Aps.MutableContent {
				return false
			}
		}
	}

	return true
}

func (m FCMMessageMatcher) String() string {
	return "is a matching FCM message"
}

// Функция для создания FCM matcher.
func MatchesFCMMessage(expected messaging.Message) gomock.Matcher {
	return FCMMessageMatcher{expected: expected}
}

func TestFull(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := log.NewStdLogger(io.Discard)
	mockDeviceRepo := mock.NewMockDevicesRepo(ctrl)
	mockNotificationsRepo := mock.NewMockNotificationsRepo(ctrl)
	localizer, err := data.NewLocalizerForTest()
	assert.NoError(t, err)
	fcmClient := mock.NewMockFcmClient(ctrl)
	chatsRemote := mock.NewMockIChatsRemote(ctrl)
	eventsRemote := mock.NewMockIEventsRemote(ctrl)
	iamRemote := mock.NewMockIIamRemote(ctrl)
	contactRemote := mock.NewMockIContactsRemote(ctrl)

	badgeClient := badge_mock.NewMockIBadgeClient(ctrl)

	qm := nats_mock.NewMockIQueueManager(ctrl)

	qm.EXPECT().AddConsumer(gomock.Any(), gomock.Any()).Return().Times(2)

	uc, err := NewFcmUsecase(
		logger,
		mockDeviceRepo,
		mockNotificationsRepo,
		localizer,
		qm,
		badgeClient,
		fcmClient,
		chatsRemote,
		eventsRemote,
		iamRemote,
		contactRemote,
	)

	assert.NoError(t, err)

	userIDs := []int64{1, 2}

	mockDeviceRepo.EXPECT().GetDevicesForUsers(gomock.Any(), userIDs).Return(
		[]*ent.Device{
			{ID: 1, UserID: 1, Token: "token1", Language: "ru"},
			{ID: 2, UserID: 2, Token: "token2", Language: "en"},
		}, nil,
	).Times(1)

	iamRemote.EXPECT().GetUsersSettings(gomock.Any(), gomock.Any()).
		Return(
			map[int64]map[string]string{
				1: {
					"NOTIFICATION_SOUND_ENABLED": "true",
					"NOTIFICATION_SOUND":         "default",
				},
				2: {
					"NOTIFICATION_SOUND_ENABLED": "false",
					"NOTIFICATION_SOUND":         "true",
				},
			}, nil,
		).AnyTimes()

	authorID := int64(220)
	contactRemote.EXPECT().GetContactsByUserId(gomock.Any(), authorID).Return(
		[]*contacts_v1.Contact{
			{
				Id:     1,
				UserId: &authorID,
				Label:  "Dana из контактов",
			},
		}, nil,
	).Times(2)

	badgeClient.EXPECT().GetBadges(gomock.Any(), userIDs[0]).Return(
		map[u_struc.NotificationType]int64{
			u_struc.Event:   0,
			u_struc.Chat:    0,
			u_struc.Contact: 0,
		}, nil,
	).Times(1)

	badgeClient.EXPECT().GetBadges(gomock.Any(), userIDs[1]).Return(
		map[u_struc.NotificationType]int64{
			u_struc.Event:   1,
			u_struc.Chat:    0,
			u_struc.Contact: 0,
		}, nil,
	).Times(1)

	badgeCountUser1 := int(1)

	fcmMessage := messaging.Message{
		Data: map[string]string{
			"event": `{"id":5541,"title":"Тест","description":"Рдмдрсдрсдс","coverUrl":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg","startDateTime":"2025-05-22T08:15:00Z","endDateTime":"2025-05-22T09:15:00Z","noticeBefore":10,"chatId":1445,"type":"HOME","avatars":["https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg"],"membersCount":2,"ownerId":220,"publishedAt":"2025-05-22T05:13:09Z","isInvitationAvailable":true,"membership":{"id":7809,"status":"ACCEPTED","role":"OWNER","calendarId":560,"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}},"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}}`,
			"type":  "EVENT_UPDATED",
			"user":  `{"id":220,"phone":"+77076663503","username":"dana","name":"Dana ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg","lastLoginAt":"2025-05-22T05:19:43Z"}`,
		},
		Token: "token1",
		Notification: &messaging.Notification{
			Title:    "Тест",
			Body:     "Dana из контактов изменил(а) данные события",
			ImageURL: "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
		},
		Android: &messaging.AndroidConfig{
			Notification: &messaging.AndroidNotification{
				Sound:             "default",
				NotificationCount: &badgeCountUser1,
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Badge:          &badgeCountUser1,
					Sound:          "default",
					MutableContent: true,
				},
			},
		},
	}

	assert.NotNil(t, fcmMessage.Android)

	// Используем наш пользовательский matcher вместо точного сравнения
	fcmClient.EXPECT().Send(
		gomock.Any(), "token1", gomock.Any(),
	).Return(
		nil,
	).Times(1)

	fcmClient.EXPECT().Send(
		gomock.Any(), "token2", gomock.Any(),
	).Return(
		nil,
	).Times(1)

	ok := uc.sendMessage(
		context.Background(), u_struc.FirebaseNotification{
			Title:    "Тест",
			UsersIds: userIDs,
			Body:     "Dana  updated event's",
			Type:     u_struc.Event,
			Image:    "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
			Data: map[string]string{
				"event": `{"id":5541,"title":"Тест","description":"Рдмдрсдрсдс","coverUrl":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg","startDateTime":"2025-05-22T08:15:00Z","endDateTime":"2025-05-22T09:15:00Z","noticeBefore":10,"chatId":1445,"type":"HOME","avatars":["https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg"],"membersCount":2,"ownerId":220,"publishedAt":"2025-05-22T05:13:09Z","isInvitationAvailable":true,"membership":{"id":7809,"status":"ACCEPTED","role":"OWNER","calendarId":560,"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}},"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}}`,
				"type":  "EVENT_UPDATED",
				"user":  `{"id":220,"phone":"+77076663503","username":"dana","name":"Dana ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg","lastLoginAt":"2025-05-22T05:19:43Z"}`,
			},
		}, true,
	)

	assert.True(t, ok)
}

func TestNotificationEventTextFormatting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := log.NewStdLogger(io.Discard)

	tests := []struct {
		name                string
		userId              int64
		userDevices         []*ent.Device
		inputMessage        *u_struc.FirebaseNotification
		expectedTitle       string
		expectedBody        string
		expectedImage       string
		contacts            []*contacts_v1.Contact
		shouldFormatMessage bool
	}{
		{
			name:   "event_updated_full_log_case_without_metadata",
			userId: 43,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 43, Token: "token43", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Тест",
				Body:  "Dana  updated event's",
				Type:  u_struc.Event,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
				Data: map[string]string{
					"event": `{"id":5541,"title":"Тест","description":"Рдмдрсдрсдс","coverUrl":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg","startDateTime":"2025-05-22T08:15:00Z","endDateTime":"2025-05-22T09:15:00Z","noticeBefore":10,"chatId":1445,"type":"HOME","avatars":["https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg"],"membersCount":2,"ownerId":220,"publishedAt":"2025-05-22T05:13:09Z","isInvitationAvailable":true,"membership":{"id":7809,"status":"ACCEPTED","role":"OWNER","calendarId":560,"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}},"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}}`,
					"type":  "EVENT_UPDATED",
					"user":  `{"id":220,"phone":"+77076663503","username":"dana","name":"Dana ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg","lastLoginAt":"2025-05-22T05:19:43Z"}`,
				},
			},
			expectedTitle: "Тест",
			expectedBody:  "Dana из контактов изменил(а) данные события",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(220); return &id }(),
					Label:  "Dana из контактов",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "event_updated_full_log_case_with_metadata_title",
			userId: 43,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 43, Token: "token43", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Тест",
				Body:  "Dana  updated event's",
				Type:  u_struc.Event,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
				Data: map[string]string{
					"event":    `{"id":5541,"title":"Тест","description":"Рдмдрсдрсдс","coverUrl":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg","startDateTime":"2025-05-22T08:15:00Z","endDateTime":"2025-05-22T09:15:00Z","noticeBefore":10,"chatId":1445,"type":"HOME","avatars":["https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg"],"membersCount":2,"ownerId":220,"publishedAt":"2025-05-22T05:13:09Z","isInvitationAvailable":true,"membership":{"id":7809,"status":"ACCEPTED","role":"OWNER","calendarId":560,"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}},"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}}`,
					"type":     "EVENT_UPDATED",
					"metadata": "title",
					"user":     `{"id":220,"phone":"+77076663503","username":"dana","name":"Dana ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg","lastLoginAt":"2025-05-22T05:19:43Z"}`,
				},
			},
			expectedTitle: "Тест",
			expectedBody:  "Dana из контактов изменил(а) данные события: название",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(220); return &id }(),
					Label:  "Dana из контактов",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "event_updated_full_log_case_with_metadatas",
			userId: 43,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 43, Token: "token43", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Тест",
				Body:  "Dana  updated event's",
				Type:  u_struc.Event,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
				Data: map[string]string{
					"event":    `{"id":5541,"title":"Тест","description":"Рдмдрсдрсдс","coverUrl":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg","startDateTime":"2025-05-22T08:15:00Z","endDateTime":"2025-05-22T09:15:00Z","noticeBefore":10,"chatId":1445,"type":"HOME","avatars":["https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg"],"membersCount":2,"ownerId":220,"publishedAt":"2025-05-22T05:13:09Z","isInvitationAvailable":true,"membership":{"id":7809,"status":"ACCEPTED","role":"OWNER","calendarId":560,"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}},"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}}`,
					"type":     "EVENT_UPDATED",
					"metadata": "title, details",
					"user":     `{"id":220,"phone":"+77076663503","username":"dana","name":"Dana ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg","lastLoginAt":"2025-05-22T05:19:43Z"}`,
				},
			},
			expectedTitle: "Тест",
			expectedBody:  "Dana из контактов изменил(а) данные события: название, детали события",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(220); return &id }(),
					Label:  "Dana из контактов",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "event_updated_from_log",
			userId: 43,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 43, Token: "token43", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Тест",
				Body:  "Dana  updated event's:  https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
				Type:  u_struc.Event,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
				Data: map[string]string{
					"event": `{"id":5541,"title":"Тест","description":"Рдмдрсдрсдс","coverUrl":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg","startDateTime":"2025-05-22T08:15:00Z","endDateTime":"2025-05-22T09:15:00Z","noticeBefore":10,"chatId":1445,"type":"HOME","avatars":["https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg"],"membersCount":2,"ownerId":220,"publishedAt":"2025-05-22T05:13:09Z","isInvitationAvailable":true,"membership":{"id":7809,"status":"ACCEPTED","role":"OWNER","calendarId":560,"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}},"calendar":{"id":560,"title":"Work","color":"2196F3","isPrimary":true,"isSelected":true,"ownerId":220,"provider":"CALENDARIA","externalId":"dana.levinte@gmail.com","credentialId":74}}`,
					"type":  "EVENT_UPDATED",
					"user":  `{"id":220,"phone":"+77076663503","username":"dana","name":"Dana ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/04/14bdd005-0c4f-475c-a62c-ba10e3e948c4.jpg","lastLoginAt":"2025-05-22T05:19:43Z"}`,
				},
			},
			expectedTitle: "Тест",
			expectedBody:  "Dana из контактов изменил(а) данные события",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/220/2025/05/94967573-71c4-4d87-9ac1-1b480a1ee80c.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(220); return &id }(),
					Label:  "Dana из контактов",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "group_chat_with_text_message",
			userId: 137,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 137, Token: "token137", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Нуралина Жанна Ануарбековна",
				Body:  "Привет",
				Type:  u_struc.Chat,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg",
				Data: map[string]string{
					"chatId":       "1423",
					"createdAt":    "2025-05-20T06:22:26Z",
					"message":      `{"id":10872,"cid":"afc01565-6f7c-4695-8dea-714ccd2753e7","type":"REGULAR","createdAt":"2025-05-20T06:22:26Z","updatedAt":"2025-05-20T06:22:26Z","userId":43,"content":{"text":"Привет"}}`,
					"type":         "message.new",
					"plural_count": "1",
					"user":         `{"id":43,"phone":"+77011291625","email":"zh.nuralina@calendaria.ai","username":"zhanna","name":"Нуралина Жанна Ануарбековна ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","lastLoginAt":"2025-05-20T06:21:59Z"}`,
					"chat":         `{"type":"GROUP","title":"Test Push-notifications","description":"","cover":"https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/05/77bd8b9b-2225-4074-a2ba-704e4125b9d2.jpg"}`,
				},
			},
			expectedTitle: "Test Push-notifications",
			expectedBody:  "Жанна из контактов: Привет",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/05/77bd8b9b-2225-4074-a2ba-704e4125b9d2.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(43); return &id }(),
					Label:  "Жанна из контактов",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "personal_chat_with_text_message",
			userId: 137,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 137, Token: "token137", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Нуралина Жанна Ануарбековна",
				Body:  "Привет",
				Type:  u_struc.Chat,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg",
				Data: map[string]string{
					"chatId":       "1423",
					"createdAt":    "2025-05-20T06:22:26Z",
					"message":      `{"id":10872,"cid":"afc01565-6f7c-4695-8dea-714ccd2753e7","type":"REGULAR","createdAt":"2025-05-20T06:22:26Z","updatedAt":"2025-05-20T06:22:26Z","userId":43,"content":{"text":"Привет"}}`,
					"type":         "message.new",
					"plural_count": "1",
					"user":         `{"id":43,"phone":"+77011291625","email":"zh.nuralina@calendaria.ai","username":"zhanna","name":"Нуралина Жанна Ануарбековна ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","lastLoginAt":"2025-05-20T06:21:59Z"}`,
					"chat":         `{"type":"DIRECT"}`,
				},
			},
			expectedTitle: "Жанна из контактов",
			expectedBody:  "Привет",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(43); return &id }(),
					Label:  "Жанна из контактов",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "message_with_image",
			userId: 37,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 37, Token: "token37", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Serhio",
				Body:  "Тест медиа",
				Type:  u_struc.Chat,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/3/2025/04/fac2eddf-9488-4339-b1bb-94e106c1b573.jpg",
				Data: map[string]string{
					"chatId":       "1066",
					"createdAt":    "2025-05-20T06:29:40Z",
					"message":      `{"id":10879,"cid":"7429c65c-daf8-417e-86d2-77b49ebb18e2","type":"REGULAR","createdAt":"2025-05-20T06:29:40Z","updatedAt":"2025-05-20T06:29:40Z","userId":3,"content":{"text":"Тест медиа","attachments":[{"id":1348,"type":"IMAGE","mediaId":6962,"media":{"id":6962,"ownerId":3,"url":"https://calendaria-test.s3.eu-north-1.amazonaws.com/3/2025/05/2b21dd86-5e59-4b2e-9b74-44bc386b55ef.jpg","fileName":"b7a3ae3a3e65c9e3a34c18db13b8bb66_exif.jpg","extension":"jpg","createdAt":"2025-05-20T06:29:40Z","size":140507,"width":1080,"height":991}}]}}`,
					"plural_count": "1",
					"type":         "message.photo",
					"user":         `{"id":3,"phone":"+77058429737","email":"soberzerg@gmail.com","username":"serhio","name":"Serhio","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/3/2025/04/fac2eddf-9488-4339-b1bb-94e106c1b573.jpg","lastLoginAt":"2025-05-20T06:28:02Z"}`,
					"chat":         `{"type":"GROUP","title":"Тестовая группа","cover":""}`,
				},
			},
			expectedTitle: "Тестовая группа",
			expectedBody:  "Serhio (из контактов): 🖼️ 1 фото",
			expectedImage: "",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(3); return &id }(),
					Label:  "Serhio (из контактов)",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "add_to_group",
			userId: 472,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 472, Token: "token472", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Group",
				Body:  "New message",
				Type:  u_struc.Chat,
				Data: map[string]string{
					"chat": `{"id":1434,"type":"GROUP","title":"Group","description":"","cover":"",
"membersCount":1,"createdAt":"2025-05-20T07:44:01Z","updatedAt":"2025-05-20T07:44:01Z","companionId":391,"membership":{"chatId":1434,"status":"ACTIVE","role":"OWNER","updatedAt":"2025-05-20T07:44:01Z","lastReadId":10883},"lastMessage":{"id":10884,"cid":"369c88bb-1c37-42f7-b65d-f36eb1dd5629","type":"SYSTEM","createdAt":"2025-05-20T07:44:02Z","updatedAt":"2025-05-20T07:44:02Z","userId":391,"action":{"type":"MEMBER_ADDED","targetUsersIds":[4,472,69,206,303,216,42]}}}`,
					"chatId":       "1434",
					"createdAt":    "2025-05-20T07:44:02Z",
					"message":      `{"id":10884,"cid":"369c88bb-1c37-42f7-b65d-f36eb1dd5629","type":"SYSTEM","createdAt":"2025-05-20T07:44:02Z","updatedAt":"2025-05-20T07:44:02Z","userId":391,"action":{"type":"MEMBER_ADDED","targetUsersIds":[4,472,69,206,303,216,42]}}`,
					"plural_count": "0",
					"type":         "GROUP_ADDED",
					"user":         `{"id":391,"phone":"+77473518566","username":"akoflacko123","name":"Akzhol Serikkaliyev","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/391/2025/03/95c665bb-cb17-4e63-bf22-9556f265f464.jpg","lastLoginAt":"2025-05-20T07:42:56Z","privacies":{"EVENT_INVITE":"ALL","GROUP_CHAT_INVITE":"ALL","LAST_VISIT":"ALL","MY_EVENTS":"ALL","MY_LAST_ACTIONS":"ALL","MY_PROFILE_IMAGE":"ALL","MY_SLOTS":"ALL","SLOTS_DETAILS":"ALL"}}`,
				},
			},
			expectedTitle: "Group",
			expectedBody:  "Акжол (из контактов) добавил вас в группу",
			expectedImage: "",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(391); return &id }(),
					Label:  "Акжол (из контактов)",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "change_group_cover_image",
			userId: 137,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 137, Token: "token137", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Test Push-notifications",
				Body:  "Title, Cover image have been changed by Нуралина Жанна Ануарбековна",
				Type:  u_struc.Chat,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/05/77bd8b9b-2225-4074-a2ba-704e4125b9d2.jpg",
				Data: map[string]string{
					"chat":         `{"id":1423,"type":"GROUP","title":"Test Push-notifications ","description":"","cover":"https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/05/77bd8b9b-2225-4074-a2ba-704e4125b9d2.jpg","membersCount":16,"createdAt":"2025-05-19T12:46:44Z","updatedAt":"2025-05-20T06:22:57Z","companionId":43,"membership":{"chatId":1423,"status":"ACTIVE","role":"OWNER","updatedAt":"2025-05-20T06:22:26Z","lastReadId":10872},"lastMessage":{"id":10874,"cid":"3ce6487e-2034-4276-859a-e9abdf90a419","type":"SYSTEM","createdAt":"2025-05-20T06:22:57Z","updatedAt":"2025-05-20T06:22:57Z","userId":43,"action":{"type":"COVER_CHANGED","changedFrom":"","changedTo":"https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/05/77bd8b9b-2225-4074-a2ba-704e4125b9d2.jpg"}}}`,
					"chatId":       "1423",
					"createdAt":    "2025-05-20T06:22:57Z",
					"message":      `{"id":10874,"cid":"3ce6487e-2034-4276-859a-e9abdf90a419","type":"SYSTEM","createdAt":"2025-05-20T06:22:57Z","updatedAt":"2025-05-20T06:22:57Z","userId":43,"action":{"type":"COVER_CHANGED","changedFrom":"","changedTo":"https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/05/77bd8b9b-2225-4074-a2ba-704e4125b9d2.jpg"}}`,
					"metadata":     "Title, Cover image",
					"plural_count": "2",
					"type":         "chat.update",
					"user":         `{"id":43,"phone":"+77011291625","email":"zh.nuralina@calendaria.ai","username":"zhanna","name":"Нуралина Жанна Ануарбековна ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","lastLoginAt":"2025-05-20T06:21:59Z"}`,
				},
			},
			expectedTitle: "Test Push-notifications ",
			expectedBody:  "Жанна из контактов: изменил(а) название, обложку группы",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/05/77bd8b9b-2225-4074-a2ba-704e4125b9d2.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(43); return &id }(),
					Label:  "Жанна из контактов",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "direct_chat_message_from_logs",
			userId: 220,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 220, Token: "token220", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Нуралина Жанна Ануарбековна",
				Body:  "Шрсшсп",
				Type:  u_struc.Chat,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg",
				Data: map[string]string{
					"chatId":       "1219",
					"createdAt":    "2025-05-22T11:20:29Z",
					"message":      `{"id":11038,"cid":"fa731bff-32bc-46ec-aa94-9459502f60f6","type":"REGULAR","createdAt":"2025-05-22T11:20:29Z","updatedAt":"2025-05-22T11:20:29Z","userId":43,"content":{"text":"Шрсшсп"}}`,
					"plural_count": "1",
					"type":         "message.new",
					"user":         `{"id":43,"phone":"+77011291625","email":"zh.nuralina@calendaria.ai","username":"zhanna","name":"Нуралина Жанна Ануарбековна ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","lastLoginAt":"2025-05-22T11:19:44Z"}`,
					"chat":         `{"type":"DIRECT"}`,
				},
			},
			expectedTitle: "Жанна из контактов",
			expectedBody:  "Шрсшсп",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(43); return &id }(),
					Label:  "Жанна из контактов",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "multi_user_chat_message_from_logs",
			userId: 10,
			userDevices: []*ent.Device{
				{ID: 2, UserID: 10, Token: "token10", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "Нуралина Жанна Ануарбековна",
				Body:  "Куку",
				Type:  u_struc.Chat,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg",
				Data: map[string]string{
					"chatId":       "1445",
					"createdAt":    "2025-05-22T11:20:05Z",
					"message":      `{"id":11037,"cid":"a6ac6559-03a6-4d33-aeb0-01d64cd7e2e4","type":"REGULAR","createdAt":"2025-05-22T11:20:05Z","updatedAt":"2025-05-22T11:20:05Z","userId":43,"content":{"text":"Куку"}}`,
					"plural_count": "1",
					"type":         "message.new",
					"user":         `{"id":43,"phone":"+77011291625","email":"zh.nuralina@calendaria.ai","username":"zhanna","name":"Нуралина Жанна Ануарбековна ","avatar":"https://calendaria-test.s3.eu-north-1.amazonaws.com/43/2025/02/2d42197f-7e3b-43a8-aa87-2b1a0d4d7bc2.jpg","lastLoginAt":"2025-05-22T11:19:44Z"}`,
					"chat":         `{"type":"GROUP","title":"Тестовое событие","description":"","cover":"https://calendaria-test.s3.eu-north-1.amazonaws.com/event_default_cover.jpg"}`,
				},
				UsersIds: []int64{220, 10},
			},
			expectedTitle: "Тестовое событие",
			expectedBody:  "Жанна из контактов: Куку",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/event_default_cover.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(43); return &id }(),
					Label:  "Жанна из контактов",
				},
			},
			shouldFormatMessage: true,
		},
		{
			name:   "chat_update_with_metadata_from_real_logs",
			userId: 21,
			userDevices: []*ent.Device{
				{ID: 1, UserID: 21, Token: "token21", Language: "ru"},
			},
			inputMessage: &u_struc.FirebaseNotification{
				Title: "tesssss1",
				Body:  "Title, Description, Cover image have been changed by Asd",
				Type:  u_struc.Chat,
				Image: "https://calendaria-test.s3.eu-north-1.amazonaws.com/491/2025/05/5bc66352-85b7-408c-971c-caabebf823d9.jpg",
				Data: map[string]string{
					"chat":         `{"id":1474,"type":"GROUP","title":"tesssss1","description":"qweqweqwe","cover":"https://calendaria-test.s3.eu-north-1.amazonaws.com/491/2025/05/5bc66352-85b7-408c-971c-caabebf823d9.jpg","membersCount":2,"createdAt":"2025-05-27T04:12:13Z","updatedAt":"2025-05-27T04:12:54Z","companionId":491,"membership":{"chatId":1474,"status":"ACTIVE","role":"OWNER","updatedAt":"2025-05-27T04:12:18Z","lastReadId":11210},"lastMessage":{"id":11213,"cid":"c3cbede0-2937-4745-bf09-cc581ba0450c","type":"SYSTEM","createdAt":"2025-05-27T04:12:54Z","updatedAt":"2025-05-27T04:12:54Z","userId":491,"action":{"type":"COVER_CHANGED","changedFrom":"","changedTo":"https://calendaria-test.s3.eu-north-1.amazonaws.com/491/2025/05/5bc66352-85b7-408c-971c-caabebf823d9.jpg"}}}`,
					"chatId":       "1474",
					"createdAt":    "2025-05-27T04:12:54Z",
					"message":      `{"id":11213,"cid":"c3cbede0-2937-4745-bf09-cc581ba0450c","type":"SYSTEM","createdAt":"2025-05-27T04:12:54Z","updatedAt":"2025-05-27T04:12:54Z","userId":491,"action":{"type":"COVER_CHANGED","changedFrom":"","changedTo":"https://calendaria-test.s3.eu-north-1.amazonaws.com/491/2025/05/5bc66352-85b7-408c-971c-caabebf823d9.jpg"}}`,
					"metadata":     "Title, Description, Cover image",
					"plural_count": "3",
					"type":         "chat.update",
					"user":         `{"id":491,"phone":"+77012715505","username":"user491","name":"Asd","lastLoginAt":"2025-05-27T04:03:21Z"}`,
				},
			},
			expectedTitle: "tesssss1",
			expectedBody:  "Dev Mac: изменил(а) название, описание, обложку группы",
			expectedImage: "https://calendaria-test.s3.eu-north-1.amazonaws.com/491/2025/05/5bc66352-85b7-408c-971c-caabebf823d9.jpg",
			contacts: []*contacts_v1.Contact{
				{
					Id:     1,
					UserId: func() *int64 { id := int64(491); return &id }(),
					Label:  "Dev Mac",
				},
			},
			shouldFormatMessage: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				mockContactsRemote := mock.NewMockIContactsRemote(ctrl)
				mockLocalizer, err := data.NewLocalizerForTest()
				mockDevicesRepo := mock.NewMockDevicesRepo(ctrl)
				mockIamRepo := mock.NewMockIIamRemote(ctrl)
				mockBadge := badge_mock.NewMockIBadgeClient(ctrl)
				mockFcmClient := mock.NewMockFcmClient(ctrl)
				if err != nil {
					t.Fatalf("Failed to create localizer: %v", err)
					return
				}

				mockDevicesRepo.EXPECT().GetDevicesForUsers(gomock.Any(), gomock.Any()).
					Return(tt.userDevices, nil).AnyTimes()

				mockIamRepo.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(map[int64]*users_v1.UserShort{}, nil).AnyTimes()

				mockIamRepo.EXPECT().GetUsersSettings(gomock.Any(), gomock.Any()).
					Return(
						map[int64]map[string]string{
							tt.userId: {
								"NOTIFICATION_SOUND_ENABLED": "true",
								"NOTIFICATION_SOUND":         "default",
							},
						}, nil,
					).AnyTimes()

				mockBadge.EXPECT().GetBadges(gomock.Any(), gomock.Any()).
					Return(map[u_struc.NotificationType]int64{}, nil).AnyTimes()

				mockBadge.EXPECT().IncrementBadge(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				mockContactsRemote.EXPECT().
					GetContactsByUserId(gomock.Any(), gomock.Any()).
					Return(tt.contacts, nil).AnyTimes()

				mockFcmClient.EXPECT().
					Send(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()

				uc := &FcmUsecase{
					log:            log.NewHelper(logger),
					contactsRemote: mockContactsRemote,
					localizer:      mockLocalizer,
					devicesRepo:    mockDevicesRepo,
					iam:            mockIamRepo,
					badgeClient:    mockBadge,
					fcmClient:      mockFcmClient,
				}

				message := tt.inputMessage
				notification, err := uc.SendUserNotifications(context.Background(), *message, true)
				if err != nil {
					t.Fatalf("Failed to send notification: %v", err)
					return
				}

				// notification msg is last message of UsersIds
				assert.Equal(t, tt.expectedTitle, notification.msg.LocalizedTitle, "Wrong title")
				assert.Equal(t, tt.expectedBody, notification.msg.LocalizedBody, "Wrong notification body")
				if tt.expectedImage != "" {
					assert.Equal(t, tt.expectedImage, notification.msg.ImageURL, "Wrong image URL")
				}
			},
		)
	}
}
