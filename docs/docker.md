# Construire et démarrer tous les services
docker-compose up -d

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