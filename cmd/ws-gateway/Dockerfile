FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /ws-gateway ./cmd/ws-gateway

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /ws-gateway .

EXPOSE 8089

CMD ["./ws-gateway"]
