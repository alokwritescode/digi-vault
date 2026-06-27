#!/bin/bash
set -e

REPO_PATH="$HOME/Desktop/git/digi-vault"
PROGRESS_FILE="$REPO_PATH/.dev-progress"

# All services in build order
SERVICES=("auth-service" "wallet-service" "txn-service" "notification-service" "api-gateway")

echo ""
echo "╔══════════════════════════════════════════╗"
echo "║     DigiVault — Daily Dev Session        ║"
echo "╚══════════════════════════════════════════╝"
echo ""

# Navigate to repo
cd "$REPO_PATH" || { echo "❌ Repo not found at $REPO_PATH"; exit 1; }

# ── Day tracking ──────────────────────────────────────────────────────────────
# Read or initialise start date
if [ ! -f "$PROGRESS_FILE" ]; then
  echo "START_DATE=$(date '+%Y-%m-%d')" > "$PROGRESS_FILE"
  echo "DAY=1" >> "$PROGRESS_FILE"
  echo "🗓  First run — project clock started."
fi

source "$PROGRESS_FILE"
TODAY=$(date '+%Y-%m-%d')

# Advance day if calendar date has changed
if [ "$TODAY" != "$START_DATE" ] && [ -n "$LAST_RUN" ] && [ "$TODAY" != "$LAST_RUN" ]; then
  DAY=$((DAY + 1))
  if [ "$DAY" -gt 7 ]; then DAY=7; fi
fi

# Cap at 7
if [ "$DAY" -gt 7 ]; then DAY=7; fi

# Persist
{
  echo "START_DATE=$START_DATE"
  echo "DAY=$DAY"
  echo "LAST_RUN=$TODAY"
} > "$PROGRESS_FILE"

echo "📍 Branch : $(git branch --show-current)"
echo "📅 Date   : $TODAY"
echo "🗓  Day    : $DAY / 7"
echo ""

BRANCH=$(git branch --show-current)

# ── 7-Day Scope Map ───────────────────────────────────────────────────────────
case $DAY in
  1)
    DAY_TITLE="Foundation — shared packages + auth domain"
    DAY_SCOPE="
SCOPE FOR TODAY (Day 1 of 7):
- shared/pkg/errors   → define ALL sentinel errors for all services
- shared/pkg/logger   → Logrus JSON formatter, RequestID hook
- shared/pkg/jwt      → generate + parse + verify (access + refresh tokens, jti claim)
- auth-service/config → Viper config loader
- auth-service/domain → user.go (User model, no gorm.Model embedding)
- auth-service/domain → token.go (RefreshToken — Redis-backed, no DB table)

TDD RULE: write tests for jwt package (generate, parse, verify) and errors package first.
TARGET COMMITS TODAY: 3–5 small focused commits.
DO NOT start repository or usecase layer today."
    ;;
  2)
    DAY_TITLE="Auth repository layer — MySQL + Redis"
    DAY_SCOPE="
SCOPE FOR TODAY (Day 2 of 7):
- auth-service/repository/user_repository.go  → GORM: Create, FindByPhone, FindByEmail, FindByID, SoftDelete
- auth-service/repository/user_repository_test.go → mock DB, table-driven tests
- auth-service/repository/token_repository.go → Redis: Set, Get, Delete refresh token by jti
- auth-service/repository/token_repository_test.go → mock Redis client, table-driven tests

TDD RULE: write each test file before the implementation file. Run go test (red) first.
TARGET COMMITS TODAY: 3–5 commits scoped to repository layer.
DO NOT start usecase layer today."
    ;;
  3)
    DAY_TITLE="Auth usecase layer — all flows"
    DAY_SCOPE="
SCOPE FOR TODAY (Day 3 of 7):
- auth-service/usecase/auth_usecase.go → Register, SendOTP, VerifyOTP, Login, RefreshToken, Logout
- auth-service/usecase/auth_usecase_test.go → mock repositories, table-driven tests for every flow including:
    - happy path
    - duplicate user
    - OTP expired (redis.Nil)
    - wrong password
    - stolen refresh token detection
    - already active user

TDD RULE: write all test cases first. Minimum 90% usecase coverage.
TARGET COMMITS TODAY: 3–5 commits.
DO NOT start handler layer today."
    ;;
  4)
    DAY_TITLE="Auth handler + wiring + Dockerise"
    DAY_SCOPE="
SCOPE FOR TODAY (Day 4 of 7):
- auth-service/handler/auth_handler.go → Gin handlers for all 6 endpoints
- auth-service/handler/auth_handler_test.go → httptest.NewRecorder() for all endpoints + error cases
- auth-service/cmd/main.go → composition root: DI wiring only, no business logic
- auth-service/Dockerfile → multi-stage build
- auth-service/.env.example → all required vars
- Update docker-compose.yml → add auth-service block

TDD RULE: handler tests use httptest, mock usecase interface.
TARGET COMMITS TODAY: 3–5 commits.
auth-service must be fully runnable by end of today."
    ;;
  5)
    DAY_TITLE="wallet-service — full service end to end"
    DAY_SCOPE="
SCOPE FOR TODAY (Day 5 of 7):
- wallet-service/domain/wallet.go
- wallet-service/repository/wallet_repository.go + test
- wallet-service/usecase/wallet_usecase.go + test (include ErrInsufficientBalance, ErrWalletLocked)
- wallet-service/handler/wallet_handler.go + test
- wallet-service/gRPC server → WalletService (CreateWallet, GetBalance, Credit, Debit)
- wallet-service/cmd/main.go → DI wiring
- wallet-service/Dockerfile
- proto/wallet/wallet.proto → generate Go code
- Update docker-compose.yml → add wallet-service block

TDD RULE: domain 100%, usecase 90%+, handler 80%+.
TARGET COMMITS TODAY: 4–6 commits.
wallet-service must be fully runnable and gRPC server live by end of today."
    ;;
  6)
    DAY_TITLE="txn-service + notification-service + Pub/Sub"
    DAY_SCOPE="
SCOPE FOR TODAY (Day 6 of 7):
txn-service:
- domain/transaction.go
- repository/txn_repository.go + test (MySQL)
- repository/txn_es_repository.go + test (Elasticsearch index + query)
- usecase/txn_usecase.go + test (ErrSelfTransfer, ErrInvalidAmount, ErrInsufficientBalance)
- handler/txn_handler.go + test
- gRPC client → wallet-service (debit sender, credit receiver)
- Pub/Sub publisher → publish transaction.completed / transaction.failed events
- cmd/main.go, Dockerfile

notification-service:
- domain/notification.go
- repository/notification_repository.go + test
- usecase/notification_usecase.go + test
- Pub/Sub subscriber → consume dv.transactions topic, insert notifications for sender + receiver
- cmd/main.go, Dockerfile

Update docker-compose.yml → add txn-service + notification-service.
TARGET COMMITS TODAY: 5–7 commits spread across both services.
proto/txn/txn.proto and proto/notification/notification.proto must be generated."
    ;;
  7)
    DAY_TITLE="api-gateway + integration + polish"
    DAY_SCOPE="
SCOPE FOR TODAY (Day 7 of 7 — FINAL DAY):
api-gateway:
- middleware/auth.go → JWT validation via auth-service gRPC (ValidateToken RPC)
- middleware/rate_limiter.go → Redis token bucket (100 req/min per userID)
- middleware/request_id.go → inject UUID into context and all log lines
- router/router.go → reverse proxy routes to all 4 downstream services
- cmd/main.go, Dockerfile
- Update docker-compose.yml → add api-gateway block

Integration:
- Verify full stack starts with: docker-compose up --build
- Run make test → all services must pass
- Verify end-to-end flow: register → OTP → login → transfer → check notification

Final polish:
- Update PRODUCT.md checklists (mark all items done)
- Update README → startup guide (make run), architecture diagram (ASCII), API reference table
- Add Makefile at repo root with: run, stop, test, proto, lint, build, dev, test-svc targets
- Verify .env.example exists for every service
- Verify no hardcoded secrets anywhere

TARGET COMMITS TODAY: 5–8 commits. This is ship day."
    ;;
esac

# ── Sync ──────────────────────────────────────────────────────────────────────
echo "🔄 Pulling latest..."
git pull origin "$BRANCH" --rebase 2>/dev/null || echo "⚠️  Nothing to pull or rebase skipped."
echo ""

# ── Repo status ───────────────────────────────────────────────────────────────
echo "📊 Repo status:"
git status --short
echo ""

# ── Service readiness ─────────────────────────────────────────────────────────
echo "🗂  Services:"
ACTIVE_SERVICES=()
for svc in "${SERVICES[@]}"; do
  if [ -f "$svc/go.mod" ]; then
    echo "   ✅  $svc"
    ACTIVE_SERVICES+=("$svc")
  else
    echo "   ⬜  $svc  (not started)"
  fi
done
echo ""

# ── Tests ─────────────────────────────────────────────────────────────────────
TOTAL_PASS=0
TOTAL_FAIL=0
if [ ${#ACTIVE_SERVICES[@]} -eq 0 ]; then
  echo "🧪 No services initialised yet — skipping tests."
else
  echo "🧪 Running tests..."
  for svc in "${ACTIVE_SERVICES[@]}"; do
    cd "$svc"
    if go test ./... 2>&1 | tail -5; then
      TOTAL_PASS=$((TOTAL_PASS + 1))
    else
      TOTAL_FAIL=$((TOTAL_FAIL + 1))
    fi
    cd "$REPO_PATH"
  done
  echo "   ✅ $TOTAL_PASS passed  ❌ $TOTAL_FAIL failed"
fi
echo ""

# ── TODOs ─────────────────────────────────────────────────────────────────────
echo "📝 Open TODOs:"
TODO_COUNT=$(grep -rn "TODO\|FIXME\|HACK" --include="*.go" . 2>/dev/null | wc -l | tr -d ' ')
if [ "$TODO_COUNT" -gt 0 ]; then
  grep -rn "TODO\|FIXME\|HACK" --include="*.go" . 2>/dev/null | head -10
  echo "   ($TODO_COUNT total)"
else
  echo "   None."
fi
echo ""

# ── Checklist progress ────────────────────────────────────────────────────────
DONE=0; PENDING=0
if [ -f "PRODUCT.md" ]; then
  DONE=$(grep -c "\- \[x\]" PRODUCT.md 2>/dev/null || true)
  PENDING=$(grep -c "\- \[ \]" PRODUCT.md 2>/dev/null || true)
fi
TOTAL=$((DONE + PENDING))
echo "📋 Checklist: ✅ $DONE / $TOTAL complete"
echo ""

# ── Recent commits ────────────────────────────────────────────────────────────
echo "📜 Last 5 commits:"
git log --oneline -5
echo ""

# ── Hand off to Claude Code ───────────────────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🤖 Day $DAY / 7 — $DAY_TITLE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

PROMPT="You are working on DigiVault — a production-grade digital wallet API in Go.

STEP 1: Read CLAUDE.md (code standards — non-negotiable).
STEP 2: Read PRODUCT.md (full spec: schemas, auth flows, gRPC contracts, Pub/Sub payloads, Elasticsearch schema, error tables, Docker setup, Makefile, checklists).

PROJECT STATE:
- Branch: $BRANCH
- Day: $DAY / 7
- Active services: ${ACTIVE_SERVICES[*]:-none initialised yet}
- Test results: $TOTAL_PASS services passing, $TOTAL_FAIL failing
- Open TODOs: $TODO_COUNT
- Checklist: $DONE / $TOTAL items complete

TODAY'S TITLE: $DAY_TITLE
$DAY_SCOPE

SELF-PROMPTING RULES (follow without being asked):
1. Run go test ./... -v → capture all failures
2. Grep TODOs in scope for today
3. Read git log --oneline -10
4. Propose a numbered plan (max 5 tasks) aligned EXACTLY to today's scope above
   → STOP HERE. Wait for approval before writing any code.
5. After approval: for each task — write test first (red) → implement (green) → refactor
6. After all tasks: run full go test ./... -coverprofile=coverage.out
7. If any public API changed, update README section
8. Stage with git add .
9. Propose a Conventional Commits message scoped to today's work
   → STOP HERE. Wait for 'commit approved' before committing.
10. Do NOT push. User pushes manually.

IMPORTANT CONSTRAINTS:
- Do NOT work outside today's scope — resist the urge to build ahead
- Each task should be one focused commit — not a giant single commit
- Every file you create must have a corresponding test file
- Never skip the red phase — run the test before writing the implementation"

claude "$PROMPT"
