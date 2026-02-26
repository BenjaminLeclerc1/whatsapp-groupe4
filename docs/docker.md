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