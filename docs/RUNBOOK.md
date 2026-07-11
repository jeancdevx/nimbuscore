# NimbusCore — Runbook

## Arquitectura

```
┌─────────────────────────────────────────────────────────────────┐
│                     DOCKER COMPOSE (dev rápido)                  │
│                                                                  │
│  API:8080  PostgreSQL:5432  Redis:6379  RabbitMQ:5672  Dex:5556  │
│                                                                  │
│  ⚠️ Solo para desarrollo de la API. NO crea workspaces reales.   │
│  Para workspaces necesitás el stack Kubernetes (abajo).          │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     KUBERNETES (kind)                            │
│                                                                  │
│  API → PostgreSQL → RabbitMQ → workspace-manager → CRD → operator→ Pod
│                                                                  │
│  El workspace-manager recibe eventos de RabbitMQ y crea CRDs.    │
│  El operator detecta CRDs y provisiona Pods/Services/Ingresses.  │
└─────────────────────────────────────────────────────────────────┘
```

| Componente                     | Stack      | Puerto                                   |
| ------------------------------ | ---------- | ---------------------------------------- |
| API (Chi)                      | K8s        | `:8080` (via port-forward)               |
| Operator (controller-runtime)  | K8s        | — (pod interno)                          |
| Workspace Manager              | K8s        | — (RabbitMQ consumer)                    |
| Dashboard (Astro 7 + React 19) | Tu máquina | `:5173` (dev)                            |
| PostgreSQL 17                  | K8s        | — (pod interno, subchart Bitnami)        |
| Redis 7                        | K8s        | — (pod interno, subchart Bitnami)        |
| RabbitMQ 4                     | K8s        | — (pod aparte con imagen oficial)        |
| Dex (OIDC)                     | Docker     | `:5556` (corre fuera de K8s, en compose) |

---

## 1. Primer inicio — full stack (K8s + Dex)

### Paso 1: Dex (OIDC)

Dex corre fuera de K8s porque necesita ser accesible desde el browser en
`localhost:5556`:

```bash
# Solo Dex, sin la API de compose
docker compose up -d dex
```

Esperar a que Dex responda:

```bash
curl http://localhost:5556/dex/.well-known/openid-configuration
# → {"issuer":"http://localhost:5556/dex", ...}
```

### Paso 2: Kind + Helm

```bash
# Crear cluster kind (si no existe)
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      - hostPort: 80
        containerPort: 80
        protocol: TCP
      - hostPort: 443
        containerPort: 443
        protocol: TCP
EOF

# NGINX Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller --timeout=120s

# RabbitMQ (imagen oficial, Bitnami no tiene imágenes públicas)
kubectl apply -f deploy/helm/standalone-rabbitmq.yaml

# Construir imágenes custom (ghcr.io/nimbuscore/* no existen en registry)
docker build -f deploy/Dockerfile.api -t ghcr.io/nimbuscore/nimbuscore-api:latest .
docker build -f deploy/Dockerfile.operator -t ghcr.io/nimbuscore/nimbuscore-operator:latest .
docker build -f deploy/Dockerfile.workspace-manager -t ghcr.io/nimbuscore/nimbuscore-workspace-manager:latest .

kind load docker-image \
  ghcr.io/nimbuscore/nimbuscore-api:latest \
  ghcr.io/nimbuscore/nimbuscore-operator:latest \
  ghcr.io/nimbuscore/nimbuscore-workspace-manager:latest

# Instalar Helm chart
helm dependency update deploy/helm/nimbuscore/
helm upgrade --install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.dev.yaml \
  --set secrets.jwt="dev-jwt-secret" \
  --set secrets.oidcClientSecret="dev-oidc-secret" \
  --set secrets.resticPassword="dev-restic-password" \
  --set global.ingress.host="nimbuscore.local"
```

### Paso 3: Exponer la API

```bash
kubectl port-forward svc/nimbuscore-api 8080:80
```

> Dejá este comando corriendo en una terminal.

### Paso 4: Frontend

```bash
cd web/dashboard && pnpm install && pnpm dev
```

### Paso 5: Verificar

```bash
curl -s http://localhost:8080/health
kubectl get pods -n default
```

---

## 2. Cada vez que cambies código

### Solo cambié frontend

```bash
cd web/dashboard && pnpm dev
# El hot reload de Vite se encarga
```

### Cambié apps/api

```bash
docker build -f deploy/Dockerfile.api -t ghcr.io/nimbuscore/nimbuscore-api:latest .
kind load docker-image ghcr.io/nimbuscore/nimbuscore-api:latest
kubectl rollout restart deployment nimbuscore-api
```

### Cambié apps/operator

```bash
docker build -f deploy/Dockerfile.operator -t ghcr.io/nimbuscore/nimbuscore-operator:latest .
kind load docker-image ghcr.io/nimbuscore/nimbuscore-operator:latest
kubectl rollout restart deployment nimbuscore-operator
```

### Cambié apps/workspace-manager

```bash
cd apps/workspace-manager && go mod tidy  # si cambiaste imports
docker build -f deploy/Dockerfile.workspace-manager -t ghcr.io/nimbuscore/nimbuscore-workspace-manager:latest .
kind load docker-image ghcr.io/nimbuscore/nimbuscore-workspace-manager:latest
kubectl rollout restart deployment nimbuscore-workspace-manager
```

### Cambié valores de Helm o templates

```bash
helm upgrade --install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.dev.yaml \
  --set secrets.jwt="dev-jwt-secret" \
  --set secrets.oidcClientSecret="dev-oidc-secret" \
  --set secrets.resticPassword="dev-restic-password" \
  --set global.ingress.host="nimbuscore.local"
```

---

## 3. Flujo de autenticación

```
Browser (:5173)         API (:8080)              Dex (:5556)
  │                        │                        │
  │  click "Sign in"       │                        │
  │───────────────────────>│                        │
  │                        │  redirect a Dex        │
  │<───────────────────────│                        │
  │─────────────────────────────────────────────────>│
  │                        │                        │
  │  login con credenciales                          │
  │<─────────────────────────────────────────────────│
  │  (?code=xxx)           │                        │
  │                        │                        │
  │  GET /auth/callback    │                        │
  │───────────────────────>│                        │
  │                        │  exchange code → token │
  │                        │───────────────────────>│
  │  {token, user_id}      │                        │
  │<───────────────────────│                        │
```

### Credenciales Dex

| Campo      | Valor                 |
| ---------- | --------------------- |
| Email      | `admin@nimbuscore.io` |
| Contraseña | `admin123`            |

---

## 4. Crear un workspace (desde el dashboard)

1. Abrir `http://localhost:5173`
2. Sign in con Dex
3. Click **"NEW WORKSPACE"**
4. Completar:

| Campo    | Ejemplo                        | Obligatorio |
| -------- | ------------------------------ | ----------- |
| Name     | `mi-dev`                       | ✅          |
| Image    | `codercom/code-server:latest`  | ❌          |
| CPU      | `1`                            | ❌          |
| RAM      | `2Gi`                          | ❌          |
| Disk     | `10Gi`                         | ❌          |
| Repo URL | `https://github.com/user/repo` | ❌          |
| Branch   | `main`                         | ❌          |
| Ports    | `3000` / HTTP / `app`          | ❌          |

5. Click **"CREATE WORKSPACE"**

### Qué pasa después (flujo E2E)

```
1. POST /api/workspaces
   ├── API valida input
   ├── API guarda en PostgreSQL (status: "pending")
   └── API publica evento en RabbitMQ

2. workspace-manager recibe evento
   └── Crea Workspace CRD en K8s (nimbuscore.io/v1alpha1)

3. Operator detecta CRD nuevo
   ├── Fase "pending": Crea Namespace
   ├── Fase "building": Crea PVC + Service + Pod + Ingress
   └── Fase "running": Workspace listo

4. Workspace accesible en:
   ├── https://<name>.nimbuscore.local (code-server)
   └── https://<subdomain>-<name>.nimbuscore.local (cada puerto)
```

### Ver el estado

```bash
kubectl get workspaces.nimbuscore.io -A
kubectl describe workspace mi-dev
kubectl get pods -A | grep mi-dev
```

---

## 5. Comandos de debug

```bash
# Logs
kubectl logs -l app.kubernetes.io/component=api --tail 50
kubectl logs -l app.kubernetes.io/component=workspace-manager --tail 50
kubectl logs -l app.kubernetes.io/component=operator --tail 50

# RabbitMQ
kubectl exec deploy/rabbitmq -- rabbitmqctl list_queues

# PostgreSQL
kubectl exec nimbuscore-postgresql-0 -- env PGPASSWORD=nimbuscore \
  psql -U nimbuscore -d nimbuscore -c "\dt"

# CRDs
kubectl api-resources | grep nimbuscore
kubectl get workspaces.nimbuscore.io -A -o yaml

# Port-forward extra
kubectl port-forward svc/nimbuscore-api 8080:80
```

---

## 6. Troubleshooting

| Síntoma                                    | Causa probable                      | Solución                                                        |
| ------------------------------------------ | ----------------------------------- | --------------------------------------------------------------- |
| `ImagePullBackOff` en pods                 | Imagen no cargada en kind           | `kind load docker-image ghcr.io/nimbuscore/<app>:latest`        |
| Workspace se queda `pending`               | workspace-manager no recibe evento  | `kubectl logs -l app.kubernetes.io/component=workspace-manager` |
| API responde pero workspace no             | Compose API (no K8s) está corriendo | `docker compose down`                                           |
| Error "relation workspaces does not exist" | DB sin schema inicializado          | Eliminar PVC: `kubectl delete pvc data-nimbuscore-postgresql-0` |
| Browser dice "no hay conexión"             | Port-forward no está corriendo      | `kubectl port-forward svc/nimbuscore-api 8080:80`               |
| Login no redirige a Dex                    | Dex no está corriendo               | `docker compose up -d dex`                                      |
| `connection refused` a Redis               | Redis arrancando                    | Esperar 15s, el health check lo reintenta automático            |

---

## 7. En caso de desastre (reset total)

```bash
# Eliminar todo de K8s
helm uninstall nimbuscore
kubectl delete crd workspaces.nimbuscore.io
kubectl delete crd prebuilds.nimbuscore.io
kubectl delete -f deploy/helm/standalone-rabbitmq.yaml

# Eliminar cluster
kind delete cluster

# Eliminar datos de compose
docker compose down -v

# Empezar de nuevo desde la sección 1
```
