# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copier les fichiers de dépendances
COPY go.mod go.sum* ./
# RUN go mod download
ENV GOPROXY=https://proxy.golang.org,https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.org

# Copier le code source
COPY . .

# Compiler l'application
RUN CGO_ENABLED=0 GOOS=linux go build -o /api-gateway ./cmd/api-gateway

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# 1. Copier le binaire compilé
COPY --from=builder /api-gateway .

# 2. CRITICAL: Copier le dossier migrations
# Sans cette ligne, votre code Go ne trouvera jamais les fichiers SQL
COPY --from=builder /app/migrations ./migrations

# 3. Facultatif: Copier le fichier .env ou config si nécessaire
# COPY --from=builder /app/.env .

EXPOSE 8080

CMD ["./api-gateway"]