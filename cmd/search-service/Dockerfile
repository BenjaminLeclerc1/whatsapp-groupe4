# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copier les fichiers de dépendances
COPY go.mod go.sum* ./
RUN go mod download

# Copier le code source
COPY . .

# Compiler l'application
RUN CGO_ENABLED=0 GOOS=linux go build -o /search-service ./cmd/search-service

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Copier le binaire compilé
COPY --from=builder /search-service .

EXPOSE 8084

CMD ["./search-service"]
