# Stage 1: Build static Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install ca-certificates and git
RUN apk add --no-cache ca-certificates git

# Cache Go modules
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source code
COPY backend/ .

# Build statically linked binary with pure Go SQLite
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/homeboard ./cmd/server

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    mkdir -p /app/data && \
    addgroup -S homeboard && adduser -S homeboard -G homeboard && \
    chown -R homeboard:homeboard /app

COPY --from=builder --chown=homeboard:homeboard /app/bin/homeboard /app/homeboard

USER homeboard

ENV PORT=8080 \
    HOST=0.0.0.0 \
    DB_PATH=/app/data/homeboard.db \
    APP_ENV=production

EXPOSE 8080

VOLUME ["/app/data"]

ENTRYPOINT ["/app/homeboard"]
