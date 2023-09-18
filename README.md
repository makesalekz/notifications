# LDAP

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

## Run

Add credentials.json, firebase.json to configs/

### Run debug

```bash
make run
```

### Build & Run

```bash
export GOOGLE_APPLICATION_CREDENTIALS={path-to-credentials.json}
export FIREBASE_CONFIG={path-to-firebase.json}
export JWT_SECRET={JWT_SECRET}

go build -o ./bin/ ./...
./bin/media -conf ./configs
```

## Run in Docker

```bash
docker compose up -d
```

## Configuration

### Consul

```txt
app/notifications/SMSC_ENDPOINT = <URL: string>
```

### Vault

TODO: DB creds, JWT token, GOOGLE_APPLICATION_CREDENTIALS, FIREBASE_CONFIG

```txt
secret/data/app/notifications/smsc = {
    login: string,
    password: string
}
```
