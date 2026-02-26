FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./

COPY . .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /channel-service ./cmd/channel-service

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /channel-service .

EXPOSE 8085

CMD ["./channel-service"]
