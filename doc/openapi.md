<!-- Generator: Widdershins v4.0.1 -->

<h1 id="api"> v0.0.1</h1>

> Scroll down for example requests and responses.

<h1 id="api-notifications">Notifications</h1>

## Notifications_GetNotificationsCounters

<a id="opIdNotifications_GetNotificationsCounters"></a>

`GET /v1/notifications/counters`

> Example responses

> 200 Response

```json
{
  "unreadCount": {
    "property1": 0,
    "property2": 0
  }
}
```

<h3 id="notifications_getnotificationscounters-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[notifications.v1.NotificationCountersReply](#schemanotifications.v1.notificationcountersreply)|

<aside class="success">
This operation does not require authentication
</aside>

## Notifications_ListNotifications

<a id="opIdNotifications_ListNotifications"></a>

`POST /v1/notifications/list`

> Body parameter

```json
{
  "type": "string",
  "paginate": {
    "limit": 0,
    "fromId": "string",
    "toId": "string",
    "aroundId": "string",
    "fromDate": "string",
    "toDate": "string",
    "aroundDate": "string",
    "fromLabel": "string",
    "descending": true
  }
}
```

<h3 id="notifications_listnotifications-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[notifications.v1.ListNotificationsRequest](#schemanotifications.v1.listnotificationsrequest)|true|none|

> Example responses

> 200 Response

```json
{
  "notifications": [
    {
      "id": "string",
      "type": "string",
      "title": "string",
      "text": "string",
      "eventId": "string",
      "contactId": "string",
      "createdAt": "string"
    }
  ],
  "contacts": [
    {
      "id": "string",
      "phone": "string",
      "label": "string",
      "updatedAt": "string",
      "email": "string",
      "user": {
        "id": "string",
        "phone": "string",
        "email": "string",
        "name": "string",
        "avatar": "string",
        "lastLoginAt": "string"
      },
      "createdAt": "string"
    }
  ],
  "events": [
    {
      "id": "string",
      "title": "string",
      "description": "string",
      "coverUrl": "string",
      "startDateTime": "string",
      "endDateTime": "string",
      "isAllDay": true,
      "noticeBefore": 0,
      "chatId": "string",
      "locationId": "string",
      "type": "string",
      "recurrenceRule": "string",
      "avatars": [
        "string"
      ],
      "membersCount": 0,
      "owner": {
        "id": "string",
        "phone": "string",
        "email": "string",
        "name": "string",
        "avatar": "string",
        "lastLoginAt": "string"
      },
      "membership": {
        "status": "string",
        "role": "string",
        "remind": 0,
        "remindAt": "string"
      },
      "location": {
        "address": "string",
        "latitude": 0,
        "longitude": 0
      }
    }
  ],
  "paginate": {
    "total": 0,
    "fromId": "string",
    "toId": "string",
    "fromDate": "string",
    "toDate": "string",
    "fromLabel": "string"
  }
}
```

<h3 id="notifications_listnotifications-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[notifications.v1.ListNotificationsReply](#schemanotifications.v1.listnotificationsreply)|

<aside class="success">
This operation does not require authentication
</aside>

## Notifications_DoActionOnNotification

<a id="opIdNotifications_DoActionOnNotification"></a>

`POST /v1/notifications/{notificationId}/action`

> Body parameter

```json
{
  "notificationId": "string",
  "action": "string",
  "type": "string"
}
```

<h3 id="notifications_doactiononnotification-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|notificationId|path|string|true|none|
|body|body|[notifications.v1.DoActionOnNotificationRequest](#schemanotifications.v1.doactiononnotificationrequest)|true|none|

> Example responses

> 200 Response

```json
{}
```

<h3 id="notifications_doactiononnotification-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[utils.v1.EmptyReply](#schemautils.v1.emptyreply)|

<aside class="success">
This operation does not require authentication
</aside>

<h1 id="api-sender">Sender</h1>

## Sender_CreateFcmDevice

<a id="opIdSender_CreateFcmDevice"></a>

`POST /v1/notifications/fcm/devices`

> Body parameter

```json
{
  "token": "string"
}
```

<h3 id="sender_createfcmdevice-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[notifications.v1.FcmDeviceRequest](#schemanotifications.v1.fcmdevicerequest)|true|none|

> Example responses

> 200 Response

```json
{}
```

<h3 id="sender_createfcmdevice-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[utils.v1.EmptyReply](#schemautils.v1.emptyreply)|

<aside class="success">
This operation does not require authentication
</aside>

## Sender_DeleteFcmDevice

<a id="opIdSender_DeleteFcmDevice"></a>

`DELETE /v1/notifications/fcm/devices`

<h3 id="sender_deletefcmdevice-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|token|query|string|false|none|

> Example responses

> 200 Response

```json
{}
```

<h3 id="sender_deletefcmdevice-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|[utils.v1.EmptyReply](#schemautils.v1.emptyreply)|

<aside class="success">
This operation does not require authentication
</aside>

# Schemas

<h2 id="tocS_contacts.v1.Contact">contacts.v1.Contact</h2>
<!-- backwards compatibility -->
<a id="schemacontacts.v1.contact"></a>
<a id="schema_contacts.v1.Contact"></a>
<a id="tocScontacts.v1.contact"></a>
<a id="tocscontacts.v1.contact"></a>

```json
{
  "id": "string",
  "phone": "string",
  "label": "string",
  "updatedAt": "string",
  "email": "string",
  "user": {
    "id": "string",
    "phone": "string",
    "email": "string",
    "name": "string",
    "avatar": "string",
    "lastLoginAt": "string"
  },
  "createdAt": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|false|none|none|
|phone|string|false|none|none|
|label|string|false|none|none|
|updatedAt|string|false|none|none|
|email|string|false|none|none|
|user|[iam.v1.UserShort](#schemaiam.v1.usershort)|false|none|none|
|createdAt|string|false|none|none|

<h2 id="tocS_events.v1.Event">events.v1.Event</h2>
<!-- backwards compatibility -->
<a id="schemaevents.v1.event"></a>
<a id="schema_events.v1.Event"></a>
<a id="tocSevents.v1.event"></a>
<a id="tocsevents.v1.event"></a>

```json
{
  "id": "string",
  "title": "string",
  "description": "string",
  "coverUrl": "string",
  "startDateTime": "string",
  "endDateTime": "string",
  "isAllDay": true,
  "noticeBefore": 0,
  "chatId": "string",
  "locationId": "string",
  "type": "string",
  "recurrenceRule": "string",
  "avatars": [
    "string"
  ],
  "membersCount": 0,
  "owner": {
    "id": "string",
    "phone": "string",
    "email": "string",
    "name": "string",
    "avatar": "string",
    "lastLoginAt": "string"
  },
  "membership": {
    "status": "string",
    "role": "string",
    "remind": 0,
    "remindAt": "string"
  },
  "location": {
    "address": "string",
    "latitude": 0,
    "longitude": 0
  }
}

```

reply

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|false|none|none|
|title|string|false|none|none|
|description|string|false|none|none|
|coverUrl|string|false|none|none|
|startDateTime|string|false|none|none|
|endDateTime|string|false|none|none|
|isAllDay|boolean|false|none|none|
|noticeBefore|integer(int32)|false|none|none|
|chatId|string|false|none|none|
|locationId|string|false|none|none|
|type|string|false|none|none|
|recurrenceRule|string|false|none|none|
|avatars|[string]|false|none|none|
|membersCount|integer(int32)|false|none|none|
|owner|[iam.v1.UserShort](#schemaiam.v1.usershort)|false|none|none|
|membership|[events.v1.Membership](#schemaevents.v1.membership)|false|none|none|
|location|[events.v1.LocationDto](#schemaevents.v1.locationdto)|false|none|none|

<h2 id="tocS_events.v1.LocationDto">events.v1.LocationDto</h2>
<!-- backwards compatibility -->
<a id="schemaevents.v1.locationdto"></a>
<a id="schema_events.v1.LocationDto"></a>
<a id="tocSevents.v1.locationdto"></a>
<a id="tocsevents.v1.locationdto"></a>

```json
{
  "address": "string",
  "latitude": 0,
  "longitude": 0
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|address|string|false|none|none|
|latitude|number(float)|false|none|none|
|longitude|number(float)|false|none|none|

<h2 id="tocS_events.v1.Membership">events.v1.Membership</h2>
<!-- backwards compatibility -->
<a id="schemaevents.v1.membership"></a>
<a id="schema_events.v1.Membership"></a>
<a id="tocSevents.v1.membership"></a>
<a id="tocsevents.v1.membership"></a>

```json
{
  "status": "string",
  "role": "string",
  "remind": 0,
  "remindAt": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|status|string|false|none|none|
|role|string|false|none|none|
|remind|integer(int32)|false|none|none|
|remindAt|string|false|none|none|

<h2 id="tocS_iam.v1.UserShort">iam.v1.UserShort</h2>
<!-- backwards compatibility -->
<a id="schemaiam.v1.usershort"></a>
<a id="schema_iam.v1.UserShort"></a>
<a id="tocSiam.v1.usershort"></a>
<a id="tocsiam.v1.usershort"></a>

```json
{
  "id": "string",
  "phone": "string",
  "email": "string",
  "name": "string",
  "avatar": "string",
  "lastLoginAt": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|false|none|none|
|phone|string|false|none|none|
|email|string|false|none|none|
|name|string|false|none|none|
|avatar|string|false|none|none|
|lastLoginAt|string|false|none|none|

<h2 id="tocS_notifications.v1.DoActionOnNotificationRequest">notifications.v1.DoActionOnNotificationRequest</h2>
<!-- backwards compatibility -->
<a id="schemanotifications.v1.doactiononnotificationrequest"></a>
<a id="schema_notifications.v1.DoActionOnNotificationRequest"></a>
<a id="tocSnotifications.v1.doactiononnotificationrequest"></a>
<a id="tocsnotifications.v1.doactiononnotificationrequest"></a>

```json
{
  "notificationId": "string",
  "action": "string",
  "type": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|notificationId|string|false|none|none|
|action|string|false|none|none|
|type|string|false|none|none|

<h2 id="tocS_notifications.v1.FcmDeviceRequest">notifications.v1.FcmDeviceRequest</h2>
<!-- backwards compatibility -->
<a id="schemanotifications.v1.fcmdevicerequest"></a>
<a id="schema_notifications.v1.FcmDeviceRequest"></a>
<a id="tocSnotifications.v1.fcmdevicerequest"></a>
<a id="tocsnotifications.v1.fcmdevicerequest"></a>

```json
{
  "token": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|token|string|false|none|none|

<h2 id="tocS_notifications.v1.ListNotificationsReply">notifications.v1.ListNotificationsReply</h2>
<!-- backwards compatibility -->
<a id="schemanotifications.v1.listnotificationsreply"></a>
<a id="schema_notifications.v1.ListNotificationsReply"></a>
<a id="tocSnotifications.v1.listnotificationsreply"></a>
<a id="tocsnotifications.v1.listnotificationsreply"></a>

```json
{
  "notifications": [
    {
      "id": "string",
      "type": "string",
      "title": "string",
      "text": "string",
      "eventId": "string",
      "contactId": "string",
      "createdAt": "string"
    }
  ],
  "contacts": [
    {
      "id": "string",
      "phone": "string",
      "label": "string",
      "updatedAt": "string",
      "email": "string",
      "user": {
        "id": "string",
        "phone": "string",
        "email": "string",
        "name": "string",
        "avatar": "string",
        "lastLoginAt": "string"
      },
      "createdAt": "string"
    }
  ],
  "events": [
    {
      "id": "string",
      "title": "string",
      "description": "string",
      "coverUrl": "string",
      "startDateTime": "string",
      "endDateTime": "string",
      "isAllDay": true,
      "noticeBefore": 0,
      "chatId": "string",
      "locationId": "string",
      "type": "string",
      "recurrenceRule": "string",
      "avatars": [
        "string"
      ],
      "membersCount": 0,
      "owner": {
        "id": "string",
        "phone": "string",
        "email": "string",
        "name": "string",
        "avatar": "string",
        "lastLoginAt": "string"
      },
      "membership": {
        "status": "string",
        "role": "string",
        "remind": 0,
        "remindAt": "string"
      },
      "location": {
        "address": "string",
        "latitude": 0,
        "longitude": 0
      }
    }
  ],
  "paginate": {
    "total": 0,
    "fromId": "string",
    "toId": "string",
    "fromDate": "string",
    "toDate": "string",
    "fromLabel": "string"
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|notifications|[[notifications.v1.Notification](#schemanotifications.v1.notification)]|false|none|none|
|contacts|[[contacts.v1.Contact](#schemacontacts.v1.contact)]|false|none|none|
|events|[[events.v1.Event](#schemaevents.v1.event)]|false|none|[reply]|
|paginate|[utils.v1.PaginateReply](#schemautils.v1.paginatereply)|false|none|none|

<h2 id="tocS_notifications.v1.ListNotificationsRequest">notifications.v1.ListNotificationsRequest</h2>
<!-- backwards compatibility -->
<a id="schemanotifications.v1.listnotificationsrequest"></a>
<a id="schema_notifications.v1.ListNotificationsRequest"></a>
<a id="tocSnotifications.v1.listnotificationsrequest"></a>
<a id="tocsnotifications.v1.listnotificationsrequest"></a>

```json
{
  "type": "string",
  "paginate": {
    "limit": 0,
    "fromId": "string",
    "toId": "string",
    "aroundId": "string",
    "fromDate": "string",
    "toDate": "string",
    "aroundDate": "string",
    "fromLabel": "string",
    "descending": true
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|type|string|false|none|none|
|paginate|[utils.v1.PaginateRequest](#schemautils.v1.paginaterequest)|false|none|none|

<h2 id="tocS_notifications.v1.Notification">notifications.v1.Notification</h2>
<!-- backwards compatibility -->
<a id="schemanotifications.v1.notification"></a>
<a id="schema_notifications.v1.Notification"></a>
<a id="tocSnotifications.v1.notification"></a>
<a id="tocsnotifications.v1.notification"></a>

```json
{
  "id": "string",
  "type": "string",
  "title": "string",
  "text": "string",
  "eventId": "string",
  "contactId": "string",
  "createdAt": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|false|none|none|
|type|string|false|none|none|
|title|string|false|none|none|
|text|string|false|none|none|
|eventId|string|false|none|none|
|contactId|string|false|none|none|
|createdAt|string|false|none|none|

<h2 id="tocS_notifications.v1.NotificationCountersReply">notifications.v1.NotificationCountersReply</h2>
<!-- backwards compatibility -->
<a id="schemanotifications.v1.notificationcountersreply"></a>
<a id="schema_notifications.v1.NotificationCountersReply"></a>
<a id="tocSnotifications.v1.notificationcountersreply"></a>
<a id="tocsnotifications.v1.notificationcountersreply"></a>

```json
{
  "unreadCount": {
    "property1": 0,
    "property2": 0
  }
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|unreadCount|object|false|none|none|
|» **additionalProperties**|integer(int32)|false|none|none|

<h2 id="tocS_utils.v1.EmptyReply">utils.v1.EmptyReply</h2>
<!-- backwards compatibility -->
<a id="schemautils.v1.emptyreply"></a>
<a id="schema_utils.v1.EmptyReply"></a>
<a id="tocSutils.v1.emptyreply"></a>
<a id="tocsutils.v1.emptyreply"></a>

```json
{}

```

### Properties

*None*

<h2 id="tocS_utils.v1.PaginateReply">utils.v1.PaginateReply</h2>
<!-- backwards compatibility -->
<a id="schemautils.v1.paginatereply"></a>
<a id="schema_utils.v1.PaginateReply"></a>
<a id="tocSutils.v1.paginatereply"></a>
<a id="tocsutils.v1.paginatereply"></a>

```json
{
  "total": 0,
  "fromId": "string",
  "toId": "string",
  "fromDate": "string",
  "toDate": "string",
  "fromLabel": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|total|integer(int32)|false|none|none|
|fromId|string|false|none|none|
|toId|string|false|none|none|
|fromDate|string|false|none|none|
|toDate|string|false|none|none|
|fromLabel|string|false|none|none|

<h2 id="tocS_utils.v1.PaginateRequest">utils.v1.PaginateRequest</h2>
<!-- backwards compatibility -->
<a id="schemautils.v1.paginaterequest"></a>
<a id="schema_utils.v1.PaginateRequest"></a>
<a id="tocSutils.v1.paginaterequest"></a>
<a id="tocsutils.v1.paginaterequest"></a>

```json
{
  "limit": 0,
  "fromId": "string",
  "toId": "string",
  "aroundId": "string",
  "fromDate": "string",
  "toDate": "string",
  "aroundDate": "string",
  "fromLabel": "string",
  "descending": true
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|limit|integer(int32)|false|none|none|
|fromId|string|false|none|none|
|toId|string|false|none|none|
|aroundId|string|false|none|none|
|fromDate|string|false|none|none|
|toDate|string|false|none|none|
|aroundDate|string|false|none|none|
|fromLabel|string|false|none|none|
|descending|boolean|false|none|none|

