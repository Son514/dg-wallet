# dg-wallet

Wallet platform. Currently contains the `auth-service` Go module and its database migrations.

## Requirements

- Go 1.27+
- PostgreSQL
- `make`
- `migrate` CLI (`github.com/golang-migrate/migrate`) with the `postgres` driver

## Setup

```bash
git clone <repo-url> dg-wallet
cd dg-wallet/auth-service
go mod download
```

The `migrate` CLI is required on `PATH`:

```bash
go install github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1
```

## Environment Variables

Read from `auth-service/.env`.

| Name         | Description                        | Required |
|--------------|------------------------------------|----------|
| `DB_HOST`    | PostgreSQL host                    | Yes      |
| `DB_PORT`    | PostgreSQL port                    | Yes      |
| `DB_USER`    | PostgreSQL user                    | Yes      |
| `DB_PASS`    | PostgreSQL password                | Yes      |
| `DB_NAME`    | Database name                      | Yes      |
| `JWT_SECRET` | HS256 signing key for JWTs         | Yes      |
| `GRPC_PORT`  | Port for the gRPC server           | No       |

Generate a real secret with `openssl rand -base64 32`. Anyone holding it can mint
valid tokens, so never commit `.env`.

## Migrations

Migrations live in `auth-service/database/migrations` and are applied with the
`migrate` CLI via the `Makefile` in `auth-service`. The targets load `.env` into
the shell themselves, so no setup is needed beforehand.

```bash
cd auth-service
```

| Command        | Description                       |
|----------------|-----------------------------------|
| `make up`      | Apply all pending migrations      |
| `make down`    | Roll back the last migration      |
| `make down-all`| Roll back all migrations          |
| `make version` | Show current migration version    |

Override the migrations directory with `MIGRATE_PATH`:

```bash
make up MIGRATE_PATH=./database/migrations
```

Each target runs the equivalent command below:

```bash
set -a && source .env && set +a && \
migrate -path database/migrations \
  -database "postgres://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable" up
```

Creating a new migration — add a sequential version pair by hand:

```
database/migrations/000002_add_<table>_table.up.sql
database/migrations/000002_add_<table>_table.down.sql
```

## Schema

| Version | Migration                     | Description              |
|---------|-------------------------------|--------------------------|
| 1       | `000001_create_users_table`   | Creates `users`          |
| 2       | `000002_add_users_id`         | Adds `id` primary key    |

`users` columns:

| Column    | Type          | Constraints              |
|-----------|---------------|--------------------------|
| `email`   | `VARCHAR(255)`| `NOT NULL`, `UNIQUE`     |
| `password`| `VARCHAR(255)`| `NOT NULL`, bcrypt hash  |
| `id`      | `BIGSERIAL`   | `PRIMARY KEY`            |

## Endpoints

| Method | Path      | Description       |
|--------|-----------|-------------------|
| POST   | `/users`  | Create a new user |
| POST   | `/login`  | Log in, returns a JWT |

### POST /users

Request body:

```json
{
  "email": "user@example.com",
  "password": "secret"
}
```

Passwords are hashed with bcrypt (`bcrypt.DefaultCost`) before storage and are
never returned by the API. bcrypt only accepts passwords up to 72 bytes.

| Status | Body                                | When                                              |
|--------|-------------------------------------|---------------------------------------------------|
| `201`  | `{"id":1,"email":"user@example.com"}`| Created; password is never echoed                 |
| `400`  | `{"error":"..."}`                    | Body is not valid JSON, or password exceeds 72 bytes |
| `409`  | `{"error":"email already exists"}`   | Email is already registered                       |
| `500`  | `{"error":"..."}`                    | Insert failed for any other reason                |

### POST /login

Request body:

```json
{
  "email": "user@example.com",
  "password": "secret"
}
```

| Status | Body                             | When                                              |
|--------|----------------------------------|---------------------------------------------------|
| `200`  | `{"token":"eyJhbGciOi..."}`      | Credentials valid; JWT issued                     |
| `400`  | `{"error":"..."}`                | Body is not valid JSON                            |
| `401`  | `{"error":"invalid email or password"}` | Email unknown **or** password wrong        |
| `500`  | `{"error":"..."}`                | Lookup or signing failed for any other reason     |

An unknown email and a wrong password return the same `401` body so the endpoint
cannot be used to discover which addresses are registered.

Tokens are signed with HS256 using `JWT_SECRET` and are valid for 1 hour. Claims:

```json
{
  "email": "user@example.com",
  "sub": "1",
  "iat": 1791101769,
  "exp": 1791105369
}
```

## gRPC

The service exposes `auth.AuthService/ValidateToken` over plaintext gRPC on
`GRPC_PORT` (default `50051`), started by `main` in a goroutine alongside the HTTP
server.

The schema lives in `proto/auth.proto`; generated Go code is written to
`gen/auth/`. Regenerate after editing the proto:

```bash
make proto
```

Run from `auth-service/`. It compiles every `*.proto` in `proto/`, so adding a
schema needs no Makefile change. The underlying command:

```bash
protoc -I proto \
  --go_out=. --go_opt=module=son514/auth-service \
  --go-grpc_out=. --go-grpc_opt=module=son514/auth-service \
  proto/auth.proto
```

Requires `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` on `PATH`.
`make proto` does not delete stale output, so removing a message from a proto
leaves its old `.pb.go` behind — delete it by hand.

### ValidateToken

```bash
grpcurl -plaintext -import-path proto -proto auth.proto \
  -d '{"token":"<jwt>"}' \
  localhost:50051 auth.AuthService/ValidateToken
```

```json
{ "token": "eyJhbGciOiJIUzI1NiIs..." }
```

The call always returns gRPC `OK`; rejection is reported in the response.

```json
{
  "valid": true,
  "userId": "1",
  "email": "user@example.com"
}
```

```json
{
  "valid": false,
  "reason": "token is expired"
}
```

`reason` is one of `token is expired`, `token signature is invalid`,
`token is malformed`, or `token is invalid`.

Note that proto3 omits default values, so a rejected token's response contains no
`valid` field at all rather than `"valid": false`. Treat an absent `valid` as false.

Validation is cryptographic only — signature and expiry. It does not check that
the user still exists in the database.

## Project Structure

```
.
├── README.md
└── auth-service
    ├── .env
    ├── Makefile
    ├── go.mod
    ├── go.sum
    ├── main.go
    ├── database
    │   ├── db
    │   │   └── db.go
    │   └── migrations
    │       ├── 000001_create_users_table.up.sql
    │       ├── 000001_create_users_table.down.sql
    │       ├── 000002_add_users_id.up.sql
    │       └── 000002_add_users_id.down.sql
    ├── gen
    │   └── auth
    │       ├── auth.pb.go
    │       └── auth_grpc.pb.go
    ├── grpc
    │   └── server.go
    ├── jwt
    │   └── jwt.go
    ├── models
    │   └── users.go
    ├── proto
    │   └── auth.proto
    └── routes
        └── routes.go
```