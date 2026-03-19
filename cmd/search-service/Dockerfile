# # Build stage
# FROM golang:1.24-alpine AS builder

# WORKDIR /app

# # Copier les fichiers de dépendances
# COPY go.mod go.sum* ./
# RUN go mod download

# # Copier le code source
# COPY . .

# # Compiler l'application
# RUN CGO_ENABLED=0 GOOS=linux go build -o /search-service ./cmd/search-service

# # Runtime stage
# FROM alpine:3.19

# WORKDIR /app

# # Copier le binaire compilé
# COPY --from=builder /search-service .

# EXPOSE 8084

# CMD ["./search-service"]

FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /search-service ./cmd/search-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /search-service .
# This is the line that makes your main.go happy:
COPY --from=builder /app/migrations ./migrations
EXPOSE 8084
CMD ["./search-service"]