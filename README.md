# WhatsApp Clone - Groupe 4

Application de messagerie instantanée inspirée de WhatsApp, construite avec une architecture microservices en Go et un frontend React.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Frontend React                                  │
│                              (localhost:3000)                                │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           API Gateway (:8080)                                │
│                    Routing, Auth JWT, CORS, Proxy                            │
└─────────────────────────────────────────────────────────────────────────────┘
          │              │              │              │              │
          ▼              ▼              ▼              ▼              ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ Auth Service │ │ User Service │ │  Chat Service│ │Message Service│ │  WS Gateway  │
│    :8084     │ │    :8081     │ │    :8088     │ │    :8082      │ │    :8089     │
└──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘
          │              │              │              │              │
          ▼              ▼              ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Infrastructure                                  │
│     PostgreSQL (Shard 0: :5433, Shard 1: :5434)    │    Redis (:6379)        │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Microservices

| Service               | Port  | Description                                      |
|-----------------------|-------|--------------------------------------------------|
| API Gateway           | 8080  | Point d'entrée, authentification JWT, routing    |
| User Service          | 8081  | Gestion des utilisateurs et profils              |
| Message Service       | 8082  | Envoi et récupération des messages               |
| Notification Service  | 8083  | Gestion des notifications push                   |
| Auth Service          | 8084  | Authentification, JWT, refresh tokens            |
| Channel Service       | 8085  | Gestion des groupes/canaux                       |
| Presence Service      | 8086  | Statut en ligne/hors ligne des utilisateurs      |
| Search Service        | 8087  | Recherche de messages et utilisateurs            |
| Chat Service          | 8088  | Gestion des conversations                        |
| WS Gateway            | 8089  | WebSocket pour la communication temps réel       |

## Technologies

### Backend
- **Go 1.24** - Langage principal
- **Gin** - Framework HTTP
- **JWT** - Authentification stateless
- **pgx** - Driver PostgreSQL haute performance
- **gorilla/websocket** - Communication temps réel
- **golang-migrate** - Migrations de base de données

### Frontend
- **React 19** - Framework UI
- **React Router 7** - Navigation SPA
- **Axios** - Client HTTP

### Infrastructure
- **PostgreSQL 16** - Base de données avec sharding
- **Redis 7** - Cache et pub/sub pour WebSocket
- **Docker & Docker Compose** - Containerisation

## Prérequis

- Docker Desktop (ou Docker Engine + Docker Compose)
- Git

> **Note** : Aucune installation locale de Go, Node.js ou autre n'est nécessaire. Tout s'exécute dans Docker.

## Installation

### 1. Cloner le repository

```bash
git clone https://github.com/votre-org/whatsapp-groupe4.git
cd whatsapp-groupe4
```

### 2. Configurer les variables d'environnement

Créer un fichier `.env` à la racine du projet :

```bash
# Base de données PostgreSQL
POSTGRES_USER=whatsapp
POSTGRES_PASSWORD=whatsapp_secret

# Authentification JWT
JWT_SECRET=votre-secret-jwt-securise-en-production

# Mode applicatif
APP_ENV=dev
```

### 3. Lancer l'application

```bash
# Construire et démarrer tous les services
docker compose up -d --build

# Vérifier que tous les services sont up
docker compose ps
```

### 4. Accéder à l'application

- **Frontend** : http://localhost:3000
- **API Gateway** : http://localhost:8080
- **Health Check** : http://localhost:8080/health

## Commandes utiles

### Gestion des services

```bash
# Démarrer tous les services
docker compose up -d

# Arrêter tous les services
docker compose down

# Redémarrer un service spécifique
docker compose restart auth-service

# Voir les logs d'un service
docker compose logs -f api-gateway

# Voir les logs de tous les services
docker compose logs -f
```

### Build et développement

```bash
# Reconstruire un service après modification
docker compose build auth-service
docker compose up -d auth-service

# Reconstruire tous les services
docker compose build

# Reconstruire sans cache
docker compose build --no-cache
```

### Base de données

```bash
# Accéder à PostgreSQL (Shard 0)
docker exec -it whatsapp-shard-0 psql -U whatsapp -d whatsapp_shard_0

# Accéder à Redis
docker exec -it whatsapp-redis redis-cli
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
whatsapp-groupe4/
├── cmd/                          # Points d'entrée des microservices
│   ├── api-gateway/
│   ├── auth-service/
│   ├── chat-service/
│   ├── message-service/
│   ├── notification-service/
│   ├── user-service/
│   ├── ws-gateway/
│   ├── channel-service/
│   ├── presence-service/
│   └── search-service/
├── internal/                     # Code interne partagé
│   ├── chats/
│   ├── messages/
│   ├── wsgateway/
│   └── pkg/
│       ├── redis/
│       └── sharding/
├── middleware/                   # Middlewares (auth JWT, etc.)
│   └── auth/
├── migrations/                   # Scripts SQL de migration
│   ├── auth-service/
│   ├── chat-service/
│   ├── message-service/
│   ├── notification-service/
│   ├── search-service/
│   └── user-service/
├── frontend/                     # Application React
│   ├── src/
│   │   ├── api/
│   │   ├── components/
│   │   ├── context/
│   │   ├── pages/
│   │   └── services/
│   └── public/
├── loadtest/                     # Tests de charge K6
├── infra/                        # Infrastructure as Code (Terraform)
├── docker-compose.yml
├── go.mod
└── .env
```

## API Endpoints

### Authentification (publics)

```
POST   /api/v1/auth/register    # Inscription
POST   /api/v1/auth/login       # Connexion
POST   /api/v1/auth/refresh     # Rafraîchir le token
```

### Chats (authentifié)

```
GET    /api/v1/chats            # Liste des conversations
POST   /api/v1/chats            # Créer une conversation
GET    /api/v1/chats/:id        # Détails d'une conversation
```

### Messages (authentifié)

```
GET    /api/v1/messages/:chatId # Messages d'une conversation
POST   /api/v1/messages         # Envoyer un message
```

### Utilisateurs (authentifié)

```
GET    /api/v1/users/me         # Profil de l'utilisateur connecté
GET    /api/v1/users/:id        # Profil d'un utilisateur
PUT    /api/v1/users/me         # Modifier son profil
```

### WebSocket

```
WS     /ws?token=<jwt>          # Connexion WebSocket temps réel
```

## Tests de charge

Le projet inclut des scripts K6 pour les tests de performance :

```bash
# Lancer les tests de charge
docker compose -f loadtest/docker-compose.k6.yml up
```

## Priorités du projet

### Performance
- **Keyset pagination** pour des performances O(log n) constantes
- **Connection pooling** PostgreSQL agressif (min=10, max=50)
- **Sharding** de la base de données pour la scalabilité horizontale
- **Redis** pour le cache et la coordination WebSocket

### Sécurité
- Vérification du membership avant tout accès aux données
- Requêtes SQL paramétrées (protection injection SQL)
- Messages d'erreur opaques (pas d'exposition d'erreurs internes)
- Rate limiting sur les endpoints d'écriture
- Validation stricte de tous les inputs

## Déploiement Production

Pour un déploiement en production :

1. **Modifier les secrets** dans `.env` (JWT_SECRET, mots de passe DB)
2. **Configurer HTTPS** via un reverse proxy (Nginx, Traefik)
3. **Activer le mode production** : `APP_ENV=prod`
4. **Infrastructure Terraform** disponible dans `infra/terraform/`

## Contribuer

1. Créer une branche depuis `dev`
2. Faire les modifications
3. Lancer les tests : `docker compose build`
4. Créer une Pull Request vers `dev`

## Équipe

**Groupe 4** - Projet de messagerie instantanée

## Licence

Projet académique - Usage interne uniquement
