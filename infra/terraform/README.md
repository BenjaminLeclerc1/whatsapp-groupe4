## IaC (Terraform) – Azure

Ce dossier permet de déployer l’infra minimale sur Azure :

- Resource Group
- Log Analytics Workspace
- Azure Container Apps Environment
- Azure Container Registry (admin activé pour simplifier la démo)
- PostgreSQL Flexible Server + 2 bases (`whatsapp_shard_0`, `whatsapp_shard_1`)
- Container Apps : `auth-service`, `chat-service`, `message-service`, `api-gateway`

### Exécution via Docker (sans installer Terraform)

À lancer depuis la racine du repo.

1) Se connecter à Azure :

```powershell
az login
```

2) Créer un fichier `infra/terraform/terraform.tfvars` (NE PAS le commit) :

```hcl
postgres_admin_password = "CHANGE_ME_STRONG_PASSWORD"
jwt_secret              = "CHANGE_ME_JWT_SECRET"

images = {
  auth_service    = "acrwhatsappg4.azurecr.io/auth-service:v1"
  chat_service    = "acrwhatsappg4.azurecr.io/chat-service:v1"
  message_service = "acrwhatsappg4.azurecr.io/message-service:v1"
  api_gateway     = "acrwhatsappg4.azurecr.io/api-gateway:v1"
}
```

3) Init / Plan :

```powershell
docker run --rm -it `
  -v "${PWD}/infra/terraform:/work" `
  -w /work `
  -e ARM_USE_CLI=true `
  hashicorp/terraform:1.9.8 init

docker run --rm -it `
  -v "${PWD}/infra/terraform:/work" `
  -w /work `
  -e ARM_USE_CLI=true `
  hashicorp/terraform:1.9.8 plan
```

4) Apply :

```powershell
docker run --rm -it `
  -v "${PWD}/infra/terraform:/work" `
  -w /work `
  -e ARM_USE_CLI=true `
  hashicorp/terraform:1.9.8 apply
```

### Notes

- Les images doivent exister dans l’ACR avant le `apply` (build/push via `docker build` + `docker push`).
- Pour la sécurité en prod : remplacer l’ACR admin + secrets par Managed Identity + Key Vault.

