# ── Stage 1: builder ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /workspace

# Copy workspace file — needed for cross-module resolution (shared, proto, auth-service)
COPY go.work ./

# Copy go.mod + go.sum for each workspace module first so Docker can cache this
# layer independently of source changes. Dependencies are re-downloaded only when
# go.mod or go.sum change, not on every source edit.
COPY shared/go.mod shared/go.sum ./shared/
COPY proto/go.mod proto/go.sum ./proto/
COPY auth-service/go.mod auth-service/go.sum ./auth-service/

# Download external dependencies for auth-service.
# shared and proto are local workspace modules — no download needed for them.
RUN cd auth-service && go mod download

# Copy full source after deps are cached
COPY shared/ ./shared/
COPY proto/ ./proto/
COPY auth-service/ ./auth-service/

WORKDIR /workspace/auth-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service ./cmd/...

# ── Stage 2: runner ───────────────────────────────────────────────────────────
FROM alpine:3.19

# ca-certificates: needed for outbound HTTPS (e.g. token introspection)
# tzdata: needed for time.LoadLocation in config or logging
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /auth-service .

EXPOSE 8081 9091

ENTRYPOINT ["/app/auth-service"]
