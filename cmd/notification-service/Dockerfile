# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Copier les fichiers de dépendances
COPY go.mod go.sum* ./
RUN go mod download

# Copier le code source
COPY . .

# Compiler l'application
RUN CGO_ENABLED=0 GOOS=linux go build -o /notification-service ./cmd/notification-service

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# 1. Copier le binaire compilé
COPY --from=builder /notification-service .

# 2. INDISPENSABLE: Copier le dossier migrations pour créer la table 'notifications'
COPY --from=builder /app/migrations ./migrations

EXPOSE 8083

CMD ["./notification-service"]