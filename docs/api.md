curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health

/whatsapp-groupe4
├── /api-gateway       # Go: Entrypoint, WebSocket management
├── /user-service      # Go: JWT, User sessions (Postgres + Redis)
├── /message-service   # Go: Message validation, NATS producer
├── /presence-service  # Go: Tracking online/offline (Redis)
├── /database# Go: Consumer (NATS), Batch insert (Postgres)
└── /infra             # Terraform/Kubernetes manifests