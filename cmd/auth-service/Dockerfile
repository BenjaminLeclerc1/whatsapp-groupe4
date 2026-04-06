# # Build stage
# FROM golang:1.24-alpine AS builder

# WORKDIR /app

# COPY go.mod go.sum* ./
# # RUN go mod download
# ENV GOPROXY=https://proxy.golang.org,https://goproxy.cn,direct
# ENV GOSUMDB=sum.golang.org
# COPY . .
# RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service ./cmd/auth-service

# # Runtime stage
# FROM alpine:3.19

# WORKDIR /app

# # 1. Copier le binaire
# COPY --from=builder /auth-service .

# # 2. CRITICAL: Copier les fichiers de migration
# # C'est ce qui permet de créer les tables messages, channels, etc.
# COPY --from=builder /app/migrations ./migrations

# EXPOSE 8084

# CMD ["./auth-service"]


# --- Stage 1: Build ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build tools (needed for some Go modules)
RUN apk add --no-cache git gcc musl-dev ca-certificates

# Use a reliable Go proxy
ENV GOPROXY=https://goproxy.io,direct
ENV GOSUMDB=off

# 1️⃣ Copy go.mod/go.sum and download dependencies
COPY go.mod go.sum* ./
RUN go mod download

# 2️⃣ Copy the full source code
COPY . .

# 3️⃣ Build the auth-service binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service ./cmd/auth-service

# --- Stage 2: Runtime ---
FROM alpine:3.19

WORKDIR /app

# Install certificates to allow HTTPS requests from the binary
RUN apk --no-cache add ca-certificates

# Copy binary and migrations
COPY --from=builder /auth-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8084
CMD ["./auth-service"]