# DigiVault — Product Specification

## What is DigiVault?
Production-grade digital wallet API in Go.
5 microservices, clean architecture, gRPC inter-service communication, JWT auth with refresh rotation,
Redis, MySQL, GCP Pub/Sub, Elasticsearch, Docker, Kubernetes-ready.

## Completion Definition
The project is COMPLETE when:
- All 5 services build and run end-to-end
- All services pass coverage targets (CLAUDE.md)
- Docker Compose brings up entire stack with `make run`
- All gRPC contracts defined in `.proto` files and generated
- README documents every API, auth flow, and architecture decision
- No hardcoded secrets — everything via Viper + .env
- `make test` passes across all services with zero failures

---

## Architecture Overview

```
Client
  │
  ▼
api-gateway :8080          ← JWT validation, rate limiting, routing
  │
  ├──gRPC──► auth-service :8081        ← register, login, OTP, token rotation
  ├──gRPC──► wallet-service :8082      ← balance, credit, debit
  ├──gRPC──► txn-service :8083         ← transfers, history
  └──gRPC──► notification-service :8084 (internal only)
                │
                └── Pub/Sub subscriber ← txn-service publishes events here

Shared infra:
  MySQL      ← auth, wallet, txn, notification (separate DBs per service)
  Redis      ← OTP store, refresh token store, rate limiter
  Elasticsearch ← transaction search + history
  GCP Pub/Sub   ← async event bus between txn and notification
```

---

## Database Schema

### auth-service DB: `dv_auth`

```sql
CREATE TABLE users (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  phone        VARCHAR(15)  NOT NULL UNIQUE,
  email        VARCHAR(255) NOT NULL UNIQUE,
  password     VARCHAR(255) NOT NULL,           -- bcrypt hashed
  is_active    BOOLEAN      NOT NULL DEFAULT FALSE,
  created_at   DATETIME(3)  NOT NULL,
  updated_at   DATETIME(3)  NOT NULL,
  deleted_at   DATETIME(3)  DEFAULT NULL,
  INDEX idx_users_phone (phone),
  INDEX idx_users_email (email),
  INDEX idx_users_deleted_at (deleted_at)
);
```

No refresh token table — refresh tokens live in Redis only (jti as key).

### wallet-service DB: `dv_wallet`

```sql
CREATE TABLE wallets (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id      BIGINT UNSIGNED NOT NULL UNIQUE,
  balance      DECIMAL(15,2)   NOT NULL DEFAULT 0.00,
  currency     VARCHAR(3)      NOT NULL DEFAULT 'INR',
  is_locked    BOOLEAN         NOT NULL DEFAULT FALSE,
  created_at   DATETIME(3)     NOT NULL,
  updated_at   DATETIME(3)     NOT NULL,
  deleted_at   DATETIME(3)     DEFAULT NULL,
  INDEX idx_wallets_user_id (user_id),
  INDEX idx_wallets_deleted_at (deleted_at)
);
```

### txn-service DB: `dv_txn`

```sql
CREATE TABLE transactions (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  reference    VARCHAR(36)     NOT NULL UNIQUE,  -- UUID
  sender_id    BIGINT UNSIGNED NOT NULL,
  receiver_id  BIGINT UNSIGNED NOT NULL,
  amount       DECIMAL(15,2)   NOT NULL,
  currency     VARCHAR(3)      NOT NULL DEFAULT 'INR',
  status       ENUM('PENDING','SUCCESS','FAILED') NOT NULL DEFAULT 'PENDING',
  failure_reason VARCHAR(255)  DEFAULT NULL,
  created_at   DATETIME(3)     NOT NULL,
  updated_at   DATETIME(3)     NOT NULL,
  INDEX idx_txn_reference (reference),
  INDEX idx_txn_sender_id (sender_id),
  INDEX idx_txn_receiver_id (receiver_id),
  INDEX idx_txn_status (status),
  INDEX idx_txn_created_at (created_at)
);
```

### notification-service DB: `dv_notification`

```sql
CREATE TABLE notifications (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id      BIGINT UNSIGNED NOT NULL,
  message      TEXT            NOT NULL,
  type         VARCHAR(50)     NOT NULL,   -- CREDIT | DEBIT | SYSTEM
  is_read      BOOLEAN         NOT NULL DEFAULT FALSE,
  created_at   DATETIME(3)     NOT NULL,
  INDEX idx_notif_user_id (user_id),
  INDEX idx_notif_is_read (is_read),
  INDEX idx_notif_created_at (created_at)
);
```

---

## Auth Flows

### Registration Flow
```
POST /auth/register
  → validate input (phone, email, password)
  → check duplicate phone/email → ErrUserAlreadyExists
  → hash password (bcrypt, cost 12)
  → insert user (is_active=false)
  → return 201 {message: "OTP sent to phone"}
  (OTP send triggered separately via /auth/send-otp)
```

### OTP Flow
```
POST /auth/send-otp {phone}
  → check user exists → ErrUserNotFound
  → check already active → ErrUserAlreadyActive
  → generate 6-digit OTP
  → SET Redis: otp:{phone} = OTP, TTL=5min
  → [mock SMS send in dev — log OTP]
  → return 200

POST /auth/verify-otp {phone, otp}
  → GET Redis: otp:{phone}
  → if redis.Nil → ErrOTPExpired
  → if mismatch → ErrOTPInvalid
  → DEL Redis: otp:{phone}   ← single use, delete immediately
  → UPDATE users SET is_active=true
  → return 200
```

### Login Flow
```
POST /auth/login {phone, password}
  → fetch user by phone → ErrUserNotFound
  → check is_active=true → ErrUserNotVerified
  → bcrypt.CompareHashAndPassword → ErrInvalidCredentials
  → generate access token (JWT, exp: 15min, claims: userID, jti)
  → generate refresh token (JWT, exp: 7days, claims: userID, jti)
  → SET Redis: refresh:{jti} = userID, TTL=7days
  → return 200 {access_token, refresh_token}
```

### Refresh Token Rotation
```
POST /auth/refresh {refresh_token}
  → parse and verify refresh JWT → ErrInvalidToken
  → extract jti from claims
  → GET Redis: refresh:{jti}
  → if redis.Nil → ErrTokenRevoked (STOLEN TOKEN — invalidate all tokens for user)
  → DEL Redis: refresh:{jti}           ← rotate: old token gone immediately
  → generate new access token (new jti)
  → generate new refresh token (new jti)
  → SET Redis: refresh:{new_jti} = userID, TTL=7days
  → return 200 {access_token, refresh_token}

STOLEN TOKEN DETECTION:
  → If refresh:{jti} not found in Redis but JWT is valid:
    → someone already used this token (attacker or legit user rotated it)
    → find all refresh:{*} keys for this userID and DEL all
    → return 401 ErrTokenRevoked
    → client must re-login
```

### Logout Flow
```
POST /auth/logout   (requires Authorization: Bearer <access_token>)
  → extract jti from access token claims
  → GET Redis: refresh:{jti} for this user
  → DEL Redis: refresh:{jti}
  → return 200
```

---

## gRPC Contracts

### proto/auth/auth.proto
```protobuf
syntax = "proto3";
package auth;
option go_package = "github.com/alokwritescode/digi-vault/proto/auth";

service AuthService {
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}

message ValidateTokenRequest {
  string token = 1;
}
message ValidateTokenResponse {
  bool   is_valid = 1;
  uint64 user_id  = 2;
  string jti      = 3;
}

message GetUserRequest {
  uint64 user_id = 1;
}
message GetUserResponse {
  uint64 user_id  = 1;
  string phone    = 2;
  string email    = 3;
  bool   is_active = 4;
}
```

### proto/wallet/wallet.proto
```protobuf
syntax = "proto3";
package wallet;
option go_package = "github.com/alokwritescode/digi-vault/proto/wallet";

service WalletService {
  rpc CreateWallet(CreateWalletRequest) returns (CreateWalletResponse);
  rpc GetBalance(GetBalanceRequest)     returns (GetBalanceResponse);
  rpc Credit(CreditRequest)             returns (MutationResponse);
  rpc Debit(DebitRequest)               returns (MutationResponse);
}

message CreateWalletRequest { uint64 user_id = 1; string currency = 2; }
message CreateWalletResponse { uint64 wallet_id = 1; }

message GetBalanceRequest  { uint64 user_id = 1; }
message GetBalanceResponse { double balance = 1; string currency = 2; }

message CreditRequest  { uint64 user_id = 1; double amount = 2; string reference = 3; }
message DebitRequest   { uint64 user_id = 1; double amount = 2; string reference = 3; }
message MutationResponse { bool success = 1; double new_balance = 2; }
```

### proto/txn/txn.proto
```protobuf
syntax = "proto3";
package txn;
option go_package = "github.com/alokwritescode/digi-vault/proto/txn";

service TxnService {
  rpc GetTransaction(GetTxnRequest) returns (GetTxnResponse);
}

message GetTxnRequest  { string reference = 1; }
message GetTxnResponse {
  string reference   = 1;
  uint64 sender_id   = 2;
  uint64 receiver_id = 3;
  double amount      = 4;
  string status      = 5;
  string created_at  = 6;
}
```

### proto/notification/notification.proto
```protobuf
syntax = "proto3";
package notification;
option go_package = "github.com/alokwritescode/digi-vault/proto/notification";

service NotificationService {
  rpc GetUnread(GetUnreadRequest) returns (GetUnreadResponse);
  rpc MarkRead(MarkReadRequest)   returns (MarkReadResponse);
}

message GetUnreadRequest  { uint64 user_id = 1; }
message GetUnreadResponse { repeated Notification notifications = 1; }
message Notification {
  uint64 id      = 1;
  string message = 2;
  string type    = 3;
  bool   is_read = 4;
}
message MarkReadRequest  { uint64 notification_id = 1; }
message MarkReadResponse { bool success = 1; }
```

---

## Pub/Sub Event Payloads

### Topic: `dv.transactions`

Published by: `txn-service` on every terminal transaction state (SUCCESS or FAILED)
Subscribed by: `notification-service`

#### Event: transaction.completed
```json
{
  "event_type": "transaction.completed",
  "version": "1.0",
  "timestamp": "2026-06-01T10:00:00Z",
  "data": {
    "reference": "uuid-v4",
    "sender_id": 1001,
    "receiver_id": 1002,
    "amount": 500.00,
    "currency": "INR",
    "status": "SUCCESS"
  }
}
```

#### Event: transaction.failed
```json
{
  "event_type": "transaction.failed",
  "version": "1.0",
  "timestamp": "2026-06-01T10:00:00Z",
  "data": {
    "reference": "uuid-v4",
    "sender_id": 1001,
    "receiver_id": 1002,
    "amount": 500.00,
    "currency": "INR",
    "status": "FAILED",
    "failure_reason": "insufficient_balance"
  }
}
```

### notification-service behavior on event
```
transaction.completed:
  → INSERT notification for sender:   "You sent ₹500.00 to user #1002. Ref: uuid"
  → INSERT notification for receiver: "You received ₹500.00 from user #1001. Ref: uuid"

transaction.failed:
  → INSERT notification for sender only: "Transfer of ₹500.00 failed. Reason: insufficient_balance"
```

---

## Elasticsearch Schema

### Index: `dv_transactions`

```json
{
  "mappings": {
    "properties": {
      "reference":    { "type": "keyword" },
      "sender_id":    { "type": "long" },
      "receiver_id":  { "type": "long" },
      "amount":       { "type": "double" },
      "currency":     { "type": "keyword" },
      "status":       { "type": "keyword" },
      "failure_reason": { "type": "keyword" },
      "created_at":   { "type": "date" }
    }
  },
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  }
}
```

### Query Patterns

**Transaction history for a user (sender or receiver), paginated, sorted by date:**
```json
{
  "query": {
    "bool": {
      "should": [
        { "term": { "sender_id": 1001 } },
        { "term": { "receiver_id": 1001 } }
      ],
      "minimum_should_match": 1
    }
  },
  "sort": [{ "created_at": { "order": "desc" } }],
  "from": 0,
  "size": 20
}
```

**Filter by status:**
```json
{
  "query": {
    "bool": {
      "must": [
        { "term": { "sender_id": 1001 } },
        { "term": { "status": "SUCCESS" } }
      ]
    }
  }
}
```

Write path: txn-service indexes to ES after MySQL insert succeeds.
Read path: txn-service queries ES for `/txn/history/:userID`.

---

## Error Scenarios Per Endpoint

### auth-service

| Endpoint | Scenario | HTTP | Error Sentinel |
|----------|----------|------|----------------|
| POST /auth/register | phone already exists | 409 | ErrUserAlreadyExists |
| POST /auth/register | email already exists | 409 | ErrUserAlreadyExists |
| POST /auth/register | invalid phone format | 400 | ErrValidation |
| POST /auth/send-otp | user not found | 404 | ErrUserNotFound |
| POST /auth/send-otp | user already active | 400 | ErrUserAlreadyActive |
| POST /auth/verify-otp | OTP expired (redis.Nil) | 410 | ErrOTPExpired |
| POST /auth/verify-otp | OTP mismatch | 400 | ErrOTPInvalid |
| POST /auth/login | user not found | 401 | ErrInvalidCredentials (never leak which field) |
| POST /auth/login | wrong password | 401 | ErrInvalidCredentials |
| POST /auth/login | user not verified | 403 | ErrUserNotVerified |
| POST /auth/refresh | malformed JWT | 401 | ErrInvalidToken |
| POST /auth/refresh | jti not in Redis | 401 | ErrTokenRevoked |
| POST /auth/refresh | token expired | 401 | ErrTokenExpired |
| POST /auth/logout | missing auth header | 401 | ErrMissingToken |

### wallet-service

| Endpoint | Scenario | HTTP | Error Sentinel |
|----------|----------|------|----------------|
| POST /wallet | wallet already exists for user | 409 | ErrWalletAlreadyExists |
| GET /wallet/:userID | wallet not found | 404 | ErrWalletNotFound |
| POST /wallet/debit | insufficient balance | 422 | ErrInsufficientBalance |
| POST /wallet/debit | wallet locked | 423 | ErrWalletLocked |
| POST /wallet/credit | wallet locked | 423 | ErrWalletLocked |

### txn-service

| Endpoint | Scenario | HTTP | Error Sentinel |
|----------|----------|------|----------------|
| POST /txn/transfer | sender == receiver | 400 | ErrSelfTransfer |
| POST /txn/transfer | amount <= 0 | 400 | ErrInvalidAmount |
| POST /txn/transfer | sender wallet not found | 404 | ErrWalletNotFound |
| POST /txn/transfer | insufficient balance | 422 | ErrInsufficientBalance |
| GET /txn/:id | transaction not found | 404 | ErrTransactionNotFound |

---

## Docker / Infra Setup

### docker-compose.yml services

```yaml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: dv_auth      # auth service default; others created via migrations
    ports: ["3306:3306"]
    volumes: [mysql_data:/var/lib/mysql]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  elasticsearch:
    image: elasticsearch:8.11.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
    ports: ["9200:9200"]

  auth-service:
    build: ./auth-service
    ports: ["8081:8081"]
    depends_on: [mysql, redis]
    env_file: ./auth-service/.env

  wallet-service:
    build: ./wallet-service
    ports: ["8082:8082"]
    depends_on: [mysql]
    env_file: ./wallet-service/.env

  txn-service:
    build: ./txn-service
    ports: ["8083:8083"]
    depends_on: [mysql, elasticsearch]
    env_file: ./txn-service/.env

  notification-service:
    build: ./notification-service
    ports: ["8084:8084"]
    depends_on: [mysql]
    env_file: ./notification-service/.env

  api-gateway:
    build: ./api-gateway
    ports: ["8080:8080"]
    depends_on: [auth-service, wallet-service, txn-service, notification-service]
    env_file: ./api-gateway/.env

volumes:
  mysql_data:
```

### Per-service Dockerfile pattern
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o service ./cmd/main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/service .
COPY .env.example .env
EXPOSE 8081
CMD ["./service"]
```

### .env.example (per service — auth example)
```env
APP_PORT=8081
APP_ENV=development

DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=dv_auth

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

JWT_ACCESS_SECRET=changeme-access-secret
JWT_REFRESH_SECRET=changeme-refresh-secret
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

OTP_TTL=5m
BCRYPT_COST=12
```

---

## Makefile Targets

```makefile
.PHONY: run stop test proto lint build clean

# Start full stack
run:
	docker-compose up --build -d

# Stop full stack
stop:
	docker-compose down

# Run all tests across all services
test:
	@for svc in auth-service wallet-service txn-service notification-service api-gateway; do \
		echo "Testing $$svc..."; \
		cd $$svc && go test ./... -coverprofile=coverage.out && \
		go tool cover -func=coverage.out && cd ..; \
	done

# Generate gRPC code from proto files
proto:
	@for proto in proto/**/*.proto; do \
		protoc --go_out=. --go-grpc_out=. $$proto; \
	done

# Lint all services
lint:
	@for svc in auth-service wallet-service txn-service notification-service api-gateway; do \
		echo "Linting $$svc..."; \
		cd $$svc && golangci-lint run ./... && cd ..; \
	done

# Build all service binaries
build:
	@for svc in auth-service wallet-service txn-service notification-service api-gateway; do \
		echo "Building $$svc..."; \
		cd $$svc && go build -o bin/service ./cmd/main.go && cd ..; \
	done

# Clean build artifacts
clean:
	@for svc in auth-service wallet-service txn-service notification-service api-gateway; do \
		rm -f $$svc/bin/service $$svc/coverage.out; \
	done

# Run single service locally (usage: make dev SVC=auth-service)
dev:
	cd $(SVC) && go run ./cmd/main.go

# Run tests for single service (usage: make test-svc SVC=auth-service)
test-svc:
	cd $(SVC) && go test ./... -v -coverprofile=coverage.out && go tool cover -func=coverage.out
```

---

## Service Completion Checklists

### auth-service
- [ ] shared/pkg/errors — all sentinel errors
- [ ] shared/pkg/logger — Logrus JSON setup with RequestID hook
- [ ] shared/pkg/jwt — generate, parse, verify (access + refresh)
- [ ] config/config.go — Viper loader
- [ ] domain/user.go — User model (no gorm.Model embedding)
- [ ] domain/token.go — RefreshToken model (Redis-backed, no DB table)
- [ ] repository/user_repository.go + user_repository_test.go
- [ ] repository/token_repository.go + token_repository_test.go (Redis mock)
- [ ] usecase/auth_usecase.go + auth_usecase_test.go
- [ ] handler/auth_handler.go + auth_handler_test.go (httptest)
- [ ] cmd/main.go — DI wiring only
- [ ] Dockerfile
- [ ] .env.example
- [ ] All coverage targets met

### wallet-service
- [ ] domain/wallet.go
- [ ] repository/wallet_repository.go + test
- [ ] usecase/wallet_usecase.go + test
- [ ] handler/wallet_handler.go + test
- [ ] gRPC server (WalletService) + proto generated
- [ ] cmd/main.go
- [ ] Dockerfile

### txn-service
- [ ] domain/transaction.go
- [ ] repository/txn_repository.go + test (MySQL)
- [ ] repository/txn_es_repository.go + test (Elasticsearch)
- [ ] usecase/txn_usecase.go + test
- [ ] handler/txn_handler.go + test
- [ ] gRPC client → wallet-service
- [ ] Pub/Sub publisher (transaction events)
- [ ] cmd/main.go
- [ ] Dockerfile

### notification-service
- [ ] domain/notification.go
- [ ] repository/notification_repository.go + test
- [ ] usecase/notification_usecase.go + test
- [ ] Pub/Sub subscriber (consume dv.transactions topic)
- [ ] cmd/main.go
- [ ] Dockerfile

### api-gateway
- [ ] middleware/auth.go — JWT validation via auth-service gRPC
- [ ] middleware/rate_limiter.go — Redis token bucket
- [ ] middleware/request_id.go — inject UUID into context + logs
- [ ] router/router.go — reverse proxy to all 4 services
- [ ] cmd/main.go
- [ ] Dockerfile

### Infrastructure
- [ ] docker-compose.yml — all services + MySQL + Redis + Elasticsearch
- [ ] .env.example per service
- [ ] proto/ — all .proto files + generated Go code
- [ ] Makefile — all targets above
- [ ] README — startup guide, architecture diagram, API reference