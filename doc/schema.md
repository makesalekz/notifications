## Device:

|   Field    |   Type    | Unique | Optional | Nillable | Default | UpdateDefault | Immutable |          StructTag          | Validators | Comment |
|---|---|---|---|---|---|---|---|---|---|---|
| id         | int       | false  | false    | false    | false   | false         | false     | json:"id,omitempty"         |          0 |         |
| user_id    | int64     | false  | false    | false    | false   | false         | false     | json:"user_id,omitempty"    |          1 |         |
| token      | string    | true   | false    | false    | false   | false         | true      | json:"token,omitempty"      |          1 |         |
| created_at | time.Time | false  | false    | false    | true    | false         | true      | json:"created_at,omitempty" |          0 |         |
| language   | string    | false  | true     | false    | true    | false         | false     | json:"language,omitempty"   |          0 |         |

## LastReadNotification:

|    Field     |         Type          | Unique | Optional | Nillable | Default | UpdateDefault | Immutable |           StructTag           | Validators | Comment |
|---|---|---|---|---|---|---|---|---|---|---|
| id           | int                   | false  | false    | false    | false   | false         | false     | json:"id,omitempty"           |          0 |         |
| user_id      | int64                 | false  | false    | false    | false   | false         | true      | json:"user_id,omitempty"      |          0 |         |
| type         | u_struc.NotificationType | false  | false    | false    | true    | false         | false     | json:"type,omitempty"         |          0 |         |
| last_read_id | int64                 | false  | false    | false    | false   | false         | false     | json:"last_read_id,omitempty" |          0 |         |

## Notification:

|        Field         |         Type          | Unique | Optional | Nillable | Default | UpdateDefault | Immutable |               StructTag               | Validators | Comment |
|---|---|---|---|---|---|---|---|---|---|---|
| id                   | int                   | false  | false    | false    | false   | false         | false     | json:"id,omitempty"                   |          0 |         |
| user_id              | int64                 | false  | false    | false    | false   | false         | false     | json:"user_id,omitempty"              |          1 |         |
| type                 | u_struc.NotificationType | false  | false    | false    | true    | false         | false     | json:"type,omitempty"                 |          0 |         |
| title                | string                | false  | false    | false    | false   | false         | false     | json:"title,omitempty"                |          1 |         |
| text                 | string                | false  | false    | false    | false   | false         | false     | json:"text,omitempty"                 |          1 |         |
| event_id             | int64                 | false  | true     | true     | false   | false         | false     | json:"event_id,omitempty"             |          0 |         |
| contact_id           | int64                 | false  | true     | true     | false   | false         | false     | json:"contact_id,omitempty"           |          0 |         |
| task_id              | int64                 | false  | true     | true     | false   | false         | false     | json:"task_id,omitempty"              |          0 |         |
| created_at           | time.Time             | false  | false    | false    | true    | false         | false     | json:"created_at,omitempty"           |          0 |         |
| notification_data_id | int64                 | false  | true     | true     | false   | false         | false     | json:"notification_data_id,omitempty" |          0 |         |


|       Edge        |       Type       | Inverse |   BackRef    | Relation | Unique | Optional | Comment |
|---|---|---|---|---|---|---|---|
| notification_data | NotificationData | true    | notification | O2O      | true   | true     |         |

## NotificationData:

|    Field     |  Type  | Unique | Optional | Nillable | Default | UpdateDefault | Immutable |           StructTag           | Validators | Comment |
|---|---|---|---|---|---|---|---|---|---|---|
| id           | int64  | false  | false    | false    | false   | false         | false     | json:"id,omitempty"           |          0 |         |
| type         | string | false  | true     | true     | false   | false         | false     | json:"type,omitempty"         |          0 |         |
| event        | string | false  | true     | true     | false   | false         | false     | json:"event,omitempty"        |          0 |         |
| member       | string | false  | true     | true     | false   | false         | false     | json:"member,omitempty"       |          0 |         |
| chat         | string | false  | true     | true     | false   | false         | false     | json:"chat,omitempty"         |          0 |         |
| message      | string | false  | true     | true     | false   | false         | false     | json:"message,omitempty"      |          0 |         |
| contact      | string | false  | true     | true     | false   | false         | false     | json:"contact,omitempty"      |          0 |         |
| task         | string | false  | true     | true     | false   | false         | false     | json:"task,omitempty"         |          0 |         |
| metadata     | string | false  | true     | true     | false   | false         | false     | json:"metadata,omitempty"     |          0 |         |
| plural_count | int64  | false  | true     | true     | false   | false         | false     | json:"plural_count,omitempty" |          0 |         |


|     Edge     |     Type     | Inverse | BackRef | Relation | Unique | Optional | Comment |
|---|---|---|---|---|---|---|---|
| notification | Notification | false   |         | O2O      | true   | true     |         |

