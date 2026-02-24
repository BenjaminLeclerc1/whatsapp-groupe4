# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copier les fichiers de dépendances
COPY go.mod go.sum* ./
RUN go mod download

# Copier le code source
COPY . .

# Compiler l'application
RUN CGO_ENABLED=0 GOOS=linux go build -o /presence-service .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Copier le binaire compilé
COPY --from=builder /presence-service .

EXPOSE 8081

CMD ["./user-service"]
