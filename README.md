# notifications

## Proto files

### Add a proto template

```bash
kratos proto add api/server/server.proto
```

### Generate the proto code

```bash
kratos proto client api/server/server.proto
```

### Generate the source code of service by proto file

```bash
kratos proto server api/server/server.proto -t internal/service

go generate ./...
```

## Generate other auxiliary files by Makefile

### Download and update dependencies

```bash
make init
```

### Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file

```bash
make api
```

### Generate all files

```bash
make all
```

### Generate migrations

[Install Atlas](https://entgo.io/docs/versioned-migrations#generating-migrations)

```bash
make migrations
```

## Configuration

Add credentials.json, firebase.json to configs/

### Consul

```bash
app/notifications/AWS_REGION
app/notifications/SMSC_ENDPOINT = <URL: string>
app/notifications/SES_SOURCE_EMAIL = <info@calendaria.team>
```

### Vault

To save JWT secret in Vault terminal (write command, ENTER, paste secret, CTRL+D):

```bash
export VAULT_TOKEN=myroot
vault kv put -mount=secret app/global/jwt data=-
vault kv put -mount=secret app/notifications/aws access_key_id=<key id> secret_access_key=<secret>
vault kv put -mount=secret app/notifications/smsc login=<login> password=<password>
```

```txt
secret/data/app/notifications/smsc = {
    login: string,
    password: string
}
secret/data/app/notifications/gserviceaccount = {
    data: string
}
secret/data/app/notifications/firebase = {
    data: string
}
secret/data/app/notifications/aws = {
    access_key_id: string,
    secret_access_key: string
}
```

## Run

### Run locally

```bash
make run
```

### Run in Docker

```bash
make start
```

### Stop docker

```bash
make stop
```
