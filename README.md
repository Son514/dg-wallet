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

| Name      | Description       | Required |
|-----------|-------------------|----------|
| `DB_HOST` | PostgreSQL host   | Yes      |
| `DB_PORT` | PostgreSQL port   | Yes      |
| `DB_USER` | PostgreSQL user   | Yes      |
| `DB_PASS` | PostgreSQL password | Yes    |
| `DB_NAME` | Database name     | Yes      |

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

| Method | Path     | Description       |
|--------|----------|-------------------|
| POST   | `/users` | Create a new user |

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
    └── models
        └── users.go
```