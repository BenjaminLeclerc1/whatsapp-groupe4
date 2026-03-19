FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./

COPY . .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /channel-service ./cmd/channel-service

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# 1. Copy the binary
COPY --from=builder /channel-service .

# 2. ADD THIS LINE: Copy the migrations folder
COPY --from=builder /app/migrations ./migrations

EXPOSE 8085

CMD ["./channel-service"]