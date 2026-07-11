# NimbusCore — Kubernetes Setup

Guía paso a paso para levantar NimbusCore completo en kind.

---

## 0. Prerequisitos

```bash
# Herramientas necesarias
docker --version
kind --version          # go install sigs.k8s.io/kind@latest
kubectl --version       # o instalar con mise: mise install kubectl
helm --version          # mise install helm o brew install helm
mise --version
```

Activar mise:

```bash
eval "$(mise activate zsh)"
```

---

## 1. Crear cluster kind con Ingress

```bash
# Crear cluster con puertos 80/443 mapeados
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

# Esperar a que esté listo
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s
```

---

## 2. Construir imágenes custom y cargarlas en kind

Las imágenes `ghcr.io/nimbuscore/*` no existen en ningún registry. Hay que
construirlas localmente:

```bash
docker build -f deploy/Dockerfile.api -t ghcr.io/nimbuscore/nimbuscore-api:latest .
docker build -f deploy/Dockerfile.operator -t ghcr.io/nimbuscore/nimbuscore-operator:latest .
docker build -f deploy/Dockerfile.workspace-manager -t ghcr.io/nimbuscore/nimbuscore-workspace-manager:latest .

kind load docker-image \
  ghcr.io/nimbuscore/nimbuscore-api:latest \
  ghcr.io/nimbuscore/nimbuscore-operator:latest \
  ghcr.io/nimbuscore/nimbuscore-workspace-manager:latest
```

> **Cada vez que cambies código en apps/api, apps/operator o
> apps/workspace-manager**:
>
> ```bash
> docker build -f deploy/Dockerfile.<app> -t ghcr.io/nimbuscore/nimbuscore-<app>:latest .
> kind load docker-image ghcr.io/nimbuscore/nimbuscore-<app>:latest
> kubectl rollout restart deployment nimbuscore-<app>
> ```

---

## 3. RabbitMQ (imagen oficial)

Bitnami ya no publica imágenes `bitnami/rabbitmq` en Docker Hub. Desplegamos
RabbitMQ aparte con la imagen oficial:

```bash
kubectl apply -f deploy/helm/standalone-rabbitmq.yaml
```

Esto crea un deployment `rabbitmq` con usuario `user` / contraseña `changeme`.

---

## 4. Instalar NimbusCore con Helm

```bash
# Actualizar dependencias (postgresql, redis)
helm dependency update deploy/helm/nimbuscore/

# Instalar (upgrade --install es idempotente)
helm upgrade --install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.dev.yaml \
  --set secrets.jwt="dev-jwt-secret" \
  --set secrets.oidcClientSecret="dev-oidc-secret" \
  --set secrets.resticPassword="dev-restic-password" \
  --set global.ingress.host="nimbuscore.local"

# Verificar que todo está Running
kubectl get pods -n default
```

Deberías ver (puede tardar 30s en estabilizarse):

```
nimbuscore-api-*                   1/1     Running
nimbuscore-operator-*              1/1     Running
nimbuscore-postgresql-0            1/1     Running
nimbuscore-redis-master-0          1/1     Running
nimbuscore-redis-replicas-*        1/1     Running
nimbuscore-workspace-manager-*     1/1     Running
rabbitmq-*                         1/1     Running
```

> **Si ves `ImagePullBackOff`**: puede que las imágenes no se cargaron bien en
> kind. Repetí el paso 2 y luego
> `kubectl rollout restart deployment nimbuscore-api nimbuscore-operator nimbuscore-workspace-manager`

---

## 5. Exponer la API localmente

La API corre dentro del cluster como `ClusterIP`. Para acceder desde tu máquina:

```bash
# Opción A: port-forward (recomendado para dev)
kubectl port-forward svc/nimbuscore-api 8080:80

# Opción B: Ingress + /etc/hosts
echo "127.0.0.1 api.dev.nimbuscore.io" | sudo tee -a /etc/hosts
# Ahora http://api.dev.nimbuscore.io → API
```

> Usá el port-forward (Opción A). El Ingress se usa para los workspaces
> (subdominios), no para la API en dev.

---

## 6. Opcional: Dashboard (Astro frontend)

```bash
cd web/dashboard
pnpm install
pnpm dev   # → http://localhost:5173
```

El `astro.config.mjs` ya tiene proxy de `/api` → `http://localhost:8080`.

---

## 7. Verificar que el sistema funciona

```bash
# Health check de la API
curl -s http://localhost:8080/health
# → {"status":"ok"}

# Ver workspaces CRD
kubectl get workspaces.nimbuscore.io -A

# Ver logs del workspace-manager
kubectl logs -l app.kubernetes.io/component=workspace-manager --tail 20

# Ver logs del operator
kubectl logs -l app.kubernetes.io/component=operator --tail 20
```

---

## 8. Crear un workspace

1. Abrir `http://localhost:5173`
2. Login con Dex: `admin@nimbuscore.io` / `admin123`
3. Click **"NEW WORKSPACE"**
4. Llenar formulario y crear

O por curl:

```bash
# Obtener token (ver RUNBOOK.md para flujo completo)
TOKEN="<jwt>"

curl -s -X POST http://localhost:8080/api/workspaces \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mi-dev",
    "image": "codercom/code-server:latest",
    "resources": { "cpu": "1", "memory": "2Gi", "disk": "10Gi" },
    "ports": [
      {"port": 3000, "protocol": "http", "subdomain": "app"}
    ]
  }' | jq
```

El flujo completo es:

```
POST /api/workspaces
  → API guarda en PostgreSQL (status: pending)
  → API publica evento "workspace.create" en RabbitMQ
    → workspace-manager recibe el evento
      → workspace-manager crea Workspace CRD en K8s
        → operator detecta el CRD
          → Crea Namespace
          → Crea PVC
          → Crea Service (puerto 80→8080 + puertos extra)
          → Crea Pod (code-server + git init)
          → Crea Ingress principal (nombre.nimbuscore.local)
          → Crea Ingress por cada puerto con subdominio
          → Status → "running"
```

Para ver el estado del CRD:

```bash
kubectl get workspaces.nimbuscore.io -A
kubectl describe workspace mi-dev
```

---

## 9. Limpieza

```bash
# Eliminar todo
helm uninstall nimbuscore
kubectl delete crd workspaces.nimbuscore.io
kubectl delete crd prebuilds.nimbuscore.io
kind delete cluster
```
