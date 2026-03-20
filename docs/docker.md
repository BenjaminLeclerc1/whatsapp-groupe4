# Construire et démarrer tous les services


# Build
docker compose up --build

# Start
docker compose up -d
# Voir les logs
docker-compose logs -f

# Arrêter les services
docker-compose down

# Télécharger les dépendances
make deps

# Compiler tous les services
make build

# Lancer chaque service (dans des terminaux séparés)
make run-gateway
make run-user
make run-message




docker compose down
docker compose build --no-cache
docker compose up


 go build ./message-service/...
 docker compose ps

 docker compose logs -f api-gateway

 docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

 docker logs -f whatsapp-groupe4-user-service-1

 # Rebuild
 docker compose up --build -d

 docker compose up --build -d user-service
 docker compose up --build -d user-service

 docker exec -it whatsapp-redis redis-cli keys "*"

 docker logs whatsapp-groupe4-user-service-1 | grep -i "redis"

 docker logs whatsapp-groupe4-message-service-1 --tail 20