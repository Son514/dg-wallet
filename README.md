# dg-wallet

Wallet platform. Contains three Go modules — `auth-service` (users, JWT, gRPC
token validation), `ledger-service` (ledger entries), and `wallet-service`
(wallets and HTTP/gRPC APIs). Each has its own `.env`, Postgres database, and
migrations.

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

Read from the `.env` in each service's own directory.

| Name         | Description                        | Required |
|--------------|------------------------------------|----------|
| `DB_HOST`    | PostgreSQL host                    | Yes      |
| `DB_PORT`    | PostgreSQL port                    | Yes      |
| `DB_USER`    | PostgreSQL user                    | Yes      |
| `DB_PASS`    | PostgreSQL password                | Yes      |
| `DB_NAME`    | Database name                      | Yes      |
| `JWT_SECRET` | HS256 signing key for JWTs         | auth-service only |
| `GRPC_PORT`  | Port for the gRPC server           | auth-service, ledger-service, optional |
| `WALLET_GRPC_PORT` | wallet-service gRPC port (default `50053`) | wallet-service, optional |
| `AUTH_GRPC_ADDR` | auth-service gRPC address      | wallet-service only, optional |
| `LEDGER_GRPC_ADDR` | ledger-service gRPC address  | wallet-service only, optional |

`auth-service/.env` points at `auth_db`; `wallet-service/.env` points at
`wallet_db`. The two databases are separate, so there are no cross-database
foreign keys between them.

Both services connect at startup via `ConnectDB` and `log.Fatal` if the database
is unreachable, so the `DB_*` values must be correct before running either one.

`AUTH_GRPC_ADDR` defaults to `localhost:50051`, which is where auth-service
listens. wallet-service reads it after `ConnectDB` has loaded `.env`, so the
value only takes effect if the database connection is established first.
`LEDGER_GRPC_ADDR` defaults to `localhost:50052`, which is where ledger-service
listens.
wallet-service serves its own gRPC API on port `50053` by default; override it
with `WALLET_GRPC_PORT`.

Generate a real secret with `openssl rand -base64 32`. Anyone holding it can mint
valid tokens, so never commit `.env`.

## Migrations

Migrations live in `<service>/database/migrations` and are applied with the
`migrate` CLI via the `Makefile` in that service. The targets load `.env` into
the shell themselves, so no setup is needed beforehand.

### auth-service

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

### wallet-service

Identical targets, run from `wallet-service/`. They apply to `wallet_db` rather
than `auth_db`:

```bash
cd wallet-service
```

| Command        | Description                       |
|----------------|-----------------------------------|
| `make up`      | Apply all pending migrations      |
| `make down`    | Roll back the last migration      |
| `make down-all`| Roll back all migrations          |
| `make version` | Show current migration version    |

Each service keeps its own migration history, so both start at version `1`.

## Schema

### auth_db

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

### wallet_db

| Version | Migration                       | Description                 |
|---------|---------------------------------|-----------------------------|
| 1       | `000001_create_wallet_table`    | Creates `wallet_table`      |
| 2       | `000002_add_wallet_table_user_id` | Adds `user_id`, one wallet per user |

`wallet_table` columns:

| Column     | Type           | Constraints                     |
|------------|----------------|---------------------------------|
| `wallet_id`| `BIGSERIAL`    | `PRIMARY KEY`                   |
| `balance`  | `NUMERIC(19,4)`| `NOT NULL`, `DEFAULT 0`         |
| `user_id`  | `BIGINT`       | `NOT NULL`, `UNIQUE`            |

`user_id` holds `users.id` from `auth_db`, which is why it has no foreign key —
the two tables live in separate databases. Deleting a user leaves their wallet
behind. The `UNIQUE` constraint is what enforces one wallet per user, and it is
checked by the database, so concurrent creates cannot both succeed.

`balance` is exact decimal money with four fractional digits. There is no
`CHECK (balance >= 0)`, so negative balances are permitted. The maximum stored
value is `99999999999999.9999`; anything larger overflows and Postgres raises
`numeric field overflow`.

`wallet_id` is a `BIGSERIAL`, so values are guessable by counting up. Read and
top-up endpoints answer `403` when the requested `wallet_id` exists but belongs
to someone else, and `404` only when no such `wallet_id` exists. That is a
deliberate trade-off: clients get an unambiguous "not yours" instead of a
confusing "not found", but it does confirm that an id is taken. `wallet_table` is
also never exposed by id without a valid token.

## Endpoints

### auth-service

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

### wallet-service

| Method | Path                     | Description               |
|--------|--------------------------|---------------------------|
| POST   | `/wallets`               | Create a new wallet       |
| POST   | `/wallets/:id/topup`     | Add funds to a wallet     |
| GET    | `/wallets/:id`           | Check a wallet's balance  |
| GET    | `/wallets/:id/transactions` | Get a wallet's transaction history |

#### POST /wallets

Requires an `Authorization: Bearer <jwt>` header. The token is validated by
calling `auth.AuthService/ValidateToken` over gRPC before the insert, so
auth-service must be running and reachable at `AUTH_GRPC_ADDR`. A missing or
empty bearer token is rejected without making the gRPC call.

Takes no request body. `wallet_id` comes from the `wallet_table_wallet_id_seq`
sequence, `balance` from its column default, and `user_id` from the `sub` claim
of the validated token. One wallet per user: a second request from the same
token is rejected.

| Status | Body                                              | When                                                    |
|--------|---------------------------------------------------|---------------------------------------------------------|
| `201`  | `{"wallet_id":1,"balance":"0.0000"}`              | Token valid and the wallet was created                  |
| `401`  | `{"error":"..."}`                                 | Missing/empty bearer token, or auth-service rejected it |
| `409`  | `{"error":"wallet already exists"}`               | That user already has a wallet                          |
| `503`  | `{"error":"..."}`                                 | The gRPC call itself failed — auth-service unreachable  |
| `500`  | `{"error":"..."}`                                 | Insert failed for any other reason                      |

A `401` body carries the reason auth-service reported, e.g. `token is expired` or
`token is malformed`.

`balance` is returned as a JSON **string**, not a number, because `NUMERIC(19,4)`
holds up to 19 significant digits and a float64 only carries roughly 15-17, so
the decimal text is passed through untouched. Postgres preserves the declared
scale, so the value always has exactly four decimal places. Clients must parse
it before doing arithmetic.

To clear any wallets created while testing, run `make down` then `make up` in
`wallet-service/`; note that migration 2 adds `user_id` as `NOT NULL`, so
dropping back to version 1 is fine but re-applying it requires the table to be
empty.

#### POST /wallets/:id/topup

Adds to an existing wallet's balance. Same `Authorization: Bearer <jwt>`
requirement and same gRPC validation as `POST /wallets`.

Request body:

```json
{
  "amount": "100.00"
}
```

`:id` is the `wallet_id`. Ownership is enforced in the `UPDATE ... WHERE
wallet_id = $2 AND user_id = $3` clause, where `$3` is the `sub` claim of the
validated token, so matching and authorizing happen in one statement with no
window between a check and the write. A `403` therefore means the write never
ran and no money moved.

When the scoped `UPDATE` matches no row, the model looks up the owner of that
`wallet_id` to separate `403` from `404`. That lookup is diagnostic only — it
never gates the write, and the extra query runs only on the failure path, so a
successful top-up still costs one round trip.

`amount` must be greater than zero. The balance is updated with
`balance = balance + $1::numeric`, which is additive and atomic, so concurrent
top-ups cannot overwrite each other.

| Status | Body                                          | When                                              |
|--------|-----------------------------------------------|---------------------------------------------------|
| `200`  | `{"wallet_id":1,"balance":"100.0000"}`         | Top-up applied; `balance` is the new total         |
| `400`  | `{"error":"..."}`                             | `:id` not numeric, body invalid, or `amount <= 0`  |
| `401`  | `{"error":"..."}`                             | Missing/empty bearer token, or auth-service rejected it |
| `403`  | `{"error":"you do not own this wallet"}`      | That wallet belongs to another user                |
| `404`  | `{"error":"wallet not found"}`                | No wallet with that `wallet_id`                   |
| `503`  | `{"error":"..."}`                             | The gRPC call itself failed — auth-service unreachable |
| `500`  | `{"error":"..."}`                             | Update failed for any other reason                 |

`balance` in the response is the **new total**, not the amount added.

Two caveats follow from the column being `NUMERIC(19,4)`:

- An amount with more than four decimal places is **silently rounded** by
  Postgres, so `1.23456` adds `1.2346`. Pass at most four places.
- Because there is still no `CHECK (balance >= 0)`, the database permits
  negative balances. The `amount > 0` check is the only thing preventing a
  withdrawal.

#### GET /wallets/:id

Returns a wallet's current balance. Takes no request body and makes no writes.

Requires an `Authorization: Bearer <jwt>` header, validated over gRPC exactly as
in the other wallet endpoints.

Ownership is enforced with `WHERE wallet_id = $1 AND user_id = $2`, where `$2` is
the `sub` claim of the validated token. If that scoped query matches no row, the
model looks up the owner of that `wallet_id` to separate `403` from `404`. Only
the ownership-scoped `SELECT` decides whether a caller may read a balance; the
owner lookup runs on the failure path only.

| Status | Body                                        | When                                              |
|--------|---------------------------------------------|---------------------------------------------------|
| `200`  | `{"wallet_id":1,"balance":"110.0000"}`      | Caller owns this wallet                            |
| `400`  | `{"error":"wallet id must be a number"}`    | `:id` is not numeric                               |
| `401`  | `{"error":"..."}`                           | Missing/empty bearer token, or auth-service rejected it |
| `403`  | `{"error":"you do not own this wallet"}`    | That wallet belongs to another user                |
| `404`  | `{"error":"wallet not found"}`              | No wallet with that `wallet_id`                    |
| `503`  | `{"error":"..."}`                           | The gRPC call itself failed — auth-service unreachable |
| `500`  | `{"error":"..."}`                           | Query failed for any other reason                  |

`balance` is a string with exactly four decimal places, as on every other
endpoint. Since `wallet_table.user_id` is `UNIQUE`, a user has at most one
wallet, so `:id` is redundant in practice — `GET /wallets` without an id would
address the same single wallet. The id is kept in the path and echoed back.

#### GET /wallets/:id/transactions

Returns the wallet's ledger entries, newest first. Requires an
`Authorization: ****** header. The token is validated over gRPC, then wallet
ownership is checked against `wallet_db` before requesting history from
ledger-service. No request body is needed.

Each entry includes its ID, amount, type, and creation time:

```json
{
  "wallet_id": 1,
  "entries": [
    {
      "entry_id": 42,
      "amount": "25.0000",
      "type": "topup",
      "created_at": "2026-10-07T00:00:00Z"
    }
  ]
}
```

| Status | Body                                          | When                                              |
|--------|-----------------------------------------------|---------------------------------------------------|
| `200`  | `{"wallet_id":1,"entries":[...]}`             | Caller owns the wallet; entries may be empty      |
| `400`  | `{"error":"wallet id must be a number"}`      | `:id` is not numeric                               |
| `401`  | `{"error":"..."}`                             | Missing/empty bearer token, or auth-service rejected it |
| `403`  | `{"error":"you do not own this wallet"}`      | That wallet belongs to another user                |
| `404`  | `{"error":"wallet not found"}`                | No wallet with that `wallet_id`                    |
| `503`  | `{"error":"..."}`                             | An auth-service or ledger-service gRPC call failed |
| `500`  | `{"error":"..."}`                             | Wallet ownership query failed for another reason  |

## gRPC

### auth-service server

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

#### ValidateToken

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

### wallet-service client

wallet-service consumes the same RPC. The dial and the call wrapper live in
`grpc/client.go`; `main` dials `AUTH_GRPC_ADDR` once and shares the resulting
client with the handlers.

`NewClient` uses `grpc.NewClient` with insecure credentials, matching auth-service's
plaintext listener. It is lazy — it does not connect on startup, so auth-service
being down does not stop wallet-service from booting. The first RPC fails instead,
and `ValidateToken` wraps each call in a 3s context timeout so a hung
auth-service cannot hang the HTTP request.

The generated client in `gen/auth/` is a **separate copy** from auth-service's,
produced from `../auth-service/proto/auth.proto`. The Makefile also generates
the ledger and wallet bindings:

```bash
make proto
```

Run from `wallet-service/`. The `Mauth.proto` and `Mledger.proto` flags override
their `go_package` paths so generated code is written into wallet-service's
local copies. The wallet schema declares its wallet-service package directly.
Regenerate here as well as in auth-service whenever the auth schema changes, or
wallet-service compiles against a stale contract.

### wallet-service server

wallet-service also serves `wallet.WalletService/CreateWallet` over plaintext
gRPC, alongside its HTTP API. The listener defaults to port `50053`, configurable
with `WALLET_GRPC_PORT`.

The request carries the same bearer JWT used by the HTTP API:

```bash
grpcurl -plaintext \
  -import-path wallet-service/proto -proto wallet.proto \
  -d '{"token":"<jwt>"}' \
  localhost:50053 wallet.WalletService/CreateWallet
```

The server validates the token through auth-service before creating the wallet.
It returns `wallet_id` and `balance`; invalid tokens return `UNAUTHENTICATED`,
an existing wallet returns `ALREADY_EXISTS`, auth-service call failures return
`UNAVAILABLE`, and database failures return `INTERNAL`.

The schema is `wallet-service/proto/wallet.proto` and generated code is in
`wallet-service/gen/wallet/`. Run `make proto` from `wallet-service/` to
regenerate the auth, ledger, and wallet bindings.

## Project Structure

```
.
├── README.md
├── auth-service
│   ├── .env
│   ├── Makefile
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── database
│   │   ├── db
│   │   │   └── db.go
│   │   └── migrations
│   │       ├── 000001_create_users_table.up.sql
│   │       ├── 000001_create_users_table.down.sql
│   │       ├── 000002_add_users_id.up.sql
│   │       └── 000002_add_users_id.down.sql
│   ├── gen
│   │   └── auth
│   │       ├── auth.pb.go
│   │       └── auth_grpc.pb.go
│   ├── grpc
│   │   └── server.go
│   ├── jwt
│   │   └── jwt.go
│   ├── models
│   │   └── users.go
│   ├── proto
│   │   └── auth.proto
│   └── routes
│       └── routes.go
└── wallet-service
    ├── .env
    ├── Makefile
    ├── go.mod
    ├── go.sum
    ├── main.go
    ├── database
    │   ├── db.go
    │   └── migrations
    │       ├── 000001_create_wallet_table.up.sql
    │       ├── 000001_create_wallet_table.down.sql
    │       ├── 000002_add_wallet_table_user_id.up.sql
    │       └── 000002_add_wallet_table_user_id.down.sql
    ├── gen
    │   ├── auth
    │       ├── auth.pb.go
    │       └── auth_grpc.pb.go
    │   ├── ledger
    │   │   ├── ledger.pb.go
    │   │   └── ledger_grpc.pb.go
    │   └── wallet
    │       ├── wallet.pb.go
    │       └── wallet_grpc.pb.go
    ├── grpc
    │   ├── client.go
    │   └── server.go
    ├── proto
    │   ├── ledger.proto
    │   └── wallet.proto
    └── models
        └── wallet.go
```