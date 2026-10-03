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

| Version | Migration                     | Description        |
|---------|-------------------------------|--------------------|
| 1       | `000001_create_users_table`   | Creates `users`    |

`users` columns:

| Column    | Type          | Constraints     |
|-----------|---------------|-----------------|
| `email`   | `VARCHAR(255)`| `NOT NULL`, `UNIQUE` |
| `password`| `VARCHAR(255)`| `NOT NULL`       |

## Endpoints

None yet.

## Project Structure

```
.
├── README.md
└── auth-service
    ├── .env
    ├── Makefile
    ├── go.mod
    ├── go.sum
    └── database
        └── migrations
            ├── 000001_create_users_table.up.sql
            └── 000001_create_users_table.down.sql
```