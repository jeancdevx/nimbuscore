# NimbusCore — Kubernetes Setup

## Opciones de cluster local

Elegí una según tu sistema:

| Herramienta  | Recursos | Ideal para                          |
| ------------ | -------- | ----------------------------------- |
| **kind**     | ~500 MB  | Dev liviano, reinicio rápido        |
| **minikube** | ~2 GB    | Dev con addons (ingress, dashboard) |
| **k3d**      | ~500 MB  | Similar a kind, pero con k3s        |
| **k3s**      | ~300 MB  | Cluster "real" en VM o bare-metal   |

---

## 1. Kind (recomendado para dev)

```bash
# Instalar kind
go install sigs.k8s.io/kind@latest

# Crear cluster con ingress
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

# Instalar NGINX Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# Verificar
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s
```

## 2. Instalar dependencias del cluster

```bash
# CRDs de NimbusCore
kubectl apply -f deploy/helm/nimbuscore/crds/

# Cert-manager (para TLS en ingresses)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.1/cert-manager.yaml

# Metric Server (para HPA)
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

---

## 3. Deploy NimbusCore con Helm

```bash
# Agregar repositorios
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update deploy/helm/nimbuscore/

# Instalar (dev)
helm install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.dev.yaml \
  --set secrets.jwt="dev-jwt-secret" \
  --set secrets.oidcClientSecret="dev-oidc-secret" \
  --set secrets.resticPassword="dev-restic-password" \
  --set global.ingress.host="nimbuscore.local"

# Verificar
kubectl get pods -A | grep nimbuscore
kubectl get svc -A | grep nimbuscore
kubectl get ingress -A | grep nimbuscore
```

> **Nota para kind**: Agregá `127.0.0.1 nimbuscore.local` a `/etc/hosts`. El
> ingress escucha en `localhost:80`.

---

## 4. Operator — ciclo de vida de workspaces

El operator es el cerebro que reconcilia workspaces contra Kubernetes:

### Flujo completo

```
POST /api/workspaces  ──→  DB (status: pending)
                              │
                    Operator detecta nuevo Workspace CR
                              │
                    ├── Crea Namespace (si no existe)
                    ├── Crea Pod con DevContainer
                    ├── Crea Service (ClusterIP)
                    └── Crea Ingress (subdominio por puerto)
                              │
                    Status → running
                              │
                    POST /stop ──→ status → stopping
                              │
                    Operator elimina Pod + Service + Ingress
                              │
                    Status → stopped
```

### Ver estado de los CRDs

```bash
# Listar workspaces
kubectl get workspaces -A
kubectl get ws -A

# Ver detalle de un workspace
kubectl describe ws -n <namespace> <workspace-name>

# Listar prebuilds
kubectl get prebuilds -A
kubectl get pb -A

# Ver pods creados para workspaces
kubectl get pods -l app.kubernetes.io/component=workspace
```

### Ejecutar operator local (fuera del cluster)

```bash
cd apps/operator && go run ./cmd/
```

El operator usa el `~/.kube/config` actual. No requiere deploy en el cluster
para dev.

---

## 5. Port forwarding en Kubernetes

Al crear un workspace con puertos:

```json
{
  "ports": [
    { "port": 3000, "protocol": "tcp", "subdomain": "app" },
    { "port": 5173, "protocol": "tcp", "subdomain": "vite" }
  ]
}
```

El operator crea un Ingress por cada puerto con el patrón:

```
{subdomain}-{workspace}.nimbuscore.local
```

Ejemplo:

- `app-mi-dev-container.nimbuscore.local` → :3000
- `vite-mi-dev-container.nimbuscore.local` → :5173

Los podés ver en `status.port_urls` del Workspace CR:

```bash
kubectl get ws <name> -o jsonpath='{.status.port_urls}'
```

---

## 6. Snapshots (Restic)

Los snapshots usan Restic ejecutado dentro del workspace-manager via
`kubectl exec`:

```bash
# Backup manual
curl -X POST /api/workspaces/{id}/snapshots \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"action":"backup"}'

# Restore
curl -X POST /api/workspaces/{id}/snapshots/{snap_id}/restore \
  -H "Authorization: Bearer $TOKEN"
```

Requisitos:

- Bucket S3 configurado (MinIO para dev, AWS S3 para prod)
- `RESTIC_REPOSITORY` y `RESTIC_PASSWORD` configurados en el workspace-manager

### Snapshots automáticos (idle watcher)

El workspace-manager ejecuta un scheduler que:

1. Cada `SNAPSHOT_INTERVAL` (default 60 min) hace backup si el workspace está
   `running`
2. Al hacer `stop`, hace backup automático antes de eliminar el pod

---

## 7. Prebuilds

Los prebuilds construyen imágenes DevContainer con jobs de Kubernetes:

```bash
# Crear prebuild via API
curl -X POST /api/prebuilds \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mi-prebuild",
    "image": "ghcr.io/nimbuscore/prebuilds/mi-prebuild:latest",
    "repo_url": "https://github.com/mi-user/mi-repo",
    "branch": "main",
    "devcontainer_path": ".devcontainer/devcontainer.json"
  }'

# Ver estado
kubectl get jobs -l app.kubernetes.io/component=prebuild
kubectl logs -l job-name=<prebuild-job-name>
```

> El prebuild **no** construye la imagen en k8s — CI/CD se encarga de eso. El
> operator crea un Job `batch/v1` que clona el repo y copia el
> `devcontainer.json`.

---

## 8. Auto-stop y quotas

### Idle timeout

El workspace-manager tiene un watcher que periódicamente:

```bash
# Configuración vía env vars
IDLE_TIMEOUT=20              # minutos sin actividad antes de stop
SNAPSHOT_INTERVAL=60         # minutos entre snapshots automáticos
IDLE_CHECK_INTERVAL=5        # minutos entre chequeos
```

Marca workspaces con la anotación `nimbuscore.io/last-activity` y detiene los
que superan `IDLE_TIMEOUT`.

### Quotas por team

```bash
# Ver quota de un team
curl -s /api/teams/{id}/quota -H "Authorization: Bearer $TOKEN"

# Crear/actualizar quota
curl -X PUT /api/teams/{id}/quota \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"max_workspaces": 5, "max_cpu": "8", "max_memory": "16Gi"}'

# Verificar si hay cupo antes de crear
curl -s /api/teams/{id}/quota/check -H "Authorization: Bearer $TOKEN"
```

La API rechaza creación de workspace con `409 Conflict` si se excede
`max_workspaces`.

---

## 9. Multi-cluster

Registrar clusters adicionales via API:

```bash
# Registrar cluster
curl -X POST /api/admin/clusters \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "prod-us-east",
    "api_endpoint": "https://k8s-prod.example.com",
    "region": "us-east-1",
    "provider": "eks"
  }'

# Listar clusters
curl /api/admin/clusters -H "Authorization: Bearer $TOKEN"

# Eliminar cluster
curl -X DELETE /api/admin/clusters/{id} -H "Authorization: Bearer $TOKEN"
```

Los clusters se almacenan en la tabla `clusters` de PostgreSQL. El operator
determina en qué cluster crear cada workspace según etiquetas (labels).

---

## 10. Limpieza

```bash
# Eliminar todo lo de NimbusCore
helm uninstall nimbuscore

# Eliminar CRDs
kubectl delete crd workspaces.nimbuscore.io
kubectl delete crd prebuilds.nimbuscore.io

# Eliminar cluster kind
kind delete cluster
```
