FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum* ./
RUN go mod download

# Copy the entire source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /channel-service ./cmd/channel-service

# Final Stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /channel-service .

# --- IMPORTANT: Copy migrations folder to the final image ---
COPY --from=builder /app/migrations ./migrations

EXPOSE 8085

CMD ["./channel-service"]