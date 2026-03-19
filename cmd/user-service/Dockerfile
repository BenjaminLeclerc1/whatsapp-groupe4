# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copier les fichiers de dépendances
COPY go.mod go.sum* ./
RUN go mod download

# Copier TOUT le code source (this includes the migrations folder automatically)
COPY . .

# Step 12 was removed because it was redundant and causing the "not found" error

# Compiler l'application
RUN CGO_ENABLED=0 GOOS=linux go build -o /user-service ./cmd/user-service

# Runtime stage
FROM alpine:3.19

# Install ca-certificates (useful for communicating with external APIs)
RUN apk --no-cache add ca-certificates

WORKDIR /app

# IMPORTANT: You need to copy the migrations folder to the final stage 
# so the binary can find them at runtime!
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /user-service .

EXPOSE 8081

CMD ["./user-service"]