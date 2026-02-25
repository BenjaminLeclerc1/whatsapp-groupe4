# Build stage
FROM golang:1.23-alpine AS builder

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

# Copier le binaire compilé
COPY --from=builder /message-service .

EXPOSE 8082

CMD ["./message-service"]
