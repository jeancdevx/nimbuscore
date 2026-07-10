# NimbusCore — Runbook

## Stack

| Componente                     | Lenguaje   | Puerto                              |
| ------------------------------ | ---------- | ----------------------------------- |
| API (Chi)                      | Go 1.26    | `:8080`                             |
| Operator (controller-runtime)  | Go 1.26    | `:8080` (metrics), `:8081` (health) |
| Workspace Manager              | Go 1.26    | — (RabbitMQ consumer)               |
| Dashboard (Astro 7 + React 19) | TypeScript | `:5173` (dev) / `:4321` (preview)   |
| PostgreSQL 17                  | —          | `:5432`                             |
| Redis 7                        | —          | `:6379`                             |
| RabbitMQ 4                     | —          | `:5672` / `:15672` (admin)          |
| Dex (OIDC)                     | —          | `:5556`                             |

---

## 1. Primer inicio (local)

### Prerequisitos

- Docker + Docker Compose
- Go 1.26+ (`mise install go@1.26.5`)
- Node.js 22+ (`mise install node@22`)
- pnpm 11+ (`mise install pnpm@11`)
- Task (task runner)

Activar mise:

```bash
eval "$(mise activate zsh)"
```

### Levantar todo

```bash
# 1. Dependencias del frontend
cd web/dashboard && pnpm install && cd ../..

# 2. Iniciar infraestructura (PostgreSQL, Redis, RabbitMQ, Dex)
docker compose up -d

# 3. Buildear y levantar API
docker compose up -d --build api

# 4. En otra terminal — frontend dev
cd web/dashboard && pnpm dev
```

### Servicios

| Servicio    | URL                                      |
| ----------- | ---------------------------------------- |
| API         | `http://localhost:8080`                  |
| Dashboard   | `http://localhost:5173`                  |
| Dex (OIDC)  | `http://localhost:5556/dex`              |
| RabbitMQ UI | `http://localhost:15672` (user:changeme) |

---

## 2. Flujo de autenticación

```
Browser                    Frontend (:5173)           API (:8080)              Dex (:5556)
  │                             │                        │                        │
  │  click "Sign in with SSO"   │                        │                        │
  │────────────────────────────>│                        │                        │
  │                             │  GET /auth/login       │                        │
  │                             │───────────────────────>│                        │
  │                             │  307 → Dex auth page   │                        │
  │                             │<───────────────────────│                        │
  │  302 → http://dex:5556/     │                        │                        │
  │<────────────────────────────│                        │                        │
  │                             │                        │                        │
  │──── Login con credenciales ─────────────────────────────────────────────────>│
  │                             │                        │                        │
  │  302 → http://localhost:5173/?code=xxx               │                        │
  │<────────────────────────────────────────────────────────────────────────────│
  │                             │                        │                        │
  │  App.tsx lee `code` param   │                        │                        │
  │  fetch /auth/callback?code=x│                        │                        │
  │────────────────────────────>│  proxy (:5173 → :8080) │                        │
  │                             │───────────────────────>│                        │
  │                             │  {token, user_id}      │  Exchange code → token │
  │                             │<───────────────────────│──────────────────────>│
  │                             │                        │                        │
  │  Guarda token en localStorage                        │                        │
  │  window.history.replaceState('/',)                   │                        │
  │  setAuthenticated(true)                              │                        │
```

### Credenciales de prueba (Dex)

| Campo      | Valor                 |
| ---------- | --------------------- |
| Email      | `admin@nimbuscore.io` |
| Contraseña | `admin123`            |
| Username   | `admin`               |

> **Nota**: Dex usa SQLite (`deploy/dex/config.yaml`). Los datos persisten
> dentro del container. Si necesitás resetear, borrá el container y recreate.

---

## 3. Dashboard — uso guiado

### Login

Ir a `http://localhost:5173/` → click **"Sign in with SSO"** → login en Dex.

### Página principal

Muestra:

- Cantidad de workspaces **Running** / **Total**
- Nombre del usuario autenticado
- Botón **"New Workspace"**

### Crear workspace

Click **"New Workspace"** → formulario:

| Campo      | Descripción                                                            |
| ---------- | ---------------------------------------------------------------------- |
| Name       | Nombre del workspace                                                   |
| Image      | Imagen DevContainer (default: `ghcr.io/nimbuscore/code-server:latest`) |
| Repository | URL del repo a clonar (opcional)                                       |
| Branch     | Rama (default: `main`)                                                 |
| Resources  | CPU, Memoria, Disco, GPU                                               |
| Ports      | Puertos a exponer (subdomain, port, protocol)                          |

El workspace se crea con status `pending`. Requiere Kubernetes + operator para
pasar a `running`.

### Admin panel

`http://localhost:5173/admin`

Muestra:

- **Users**: tabla de usuarios registrados
- **Teams**: tabla de equipos (vacío si no hay)

Solo accesible para usuarios con rol `admin` o `team_admin`.

---

## 4. Variables de entorno — compose.yml

| Variable               | Descripción                              | Default                                                                       |
| ---------------------- | ---------------------------------------- | ----------------------------------------------------------------------------- |
| `HOST`                 | Host de escucha de la API                | `0.0.0.0`                                                                     |
| `PORT`                 | Puerto de la API                         | `8080`                                                                        |
| `DATABASE_URL`         | Conexión PostgreSQL                      | `postgres://nimbuscore:nimbuscore@postgresql:5432/nimbuscore?sslmode=disable` |
| `REDIS_URL`            | Conexión Redis                           | `redis://redis:6379`                                                          |
| `RABBITMQ_URL`         | Conexión RabbitMQ                        | `amqp://user:changeme@rabbitmq:5672`                                          |
| `OIDC_ISSUER`          | URL interna del proveedor OIDC           | `http://host.docker.internal:5556/dex`                                        |
| `OIDC_EXTERNAL_ISSUER` | URL externa del proveedor (para browser) | `http://localhost:5556/dex`                                                   |
| `OIDC_CLIENT_ID`       | Client ID OIDC                           | `nimbuscore`                                                                  |
| `OIDC_CLIENT_SECRET`   | Client Secret OIDC                       | `nimbuscore-secret`                                                           |
| `OIDC_REDIRECT_URL`    | Redirect URI del callback                | `http://localhost:5173`                                                       |
| `JWT_SECRET`           | Secreto para firmar JWTs                 | `local-dev-secret-change-in-production`                                       |

---

## 5. Endpoints de la API

### Públicos

```
GET  /health                → {"status":"ok"}
GET  /metrics               → métricas expvar (JSON)
GET  /auth/login            → redirect a Dex (OIDC)
GET  /auth/callback         → callback OIDC, devuelve {token, user_id}
```

### Protegidos (requieren header `Authorization: Bearer <token>`)

#### Usuarios

```
GET  /api/users             → listar usuarios (manage:system)
GET  /api/users/me          → usuario actual
GET  /api/users/{id}        → usuario por ID
PATCH /api/users/{id}/role  → cambiar rol (manage:system)
```

#### Teams

```
GET    /api/teams           → listar teams
POST   /api/teams           → crear team (manage:teams)
GET    /api/teams/{id}      → team por ID
PATCH  /api/teams/{id}      → actualizar team (manage:teams)
DELETE /api/teams/{id}      → eliminar team (manage:teams)
```

#### Workspaces

```
GET    /api/workspaces              → listar mis workspaces
POST   /api/workspaces              → crear workspace
GET    /api/workspaces/{id}         → workspace por ID
DELETE /api/workspaces/{id}         → eliminar workspace
POST   /api/workspaces/{id}/start   → iniciar workspace
POST   /api/workspaces/{id}/stop    → detener workspace
POST   /api/workspaces/{id}/heartbeat → heartbeat anti-idle
```

#### Snapshots

```
GET    /api/workspaces/{wid}/snapshots        → listar snapshots
POST   /api/workspaces/{wid}/snapshots        → crear snapshot manual
GET    /api/workspaces/{wid}/snapshots/{id}   → snapshot por ID
POST   /api/workspaces/{wid}/snapshots/{id}/restore → restaurar snapshot
```

#### Prebuilds

```
GET    /api/prebuilds          → listar prebuilds
POST   /api/prebuilds          → crear prebuild
GET    /api/prebuilds/{id}     → prebuild por ID
DELETE /api/prebuilds/{id}     → eliminar prebuild
```

#### Quotas

```
GET   /api/teams/{id}/quota        → ver quota + uso actual
PUT   /api/teams/{id}/quota        → crear/actualizar quota (manage:quotas)
DELETE /api/teams/{id}/quota       → eliminar quota
GET   /api/teams/{id}/quota/check  → verificar si hay cupo
```

#### Billing

```
GET /api/teams/{id}/billing?since=2025-01-01T00:00:00Z&until=2025-12-31T23:59:59Z
```

Requiere permiso `view:billing`.

#### Admin (manage:system)

```
GET    /api/admin/audit-log       → log de auditoría (read:audit_log)
GET    /api/admin/clusters        → listar clusters registrados
POST   /api/admin/clusters        → registrar cluster
DELETE /api/admin/clusters/{id}   → eliminar cluster
```

---

## 6. Roles y permisos

| Rol          | Permisos                                                                 |
| ------------ | ------------------------------------------------------------------------ |
| `admin`      | manage:system, read:audit_log, manage:teams, manage:quotas, view:billing |
| `team_admin` | manage:teams, manage:quotas, view:billing                                |
| `user`       | (ninguno especial)                                                       |

Cambiar rol de un usuario:

```bash
curl -X PATCH /api/users/{id}/role \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role": "admin"}'
```

---

## 7. Flujo de creación de workspace (curl)

```bash
TOKEN="<jwt>"
API="http://localhost:8080"

# 1. Ver mi usuario
curl -s $API/api/users/me -H "Authorization: Bearer $TOKEN" | jq

# 2. Crear workspace (queda pending hasta que operator lo reconcilie)
curl -s -X POST $API/api/workspaces \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mi-dev-container",
    "image": "ghcr.io/nimbuscore/code-server:latest",
    "repo_url": "https://github.com/mi-user/mi-repo",
    "branch": "main",
    "resources": { "cpu": "1", "memory": "2Gi", "disk": "10Gi" },
    "ports": [
      {"port": 3000, "protocol": "tcp", "subdomain": "app"},
      {"port": 5173, "protocol": "tcp", "subdomain": "vite"}
    ]
  }' | jq

# 3. Ver todos los workspaces
curl -s $API/api/workspaces -H "Authorization: Bearer $TOKEN" | jq

# 4. Heartbeat (evita idle timeout)
curl -s -X POST "$API/api/workspaces/$WS_ID/heartbeat" \
  -H "Authorization: Bearer $TOKEN" | jq
```

---

## 8. Operador (Kubernetes)

Ver [`docs/KUBERNETES_SETUP.md`](./KUBERNETES_SETUP.md) para instalación
detallada.

```bash
# 1. Instalar CRDs
kubectl apply -f deploy/helm/nimbuscore/crds/

# 2. Correr operador local (apunta al kubeconfig actual)
cd apps/operator && go run ./cmd/

# 3. Ver workspaces como CRDs
kubectl get workspaces -A
kubectl get ws -A

# 4. Ver prebuilds
kubectl get prebuilds -A
kubectl get pb -A
```

---

## 9. Deploy con Helm

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update deploy/helm/nimbuscore/

helm upgrade --install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.dev.yaml \
  --set secrets.jwt="$(openssl rand -base64 32)" \
  --set secrets.oidcClientSecret="$(openssl rand -base64 32)" \
  --set secrets.resticPassword="$(openssl rand -base64 32)"
```

---

## 10. ArgoCD (GitOps)

```bash
kubectl apply -f deploy/helm/nimbuscore/argocd/
```

---

## 11. Monitoreo

| Herramienta       | URL                                                       |
| ----------------- | --------------------------------------------------------- |
| API metrics       | `GET http://localhost:8080/metrics`                       |
| Operator metrics  | `http://localhost:8080/metrics` (controller-runtime)      |
| RabbitMQ admin    | `http://localhost:15672` (user:changeme)                  |
| Grafana           | Dashboard: NimbusCore Overview (si instalado)             |
| Prometheus alerts | `deploy/helm/nimbuscore/dashboards/prometheus-rules.yaml` |

### Métricas expvar

| Nombre                             | Tipo     | Descripción                   |
| ---------------------------------- | -------- | ----------------------------- |
| `nimbuscore_requests_total`        | contador | Total de requests HTTP        |
| `nimbuscore_active_workspaces`     | gauge    | Workspaces con status running |
| `nimbuscore_workspace_hours_total` | contador | Horas acumuladas de workspace |
| `nimbuscore_snapshots_total`       | contador | Snapshots realizados          |
| `nimbuscore_prebuilds_total`       | contador | Prebuilds ejecutados          |
| `nimbuscore_errors_total`          | contador | Errores internos              |

---

## 12. CI/CD (GitHub Actions)

Ver `.github/workflows/ci.yml`.

| Job        | Dispara en                     | Qué hace                                                       |
| ---------- | ------------------------------ | -------------------------------------------------------------- |
| `quality`  | PR a develop/production        | go vet, lint, test, prettier, oxlint, build frontend           |
| `security` | PR a develop/production        | Trivy filesystem + IaC scan                                    |
| `docker`   | Push a main                    | Build + push imágenes a GHCR + Trivy image scan + SARIF upload |
| `prebuild` | Commit con mensaje `prebuild:` | Build de imagen DevContainer                                   |

---

## 13. Arquitectura de mensajes (RabbitMQ)

```
Exchange: nimbuscore.workspace (topic, durable)

Eventos:
  workspace.create    → workspace-manager crea namespace
  workspace.start     → workspace-manager restaura snapshot (si existe)
  workspace.stop      → workspace-manager hace backup + elimina pod
  workspace.delete    → workspace-manager elimina namespace
  workspace.snapshot  → workspace-manager backup/restore via Restic
  workspace.timeout   → workspace-manager elimina pod por idle
  workspace.prebuild  → workspace-manager procesa prebuild (log)
```

---

## 14. Troubleshooting

```bash
# Ver logs de la API
docker compose logs -f api

# Ver logs de PostgreSQL
docker compose logs -f postgresql

# Ver colas de RabbitMQ
docker compose exec rabbitmq rabbitmqctl list_queues

# Ver workspaces en k8s
kubectl describe ws <workspace-name>

# Ver jobs de prebuild
kubectl get jobs -A

# Resetear base de datos (pierde todos los datos)
docker compose down -v && docker compose up -d

# Problemas de OIDC — verificar que Dex responde
curl http://localhost:5556/dex/.well-known/openid-configuration

# Verificar que el proxy de Vite funciona
curl -s http://localhost:5173/api/health

# Token expirado — loguearse de nuevo
# (borrar nimbuscore_token de localStorage y recargar)
```
