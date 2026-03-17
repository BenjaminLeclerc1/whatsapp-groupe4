# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copier les fichiers de dépendances
COPY go.mod go.sum* ./
RUN go mod download

# Copier le code source
COPY . .

RUN go mod tidy

# Compiler l'application
RUN CGO_ENABLED=0 GOOS=linux go build -o /message-service ./cmd/message-service

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# 1. Copier le binaire compilé
COPY --from=builder /message-service .

# 2. AJOUTEZ CETTE LIGNE : Copier le dossier migrations
# C'est ici que se trouvent vos fichiers SQL pour la table 'messages'
COPY --from=builder /app/migrations ./migrations

EXPOSE 8082

CMD ["./message-service"]