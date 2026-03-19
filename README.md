# WhatsApp Groupe 4 - API Microservices

API WhatsApp-like construite avec Go et une architecture microservices.

## Architecture

Le projet est composé de 5 microservices :

- **API Gateway** (port 8080) : Point d'entrée principal, route les requêtes vers les services appropriés
- **User Service** (port 8081) : Gestion des utilisateurs (CRUD)
- **Message Service** (port 8082) : Gestion des messages
- **Presence Service** (port 8083) : Gestion de la présence des utilisateurs (online/offline/typing)
- **Search Service** (port 8084) : Recherche de messages dans les conversations

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
curl http://localhost:8083/health
curl http://localhost:8084/health
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

### Presence Service

```bash
# Marquer un utilisateur comme en ligne
curl -X POST http://localhost:8083/api/v1/presence/online \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-123"}'

# Marquer un utilisateur comme hors ligne
curl -X POST http://localhost:8083/api/v1/presence/offline \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-123"}'

# Indiquer qu'un utilisateur est en train de taper
curl -X POST http://localhost:8083/api/v1/presence/typing \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-123", "chat_id": "chat-456", "typing": true}'

# Obtenir la présence d'un utilisateur
curl http://localhost:8083/api/v1/presence/user-123

# Obtenir la présence de plusieurs utilisateurs
curl -X GET http://localhost:8083/api/v1/presence/bulk \
  -H "Content-Type: application/json" \
  -d '{"user_ids": ["user-123", "user-456", "user-789"]}'

# Mettre à jour la présence (générique)
curl -X POST http://localhost:8083/api/v1/presence/update \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-123", "status": "online"}'
```

**Fonctionnalités du Presence Service :**
- 🟢 Statuts disponibles : `online`, `offline`, `typing`
- ⏱️ Timeout automatique après 5 minutes d'inactivité (passage en offline)
- ⌨️ Timeout du statut "typing" après 10 secondes
- 📊 Récupération de la présence en masse (bulk)
- 🔄 Worker background pour gérer les timeouts automatiquement

### Search Service

```bash
# Indexer un message pour la recherche
curl -X POST http://localhost:8084/api/v1/search/index \
  -H "Content-Type: application/json" \
  -d '{"id": "msg-123", "sender_id": "user-456", "content": "Bonjour, comment vas-tu?", "chat_id": "chat-789"}'

# Rechercher des messages (tous les chats)
curl "http://localhost:8084/api/v1/search/messages?q=bonjour&limit=10"

# Rechercher dans un chat spécifique
curl "http://localhost:8084/api/v1/search/messages/chat/chat-789?q=bonjour"

# Rechercher les messages d'un utilisateur
curl "http://localhost:8084/api/v1/search/messages/user/user-456?q=bonjour"

# Supprimer un message de l'index
curl -X DELETE http://localhost:8084/api/v1/search/index/msg-123

# Obtenir les statistiques de l'index
curl http://localhost:8084/api/v1/search/stats
```

**Fonctionnalités du Search Service :**
- 🔍 Recherche full-text dans les messages
- 📝 Index inversé pour des recherches rapides
- 🎯 Filtrage par chat ou par utilisateur
- 💯 Score de pertinence pour chaque résultat
- ✨ Highlight automatique des extraits pertinents
- 🔤 Normalisation et tokenization intelligente
- 🚫 Filtrage des mots vides (stop words) en français et anglais
- 📊 Statistiques sur l'index de recherche

## Structure du projet

```
.
├── cmd/
│   ├── api-gateway/      # Service API Gateway
│   ├── user-service/     # Service utilisateurs
│   ├── message-service/  # Service messages
│   ├── presence-service/ # Service de présence
│   └── search-service/   # Service de recherche
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
| `make run-presence` | Lance le Presence Service |
| `make run-search` | Lance le Search Service |
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
