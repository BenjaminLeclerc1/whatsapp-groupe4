# # --- Stage 1: Build ---
# FROM golang:1.24-alpine AS builder
# RUN apk add --no-cache gcc musl-dev
# WORKDIR /app

# # 1. Copy go.mod from the root (two levels up)
# COPY go.mod go.sum ./
# RUN go mod download

# # 2. Copy the entire project to handle internal dependencies
# COPY . .

# # 3. Build the specific service
# RUN go build -o main ./cmd/chat-service/main.go

# # --- Stage 2: Final Runtime ---
# FROM alpine:latest
# RUN apk add --no-cache ca-certificates tzdata
# WORKDIR /root/

# # Copy binary and migrations
# COPY --from=builder /app/main .
# COPY --from=builder /app/migrations/chat-service ./migrations/chat-service

# EXPOSE 8083
# CMD ["./main"]

# --- Stage 1: Build ---
FROM golang:1.24-alpine AS builder

# Install git and build tools
RUN apk add --no-cache ca-certificates gcc git musl-dev

# Set Go proxy (avoids TLS handshake timeouts)
ENV GOPROXY=https://goproxy.io,direct
ENV GOSUMDB=off

WORKDIR /app

# 1. Copy go.mod and go.sum first (for caching)
COPY go.mod go.sum ./

# 2. Download dependencies
RUN go mod download

# 3. Copy the full source code
COPY . .

# 4. Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /chat-service ./cmd/chat-service/main.go

# --- Stage 2: Final Runtime ---
FROM alpine:3.19

# Install certificates and timezone
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

# Copy binary and migrations
COPY --from=builder /chat-service .
COPY --from=builder /app/migrations/chat-service ./migrations/chat-service

EXPOSE 8083

CMD ["./chat-service"]