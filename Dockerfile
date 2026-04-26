# Dashboard build stage — outputs directly to the Go embed path
FROM node:22-alpine AS dashboard-builder
WORKDIR /dashboard
RUN corepack enable
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json tsconfig.base.json ./
COPY packages/shared/package.json packages/shared/
COPY packages/dashboard/package.json packages/dashboard/
RUN pnpm install --frozen-lockfile
COPY packages/shared/ packages/shared/
COPY packages/dashboard/ packages/dashboard/
RUN pnpm --filter @ogame-bot/shared build && pnpm --filter @ogame-bot/dashboard build

# Go build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=dashboard-builder /dashboard/internal/dashboard/static/ internal/dashboard/static/

RUN CGO_ENABLED=0 GOOS=linux go build -o /bot ./cmd/bot

# Runtime stage
FROM alpine:3.21
RUN apk --no-cache add ca-certificates

COPY --from=builder /bot /app/bot

WORKDIR /app
CMD ["./bot"]
