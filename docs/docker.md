# Construire et démarrer tous les services
docker-compose up -d

docker compose up --build

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
docker ps --filter "name=redis"
 docker exec -it whatsapp-redis redis-cli keys "*"


 docker logs whatsapp-groupe4-user-service-1 | grep -i "redis"

 curl -X GET http://127.0.0.1:8081/api/v1/users/ae1d310d-99e5-4933-add8-3166e72cd135
curl -X DELETE http://127.0.0.1:8081/api/v1/users/ae1d310d-99e5-4933-add8-3166e72cd135

check database status
docker ps

Check for the "Missing Table" error
docker exec -it whatsapp-shard-0 psql -U whatsapp -d whatsapp_shard_0 -c "\dt"

# Verify the table exists
docker exec -it whatsapp-shard-0 psql -U whatsapp -d whatsapp_shard_0 -c "\dt"

# Delete the migration history
docker exec -it whatsapp-shard-0 psql -U whatsapp -d whatsapp_shard_0 -c "DROP TABLE IF EXISTS schema_migrations;"

# Clean the existing data (Optional but Recommended)
docker exec -it whatsapp-shard-0 psql -U whatsapp -d whatsapp_shard_0 -c "DROP TABLE IF EXISTS channel_members, channels, chats, chat_participants, messages, users CASCADE;"