# WhatsApp Groupe 4 - API Microservices

API WhatsApp-like construite avec Go et une architecture microservices.

## Architecture

Le projet est composé de 3 microservices :

- **API Gateway** (port 8080) : Point d'entrée principal, route les requêtes vers les services appropriés
- **User Service** (port 8081) : Gestion des utilisateurs (CRUD)
- **Message Service** (port 8082) : Gestion des messages

## Prérequis

- Go 1.22+
- Docker & Docker Compose

## Installation

### Avec Docker (recommandé)

```bash
# Construire et démarrer tous les services
docker-compose up -d

# Voir les logs
docker-compose logs -f

# Arrêter les services
docker-compose down
```

### Sans Docker

```bash
# Télécharger les dépendances
make deps

# Compiler tous les services
make build

# Lancer chaque service (dans des terminaux séparés)
make run-gateway
make run-user
make run-message
```

## Endpoints API

### Health Check

```bash
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
```

### User Service

```bash
# Créer un utilisateur
curl -X POST http://localhost:8081/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username": "john", "email": "john@example.com"}'

# Lister tous les utilisateurs
curl http://localhost:8081/api/v1/users

# Obtenir un utilisateur par ID
curl http://localhost:8081/api/v1/users/{id}

# Mettre à jour un utilisateur
curl -X PUT http://localhost:8081/api/v1/users/{id} \
  -H "Content-Type: application/json" \
  -d '{"username": "john_updated"}'

# Supprimer un utilisateur
curl -X DELETE http://localhost:8081/api/v1/users/{id}
```

### Message Service

```bash
# Créer un message
curl -X POST http://localhost:8082/api/v1/messages \
  -H "Content-Type: application/json" \
  -d '{"sender_id": "user-123", "content": "Hello!", "chat_id": "chat-456"}'

# Lister tous les messages
curl http://localhost:8082/api/v1/messages

# Obtenir les messages d'un chat
curl http://localhost:8082/api/v1/messages/chat/{chatId}

# Supprimer un message
curl -X DELETE http://localhost:8082/api/v1/messages/{id}
```

## Structure du projet

```
.
├── cmd/
│   ├── api-gateway/      # Service API Gateway
│   ├── user-service/     # Service utilisateurs
│   └── message-service/  # Service messages
├── docker-compose.yml    # Orchestration Docker
├── go.mod               # Dépendances Go
├── Makefile             # Commandes utilitaires
└── README.md
```

## Commandes Make

| Commande | Description |
|----------|-------------|
| `make build` | Compile tous les services |
| `make run-gateway` | Lance l'API Gateway |
| `make run-user` | Lance le User Service |
| `make run-message` | Lance le Message Service |
| `make docker-build` | Construit les images Docker |
| `make docker-up` | Démarre les containers |
| `make docker-down` | Arrête les containers |
| `make test` | Lance les tests |
| `make deps` | Télécharge les dépendances |
| `make fmt` | Formate le code |

## Technologies

- **Go 1.22** - Langage de programmation
- **Gin** - Framework HTTP
- **Docker** - Containerisation
- **Alpine Linux** - Image de base légère
