# 1. Builder stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build output to a specific name that isn't the same as a source folder
RUN CGO_ENABLED=0 GOOS=linux go build -o /api-gateway-bin ./api-gateway

# 2. Final stage
FROM alpine:3.19
WORKDIR /app
# Copy the renamed binary
COPY --from=builder /api-gateway-bin .
EXPOSE 8080
# Execute the specific binary
CMD ["./api-gateway-bin"]