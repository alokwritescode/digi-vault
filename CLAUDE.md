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
5. Execute tasks ONE AT A TIME. For each task:

   a. Write test first → run test (must fail — show output) → implement → run test (must pass — show output)
   
   b. Run coverage for files changed in this task only:
      `go test ./path/to/package/... -coverprofile=coverage.out && go tool cover -func=coverage.out`
   
   c. Produce CODE WALKTHROUGH for every file created or changed in this task:
      - What this file is and its role in the architecture
      - Why it exists — what breaks without it
      - Key functions line by line in plain English
      - Design decisions and why (alternatives considered)
      - Knowledge map check: ✅ revise briefly / ⚠️ teach the gap / ❌ teach fully
      - Cross-track connection: link to System Design / LLD / Go GAP where relevant
      - TDD teaching: explain the full Red → Green → Refactor cycle for this task
      - 2–3 interview Q&As for this file with crisp answers
   
   d. Stage only the files changed in this task:
      `git add <specific files only — never git add .>`
   
   e. Propose a Conventional Commits message for THIS task only — **STOP AND WAIT FOR APPROVAL**
   
   f. After "commit approved" — commit immediately:
      `git commit -m "<type>(<scope>): <summary>"`
   
   g. Move to next task. Repeat from step 5a.

6. After ALL tasks done: run full `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out`
7. If any public API or flow changed — update the relevant README section
8. Stage and commit README change separately:
   `git add README.md`
   Propose commit: `docs(<scope>): update README for <what changed>` — **WAIT FOR APPROVAL** → commit
9. Do NOT push at any point. User pushes manually after reviewing all commits:
   `git log --oneline` then `git push origin <branch>`

## TDD — Non-Negotiable

### Rules
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

### TDD Teaching Mode — Non-Negotiable
The user has zero prior experience writing test cases. Every time a new test file
is introduced in this session, before writing any code, explain the following:

#### Before writing each test file, teach this:

1. **What is this test testing?**
   - Name the exact function/method being tested
   - Explain what "correct behaviour" means for it in plain English

2. **TDD cycle — explain it for THIS specific case**
   - Red: "We write this test first. It will fail because the function doesn't exist yet.
     Here's why that failure is intentional and valuable..."
   - Green: "Now we write the minimum code to make it pass. Not perfect code — just passing."
   - Refactor: "Now we clean it up without breaking the test. The test is our safety net."

3. **Anatomy of this test file — explain every line**
   - What is `t *testing.T` and why every test receives it
   - What `t.Run()` does and why we use it for table-driven tests
   - What the `tests := []struct{ name, input, expected }{}` pattern is and why it beats
     writing 10 separate test functions
   - What `assert.Equal(t, expected, actual)` does and why order matters
     (expected first, actual second — always)
   - What a mock is, why we mock dependencies, and how `testify/mock` works
   - What `suite.Run` is if using test suites

4. **Why this test is structured this way**
   - Why we test the interface, not the concrete type
   - Why we never hit real DB/Redis in unit tests (speed, isolation, determinism)
   - What the test is NOT testing (boundaries of the test)

5. **Common beginner mistakes to avoid in this test**
   - Asserting on the wrong value
   - Forgetting to call `mock.AssertExpectations(t)` at the end
   - Testing implementation details instead of behaviour
   - Writing tests that always pass (false positives)
   - Not running the test in red phase first

6. **Interview Q&A for this test — with crisp answers**
   - "What is table-driven testing and why do Go developers prefer it?"
   - "What is the difference between a mock and a stub?"
   - "Why should unit tests never hit a real database?"
   - "What does 100% coverage actually mean — and is it enough?"
   - Provide crisp 2–3 sentence answers the user can deliver confidently

#### During the Red phase — narrate out loud:
- Show the exact `go test ./... -v -run TestFunctionName` command to run
- Show the expected failure output and explain why each line of the error means what it means
- Confirm: "This failure is correct. This is exactly what we want to see."

#### During the Green phase — narrate out loud:
- Explain why you're writing the minimum implementation, not the full one
- Point out any shortcuts taken and flag them explicitly for the refactor phase

#### During the Refactor phase — narrate out loud:
- Explain every change made and why
- Re-run tests after each change and confirm they still pass
- Explain: "The test didn't change. Only the implementation did. That's the point."

#### After the full TDD cycle for each task, give a 2-line summary:
"You just practiced: [specific TDD concept]. Remember this for interviews because: [reason]."

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

## Knowledge Map — Personalised Learning Context

This section tells Claude Code exactly what the user knows, what they're weak on,
and how to teach during code walkthroughs. Check this before every explanation.
If a concept is marked ✅ KNOWN — revise briefly, don't re-teach from scratch.
If marked ⚠️ PARTIAL — teach the gap only.
If marked ❌ UNKNOWN — teach fully with examples before moving on.

---

### Golang

✅ KNOWN — confident, production-grade:
- Gin handlers, middleware, route grouping
- GORM: queries, associations, soft delete (knows DeletedAt field)
- Clean Architecture: handler → usecase → repository → domain
- Context propagation (`ctx context.Context` everywhere)
- Redis: SET/GET/DEL with TTL, redis.Nil check
- JWT: generate, parse, Bearer extraction
- Error wrapping with `fmt.Errorf("%w", err)`
- Sentinel errors with `errors.New`
- Interfaces: constructor returns interface pattern
- gRPC: proto definitions, basic service structure
- Docker, docker-compose, GCP basics
- Pub/Sub: publishing events
- Elasticsearch: basic indexing and querying

⚠️ PARTIAL — knows concept, gaps in implementation:
- gRPC: knows theory, weak on actual Go implementation (interceptors, deadlines)
- Refresh token rotation: understands the concept, hasn't implemented stolen token detection
- DI wiring: knows what it is, shaky on multi-layer wiring in main.go
- Logrus: used it, but not structured JSON formatter or RequestID hook pattern
- Viper: knows it exists, hasn't wired it with .env properly

❌ UNKNOWN — needs full teaching:
- GMP Scheduler (goroutine/machine/processor model)
- Escape Analysis
- Nil Interface Trap
- panic/recover patterns
- Slice internals (header, capacity growth)
- Map internals (bucket structure, hash collisions)
- Table-driven tests (ZERO prior test writing experience)
- Generics
- TDD — has never written a test file before

Recurring Go mistakes to watch (correct immediately if seen):
- iota expression repetition
- `continue` vs `break` in switch/for
- Package qualifier syntax: `user.User{}` not `User{}`
- `gin.Context` vs `context.Context` — they are NOT the same
- GORM `deleted_at` column naming (snake_case)
- Two-layer TranslateError pattern at repo boundary
- `c.BindJSON` — always wrong, use `c.ShouldBindJSON`

---

### TDD / Testing

❌ UNKNOWN — zero prior experience:
- Has never written a test file in any language
- Does not know what `t *testing.T` is
- Does not know table-driven test structure
- Does not know what a mock is or how testify/mock works
- Does not know red/green/refactor cycle from practice

Teaching rule: Treat every test file as the first one ever.
Explain every line. Never assume prior context.
After each TDD cycle, ask: "What did you just practice?"
and wait for the answer before moving on.

---

### System Design

✅ KNOWN — strong:
- Actor-based FR decomposition
- Read-heavy vs write-heavy ratio identification
- Queue and cache component selection
- Basic estimation approach

⚠️ PARTIAL:
- Tradeoffs: makes good decisions in diagrams but doesn't verbalize them explicitly
- Storage estimation: sometimes omits it
- Mechanism-level explanations: knows WHAT to use, struggles to explain HOW it works internally
- Durability NFR: was wrong early on, now corrected but needs specificity

Sessions completed: S0–S6 (scores: 6.0 → 7.5 → 8.0 → 8.0 → 7.5)
Current weak areas: tradeoff articulation, storage estimation, mechanism depth

When DigiVault concepts overlap with System Design (Redis, Pub/Sub, Elasticsearch,
gRPC, JWT, rate limiting) — connect the code to the system design concept explicitly:
"This is the same Redis TTL tradeoff you covered in System Design S3."

---

### LLD / Design Patterns

✅ KNOWN — uses in production without knowing formal names:
- Singleton (DB connection with sync.Once)
- Strategy (vehicle dispatch logic in Inventix)
- Observer (Pub/Sub event flow)
- Chain of Responsibility (Gin middleware chain)

⚠️ PARTIAL:
- Factory: confused with Prototype during calibration
- Builder: taught in S1, not yet applied in code

❌ UNKNOWN — flagged as drill topics:
- Abstract Factory
- Prototype
- Composite
- Bridge

Sessions completed: S0 (SOLID + 5-step framework), S1 (Singleton, Factory, Builder)
SOLID: reproduced all 5 correctly after one pass. ASCII diagramming taught in S0.

When DigiVault code uses a pattern — call it out by name:
"This constructor returning an interface is the Factory pattern.
This middleware chain is Chain of Responsibility.
You already know this from LLD S1."

---

### DSA

Topics strong (no re-teaching needed):
- Arrays (basic-moderate)
- Sliding Window (strong)
- Two Pointers (strong)
- HashMap / Hashing patterns (strong)

Topics weak (teach when referenced in code):
- Stack / Monotonic Stack (weak)
- Binary Search (weak)
- Strings (weak)

Topics untouched (if code references these, teach from zero):
- Queue, Linked List, Trees, Graphs, DP, Heap/PQ, Tries

27 problems solved total. ~7 sessions complete.
Recurring DSA bugs (flag immediately if seen in any code):
- Returning wrong metric (inner value instead of window size)
- Wrong map key types
- make() + append() double-allocation
- Result update inside conditional instead of outside
- Missing O(1) space optimisation when two-pointer approach applies
- Incorrect complexity notation (e.g. writing "O(n) = n")

---

### How to Use This Map During Code Walkthrough

For every file explained after a commit:

1. Check which concepts from this map appear in the file
2. For each concept:
   - ✅ KNOWN → say "You know this — quick revision: [1 sentence reminder]"
   - ⚠️ PARTIAL → say "You've seen this before but had a gap here: [teach the gap only]"
   - ❌ UNKNOWN → say "This is new. Let me teach it properly:" [full explanation]
3. Connect to other tracks where relevant:
   - DigiVault code ↔ System Design session
   - DigiVault code ↔ LLD pattern
   - DigiVault code ↔ Go GAP queue item
4. End each file explanation with:
   "Interview question for this file: [question] — Answer: [crisp 2-sentence answer]"