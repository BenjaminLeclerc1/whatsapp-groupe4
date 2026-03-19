# --- Stage 1: Build ---
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app

# 1. Copy go.mod from the root (two levels up)
COPY go.mod go.sum ./
RUN go mod download

# 2. Copy the entire project to handle internal dependencies
COPY . .

# 3. Build the specific service
RUN go build -o main ./cmd/chat-service/main.go

# --- Stage 2: Final Runtime ---
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /root/

# Copy binary and migrations
COPY --from=builder /app/main .
COPY --from=builder /app/migrations/chat-service ./migrations/chat-service

EXPOSE 8083
CMD ["./main"]