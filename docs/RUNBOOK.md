# NimbusCore — Runbook

## Stack

| Component                      | Lenguaje   | Puerto                              |
| ------------------------------ | ---------- | ----------------------------------- |
| API (Chi)                      | Go 1.26    | `:8080`                             |
| Operator (controller-runtime)  | Go 1.26    | `:8080` (metrics), `:8081` (health) |
| Workspace Manager              | Go 1.26    | — (RabbitMQ consumer)               |
| Dashboard (Astro 7 + React 19) | TypeScript | `:4321` (dev)                       |
| PostgreSQL 17                  | —          | `:5432`                             |
| Redis 7                        | —          | `:6379`                             |
| RabbitMQ 4                     | —          | `:5672` / `:15672` (admin)          |
| Dex (OIDC)                     | —          | `:5556`                             |

---

## 1. Primer inicio (local)

```bash
# 1. Activar mise
eval "$(mise activate zsh)"

# 2. Instalar dependencias
pnpm install

# 3. Iniciar infraestructura (PostgreSQL, Redis, RabbitMQ, Dex)
docker compose up -d

# 4. Inicializar base de datos (sql init ya corre automático con compose)
#    Si querés hacerlo manual:
#   docker compose exec -T postgresql psql -U nimbuscore nimbuscore < deploy/db/init.sql

# 5. Iniciar API
cd apps/api && go run ./cmd/

# 6. En otra terminal — iniciar Dashboard (opcional)
cd web/dashboard && pnpm dev
```

La API queda escuchando en `http://localhost:8080`.

---

## 2. Comandos útiles (Taskfile)

```bash
task               # Listar todas las tasks disponibles
task ci            # CI completo (vet + lint + test + build + format + web)
task go:test       # Tests Go con race detector
task go:build      # Compilar todos los binarios
task format        # Formatear todo con Prettier
task web:dev       # Servidor de desarrollo Astro
```

---

## 3. Endpoints de la API

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
GET  /api/users             → listar usuarios (admin)
GET  /api/users/me          → usuario actual
GET  /api/users/{id}        → usuario por ID
PATCH /api/users/{id}/role  → cambiar rol (admin, requiere manage:system)
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

#### Quotas (por team)

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

#### Admin (manage:system)

```
GET    /api/admin/audit-log       → log de auditoría (read:audit_log)
GET    /api/admin/clusters        → listar clusters registrados
POST   /api/admin/clusters        → registrar cluster
DELETE /api/admin/clusters/{id}   → eliminar cluster
```

---

## 4. Flujo de creación de workspace (extremo a extremo)

```bash
# Variables
TOKEN="<jwt obtenido de /auth/callback>"
API="http://localhost:8080"

# 1. Ver mi usuario
curl -s $API/api/users/me -H "Authorization: Bearer $TOKEN" | jq

# 2. Crear un workspace
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

# 3. Iniciar workspace
curl -s -X POST "$API/api/workspaces/$WS_ID/start" \
  -H "Authorization: Bearer $TOKEN" | jq

# 4. Ver estado
curl -s "$API/api/workspaces/$WS_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.status'

# 5. Detener
curl -s -X POST "$API/api/workspaces/$WS_ID/stop" \
  -H "Authorization: Bearer $TOKEN" | jq

# 6. Crear snapshot manual
curl -s -X POST "$API/api/workspaces/$WS_ID/snapshots" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"backup"}' | jq

# 7. Heartbeat (evita idle timeout)
curl -s -X POST "$API/api/workspaces/$WS_ID/heartbeat" \
  -H "Authorization: Bearer $TOKEN" | jq
```

---

## 5. Operador (Kubernetes)

Requiere un cluster k8s (kind, minikube, o real).

```bash
# 1. Instalar CRDs
kubectl apply -f deploy/helm/nimbuscore/crds/

# 2. Ver CRDs instalados
kubectl get crd | grep nimbuscore

# 3. Correr operador local (apunta al kubeconfig actual)
cd apps/operator && go run ./cmd/

# 4. Ver workspaces como CRDs
kubectl get workspaces -A
kubectl get ws -A

# 5. Ver prebuilds
kubectl get prebuilds -A
kubectl get pb -A
```

---

## 6. Deploy con Helm

```bash
# Repositorio de dependencias (Bitnami)
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update deploy/helm/nimbuscore/

# Instalar en dev
helm install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.dev.yaml \
  --set secrets.jwt="mi-jwt-secret" \
  --set secrets.oidcClientSecret="mi-oidc-secret"

# Instalar en prod
helm upgrade --install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.prod.yaml \
  --set secrets.jwt="$(openssl rand -base64 32)" \
  --set secrets.oidcClientSecret="$(openssl rand -base64 32)" \
  --set secrets.resticPassword="$(openssl rand -base64 32)"

# Ver estado
helm status nimbuscore
kubectl get pods
kubectl get svc
kubectl get ingress
```

---

## 7. ArgoCD (GitOps)

```bash
# Aplicar project + application-set
kubectl apply -f deploy/helm/nimbuscore/argocd/

# Ver en UI ArgoCD
argocd app list | grep nimbuscore
```

---

## 8. Monitoreo

| Herramienta            | URL                                                                          |
| ---------------------- | ---------------------------------------------------------------------------- |
| API metrics            | `GET http://localhost:8080/metrics`                                          |
| Operator metrics       | `GET http://localhost:8080/metrics` (controller-runtime)                     |
| RabbitMQ admin         | `http://localhost:15672` (user:changeme)                                     |
| Grafana (si instalado) | Dashboards: NimbusCore Overview                                              |
| Prometheus alerts      | PrometheusRules en `deploy/helm/nimbuscore/dashboards/prometheus-rules.yaml` |

### Alertas configuradas

| Alerta                     | Severidad | Condición                     |
| -------------------------- | --------- | ----------------------------- |
| API Down                   | critical  | `up == 0` por 1m              |
| Operator Down              | critical  | `up == 0` por 2m              |
| High Reconciliation Errors | warning   | `rate(errors) > 0.1` por 5m   |
| High Latency               | warning   | p95 > 1s por 5m               |
| Replica Mismatch           | warning   | disponibles < deseados        |
| Quota Exceeded             | warning   | uso > 90% por 10m             |
| High Error Rate            | warning   | tasa errores > 5%             |
| Vulnerability Found        | critical  | vulnerabilidades críticas > 0 |

---

## 9. CI/CD (GitHub Actions)

Archivo: `.github/workflows/ci.yml`

| Job        | Dispara en                     | Qué hace                                                       |
| ---------- | ------------------------------ | -------------------------------------------------------------- |
| `quality`  | PR a develop/production        | go vet, lint, test, prettier, oxlint, build frontend           |
| `security` | PR a develop/production        | Trivy filesystem + IaC scan                                    |
| `docker`   | Push a main                    | Build + push imágenes a GHCR + Trivy image scan + SARIF upload |
| `prebuild` | Commit con mensaje `prebuild:` | Build de imagen DevContainer                                   |

---

## 10. Arquitectura de mensajes (RabbitMQ)

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

## 11. Quotas por team

- Al crear workspace con `team_id`, la API verifica `quotas` y rechaza con 409
  si se excede `max_workspaces`
- `GET /api/teams/{id}/quota/check` devuelve uso actual vs máximo
- El idle watcher detiene workspaces inactivos después de `IDLE_TIMEOUT` minutos
  (default 20)
- El heartbeat endpoint (`POST /workspaces/{id}/heartbeat`) actualiza
  `last_activity`

---

## 12. Troubleshooting

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

# Ver snapshots programados (logs workspace-manager)
docker compose logs -f workspace-manager 2>/dev/null || echo "Corre en k8s"
```
