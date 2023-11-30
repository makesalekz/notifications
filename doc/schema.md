## Device:

|   Field    |   Type    | Unique | Optional | Nillable | Default | UpdateDefault | Immutable |          StructTag          | Validators | Comment |
|---|---|---|---|---|---|---|---|---|---|---|
| id         | int       | false  | false    | false    | false   | false         | false     | json:"id,omitempty"         |          0 |         |
| user_id    | int64     | false  | false    | false    | false   | false         | false     | json:"user_id,omitempty"    |          1 |         |
| token      | string    | true   | false    | false    | false   | false         | false     | json:"token,omitempty"      |          0 |         |
| created_at | time.Time | false  | false    | false    | true    | false         | false     | json:"created_at,omitempty" |          0 |         |

## LastReadNotification:

|    Field     |         Type          | Unique | Optional | Nillable | Default | UpdateDefault | Immutable |           StructTag           | Validators | Comment |
|---|---|---|---|---|---|---|---|---|---|---|
| id           | int                   | false  | false    | false    | false   | false         | false     | json:"id,omitempty"           |          0 |         |
| user_id      | int64                 | false  | false    | false    | false   | false         | true      | json:"user_id,omitempty"      |          0 |         |
| type         | enum.NotificationType | false  | false    | false    | true    | false         | false     | json:"type,omitempty"         |          0 |         |
| last_read_id | int64                 | false  | false    | false    | false   | false         | false     | json:"last_read_id,omitempty" |          0 |         |

## Notification:

|   Field    |         Type          | Unique | Optional | Nillable | Default | UpdateDefault | Immutable |          StructTag          | Validators | Comment |
|---|---|---|---|---|---|---|---|---|---|---|
| id         | int                   | false  | false    | false    | false   | false         | false     | json:"id,omitempty"         |          0 |         |
| user_id    | int64                 | false  | false    | false    | false   | false         | false     | json:"user_id,omitempty"    |          1 |         |
| type       | enum.NotificationType | false  | false    | false    | true    | false         | false     | json:"type,omitempty"       |          0 |         |
| title      | string                | false  | false    | false    | false   | false         | false     | json:"title,omitempty"      |          1 |         |
| text       | string                | false  | false    | false    | false   | false         | false     | json:"text,omitempty"       |          1 |         |
| event_id   | int64                 | false  | true     | true     | false   | false         | false     | json:"event_id,omitempty"   |          0 |         |
| contact_id | int64                 | false  | true     | true     | false   | false         | false     | json:"contact_id,omitempty" |          0 |         |
| created_at | time.Time             | false  | false    | false    | true    | false         | false     | json:"created_at,omitempty" |          0 |         |

