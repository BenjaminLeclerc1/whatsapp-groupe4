FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire user-service directory
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o user-service .

# Final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/user-service .
EXPOSE 8080
CMD ["./user-service"]