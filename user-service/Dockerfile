# 1. Builder Stage
FROM golang:1.26-alpine AS builder
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the binary by pointing to the correct folder
RUN CGO_ENABLED=0 GOOS=linux go build -o /user-service-binary ./user-service

# 2. Final Stage
FROM alpine:3.19
WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /user-service-binary .

EXPOSE 8081
CMD ["./user-service-binary"]