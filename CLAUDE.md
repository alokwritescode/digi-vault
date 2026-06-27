# DigiVault — Claude Code Instructions

## Project
Production-grade digital wallet API in Go. 5 microservices.
Each service: clean architecture (handler → usecase → repository → domain).

Services: auth-service | wallet-service | txn-service | api-gateway | notification-service

## Daily Automation Loop
When triggered via `daily.sh`, do the following IN ORDER:

1. Run `go test ./... -v` — capture all failures
2. Grep all TODOs: `grep -rn "TODO\|FIXME\|HACK" --include="*.go" .`
3. Read recent git log: `git log --oneline -10`
4. Propose a numbered plan (max 5 tasks for the session) — **STOP AND WAIT FOR APPROVAL**
5. For each task: write test first → run test (must fail) → implement → run test (must pass)
6. After all tasks: run full `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out` — all must pass before proceeding
7. Update relevant README section if any public API or flow changed
8. Stage changes with `git add .`
9. Propose a Conventional Commits message — **STOP AND WAIT FOR APPROVAL**
10. Do NOT commit or push without explicit "commit approved" from user

## TDD — Non-Negotiable
- **Red → Green → Refactor** on every change. No exceptions.
- Write the test file first: `<feature>_test.go` alongside the implementation file
- Use `testify/assert` and `testify/mock` for assertions and mocks
- Table-driven tests always: `tests := []struct{ name, input, expected }{...}`
- Mock all external dependencies (DB, Redis, gRPC) — never hit real infra in unit tests
- Test file naming: `user_repository_test.go` tests `user_repository.go`
- Minimum coverage targets:
  - Domain layer: 100%
  - Usecase layer: 90%+
  - Repository layer: 80%+ (integration tests allowed here)
  - Handler layer: 80%+ (use httptest)

## Architecture — Non-Negotiable
- Every repository behind an interface
- Constructor returns interface: `NewUserRepo() UserRepository`
- Context propagated through every layer — every function accepts `ctx context.Context`
- `defer cancel()` after every `context.WithTimeout` or `context.WithCancel`
- `main.go` is composition root only — no business logic

## Error Handling
- Sentinel errors: `var Err = errors.New(...)` — never const, never inline
- Wrap with `fmt.Errorf("%w", err)` — never `errors.New(err.Error())`
- Two-layer translation at repo boundary: DB driver → GORM → domain sentinel
- Never ignore errors — every `err` must be checked

## Gin Handlers
- `c.ShouldBindJSON` for request binding — never `c.BindJSON`
- `c.JSON(...)` always followed by `return`
- Typed context keys: `type contextKey string`
- `c.Request = c.Request.WithContext(ctx)` in every middleware that modifies context
- Handler tests use `httptest.NewRecorder()` + `httptest.NewRequest()`

## GORM
- `db.WithContext(ctx)` on every query
- `var model Model` (value, not pointer) for scan targets
- `db.Model(&m).Updates(&dto)` for partial updates
- Soft delete via `gorm.DeletedAt` field named `DeletedAt`

## Redis
- Always set TTL — no `Set` without expiry
- Check `redis.Nil` before `err != nil`
- Key format: `resource:identifier` — e.g. `refresh:jti`, `otp:phone`
- `Del` immediately after single-use token verification

## Logging
- Logrus JSON formatter only
- `logger.WithContext(ctx)` on every log call
- RequestId injected via Logrus hook — never manually in every handler
- Log at entry and exit of every usecase method
- Never log raw passwords, tokens, or PII

### JWT
- Always verify signing method before returning secret
- `strings.TrimPrefix` to extract Bearer token — never `header[7:]`
- Wrap parse errors with `%w`
- Every token carries `jti` (UUID) claim

### gRPC
- Embed `UnimplementedXxxServer` in every server struct
- Return `status.Errorf(codes.X, "msg")` — never `fmt.Errorf`
- Create connection once at startup — never per request

## Services and ports
- auth-service:      :8081
- wallet-service:    :8082
- txn-service:       :8083
- notification-service: :8084 (internal only)
- api-gateway:       :8080

## What not to do
- Do not use `gorm.Model` embedding — declare fields explicitly
- Do not return concrete types from constructors
- Do not store raw JWT in Redis — store jti only
- Do not use `c.BindJSON` — it writes 400 automatically and bypasses our error handling
- Do not hardcode secrets — use Viper + .env

## Build Order (do not skip ahead)
1. auth-service (current)
2. wallet-service
3. txn-service
4. notification-service
5. api-gateway (last — depends on all others)

## Commit Convention (Conventional Commits)
Format: `<type>(<scope>): <summary>`
Types: feat | fix | refactor | chore | docs | test | perf
Scope = service name: auth | wallet | txn | gateway | notification
Examples:
  test(auth): add table-driven tests for UserRepository
  feat(auth): implement register usecase with OTP flow
  fix(auth): handle duplicate email in registration

## Branch Strategy
main → develop → feat/<service-name>
Never commit directly to main or develop.
Current active branch: feat/auth-service

## Do NOT
- Commit or push without explicit approval
- Delete any files without confirmation
- Modify .env or any secrets file
- Skip writing the test before implementation
- Hit real DB/Redis in unit tests — always mock
