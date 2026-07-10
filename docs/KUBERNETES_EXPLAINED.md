# Kubernetes, kind, kubectl y Helm — Explicación desde cero

Esta guía asume que NO sabés nada de Kubernetes. Explica cada concepto desde el
principio y cómo aplica específicamente a NimbusCore.

---

## Índice

1. [El problema que resuelve Kubernetes](#1-el-problema-que-resuelve-kubernetes)
2. [Conceptos fundamentales](#2-conceptos-fundamentales)
3. [kind — Kubernetes local para desarrollo](#3-kind--kubernetes-local-para-desarrollo)
4. [kubectl — El control remoto de Kubernetes](#4-kubectl--el-control-remoto-de-kubernetes)
5. [Helm — El instalador de paquetes de Kubernetes](#5-helm--el-instalador-de-paquetes-de-kubernetes)
6. [CRDs — Extensiones de Kubernetes](#6-crds--extensiones-de-kubernetes)
7. [Operator — El cerebro autónomo](#7-operator--el-cerebro-autónomo)
8. [Cómo encaja todo en NimbusCore](#8-cómo-encaja-todo-en-nimbuscore)
9. [Escenario completo: crear un workspace](#9-escenario-completo-crear-un-workspace)
10. [Glosario de términos](#10-glosario-de-términos)

---

## 1. El problema que resuelve Kubernetes

### Sin Kubernetes (solo Docker Compose)

Imaginá que tenés que ejecutar el DevContainer de un usuario. Necesitás:

- Un **Pod** (contenedor) con VS Code Server
- Un **Service** para que sea accesible por otros pods
- Un **Ingress** para que sea accesible desde el browser con un subdominio como
  `app-miproject.nimbuscore.local`
- Un **Volume** para persistir datos
- Monitorear que el pod esté vivo, reiniciarlo si crashea
- Escalar a más usuarios

Con Docker Compose tendrías que:

1. Generar un `compose.yml` dinámicamente por cada workspace
2. Ejecutar `docker compose -p ws-miproject up -d`
3. Monitorear manualmente
4. Limpiar cuando el usuario termina
5. Hacer todo esto para CIENTOS de usuarios simultáneos

**Eso no escala.** Necesitás un orquestador.

### Con Kubernetes

Kubernetes resuelve:

| Problema                      | Solución K8s                                        |
| ----------------------------- | --------------------------------------------------- |
| ¿Dónde corre mi contenedor?   | El **Scheduler** decide en qué node ponerlo         |
| ¿Sigue vivo?                  | **Kubelet** monitorea y reinicia si crashea         |
| ¿Cómo accedo?                 | **Service** + **Ingress** con DNS automático        |
| ¿Dónde guardo datos?          | **PersistentVolume** (disco en red)                 |
| ¿Cómo escalo?                 | **Deployment** con `replicas: 3`                    |
| ¿Y si un nodo muere?          | **Controller Manager** recrea los pods en otro nodo |
| ¿Cómo actualizo sin downtime? | **Rolling update**                                  |

**Kubernetes NO es Docker.** Docker corre contenedores. Kubernetes los ORQUESTA.

### Analogía

| Concepto       | Analogía                                                                                                           |
| -------------- | ------------------------------------------------------------------------------------------------------------------ |
| Docker Compose | Un hotel con recepción. Vos pedís una habitación, te la dan.                                                       |
| Kubernetes     | Un city planner. Decide dónde van los edificios, cómo se conectan las calles, qué pasa si un edificio se incendia. |

Docker Compose es para tu máquina local (un hotel). Kubernetes es para un
datacenter entero (una ciudad).

---

## 2. Conceptos fundamentales

### Pod

El **Pod** es la unidad más pequeña en Kubernetes. Es UNO O MÁS contenedores que
comparten red y almacenamiento.

```yaml
# Un Pod simple que corre VS Code Server
apiVersion: v1
kind: Pod
metadata:
  name: ws-miproject # Nombre del pod
  labels:
    app: code-server
    workspace: miproject
spec:
  containers:
    - name: code-server
      image: ghcr.io/nimbuscore/code-server:latest
      ports:
        - containerPort: 3000 # Puerto dentro del contenedor
      resources:
        limits:
          cpu: '2' # Máximo 2 CPUs
          memory: '4Gi' # Máximo 4 GB de RAM
```

**Diferencia con Docker:**

- Docker: `docker run ghcr.io/nimbuscore/code-server:latest`
- K8s: aplicás este YAML y Kubernetes lo ejecuta en cualquier nodo disponible

### Deployment

Un **Deployment** es una plantilla para crear Pods. Define cuántas réplicas
querés, cómo actualizarlas, etc.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nimbuscore-api
spec:
  replicas: 3 # Quiero 3 copias de la API
  selector:
    matchLabels:
      app: nimbuscore-api
  template:
    metadata:
      labels:
        app: nimbuscore-api
    spec:
      containers:
        - name: api
          image: ghcr.io/nimbuscore/api:latest
          ports:
            - containerPort: 8080
```

**Diferencia con Pod:**

- **Pod**: existe hasta que lo borrás. Si crashea, no se recupera.
- **Deployment**: si un pod crashea, Kubernetes crea otro automáticamente.
  Siempre mantiene `replicas: 3`.

### Service

Un **Service** es una dirección IP estable (no cambia aunque los pods se
recreen) que balancea tráfico entre pods.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nimbuscore-api
spec:
  selector:
    app: nimbuscore-api # Balancea entre pods con esta label
  ports:
    - port: 8080 # Puerto del Service
      targetPort: 8080 # Puerto del Pod
  type: ClusterIP # Solo accesible dentro del cluster
```

**¿Por qué hace falta?** Los pods tienen IPs temporales. Si un pod muere y se
recrea, tiene IP nueva. El Service siempre tiene la misma IP y DNS
(`nimbuscore-api.default.svc.cluster.local`).

### Ingress

Un **Ingress** expone un Service al mundo exterior (HTTP/HTTPS).

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ws-miproject-app
spec:
  rules:
    - host: app-miproject.nimbuscore.local # Subdominio
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: ws-miproject # Service del workspace
                port:
                  number: 3000
```

**¿Cómo llega el tráfico del browser al pod?**

```
Browser → http://app-miproject.nimbuscore.local
  ↓ (DNS resuelve a la IP del Ingress Controller)
Ingress Controller (NGINX)
  ↓ (Lee las reglas del Ingress, busca host: app-miproject.*)
  ↓ (Reenvía al Service ws-miproject:3000)
Service
  ↓ (Balancea al Pod correcto)
Pod ws-miproject (code-server:3000)
```

### Namespace

Un **Namespace** es una partición lógica del cluster. Separa entornos (dev,
prod) o proyectos.

```bash
kubectl get namespaces
# default       → Namespace por defecto
# kube-system   → Componentes internos de Kubernetes
# ingress-nginx → Ingress Controller
# nimbuscore    → Acá instalamos NimbusCore
```

En NimbusCore, cada workspace podría ir en su propio namespace para aislar
recursos.

### ConfigMap / Secret

Almacenan configuración y secretos respectivamente.

```yaml
# ConfigMap (texto plano)
apiVersion: v1
kind: ConfigMap
metadata:
  name: nimbuscore-api-config
data:
  LOG_LEVEL: 'debug'
  DATABASE_URL: 'postgres://...'
```

```yaml
# Secret (base64, pero puede encriptarse con SOPS)
apiVersion: v1
kind: Secret
metadata:
  name: nimbuscore-secrets
type: Opaque
data:
  jwt-secret: ZGV2LWp3dC1zZWNyZXQ= # base64 de "dev-jwt-secret"
```

**Diferencia**: ConfigMap se ve en texto plano. Secret está ofuscado (base64) y
puede encriptarse con tools externas.

---

## 3. kind — Kubernetes local para desarrollo

kind = **K**ubernetes **IN** **D**ocker

### ¿Qué es?

kind crea un cluster Kubernetes DENTRO de contenedores Docker. Cada "nodo" del
cluster es un contenedor Docker que corre un proceso de Kubernetes.

### ¿Por qué kind y no minikube/k3s?

| Herramienta  | Cómo corre K8s                   | Ideal para                         |
| ------------ | -------------------------------- | ---------------------------------- |
| **kind**     | Cada nodo es un container Docker | Desarrollo, CI/CD, pruebas rápidas |
| **minikube** | VM (VirtualBox, HyperKit)        | Desarrollo con addons              |
| **k3s**      | Binario nativo (no VM)           | Edge, IoT, producción pequeña      |
| **k3d**      | k3s dentro de Docker             | Rápido como kind pero con k3s      |

Elegimos kind porque:

1. **Rápido** — crea un cluster en ~30 segundos
2. **Ligero** — ~500 MB de RAM (minikube ~2 GB)
3. **CI-friendly** — funciona en GitHub Actions sin VM
4. **Compatible** — 100% Kubernetes real (no es un simulador)

### Cómo funciona kind

```
Tu máquina (Linux/macOS/Windows)
  │
  └── Docker
       │
       ├── Container: kind-control-plane (el nodo "master")
       │    ├── kube-apiserver (la API de Kubernetes)
       │    ├── kube-controller-manager
       │    ├── kube-scheduler
       │    ├── etcd (base de datos del cluster)
       │    └── containerd (el que corre los pods)
       │
       └── Container: kind-worker (nodo "trabajador")
            ├── kubelet (agente de Kubernetes)
            └── tus pods (workspaces, API, etc.)
```

### Comandos de kind

```bash
# Crear un cluster
kind create cluster
# → Crea un cluster llamado "kind" con 1 nodo (control-plane)
# → Genera kubeconfig en ~/.kube/config

# Crear cluster con configuración personalizada
kind create cluster --config kind-config.yaml
# → Puede tener múltiples nodos, mapeo de puertos, etc.

# Listar clusters
kind get clusters
# kind

# Eliminar cluster
kind delete cluster
# → Borra TODOS los containers del cluster

# Cargar imagen Docker al cluster (sin registry)
kind load docker-image nimbuscore-api:latest
# → Útil para development sin push a registry
```

### kind-config.yaml de NimbusCore

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"  # Permite Ingress
    extraPortMappings:
      - hostPort: 80 # Puerto 80 de tu máquina
        containerPort: 80 # → mapea al puerto 80 del container kind
        protocol: TCP
      - hostPort: 443
        containerPort: 443
        protocol: TCP
```

**¿Qué hace esto?**

1. Crea UN nodo que hace de control-plane y worker (cluster mononodo)
2. Marca el nodo como `ingress-ready=true` (necesario para NGINX Ingress)
3. Mapea puertos 80/443 de tu máquina al container kind

Sin esto, no podrías acceder `http://nimbuscore.local` desde tu browser.

### kind vs cluster real

| Característica       | kind              | Cluster real (EKS, GKE, AKS)        |
| -------------------- | ----------------- | ----------------------------------- |
| Nodos                | Containers Docker | Máquinas virtuales en AWS/GCP/Azure |
| Alta disponibilidad  | No (1 nodo)       | Múltiples nodos en zonas distintas  |
| Balanceador de carga | No (usa hostPort) | Sí (ELB, GLB)                       |
| Discos persistentes  | No (efímero)      | Sí (EBS, PD)                        |
| Tiempo de creación   | 30 segundos       | 10-30 minutos                       |
| Costo                | $0                | ~$50-200/mes                        |

**kind es SOLO para desarrollo.** En producción usás un cluster managed (EKS en
AWS, GKE en GCP, AKS en Azure) o instalás k3s/kubeadm en servidores.

---

## 4. kubectl — El control remoto de Kubernetes

### ¿Qué es?

`kubectl` (kubernetes control) es la línea de comandos para hablar con
Kubernetes. Es como `docker` pero para un cluster completo.

### Cómo se conecta

```bash
kubectl get pods
# → Error: no se puede conectar (no hay kubeconfig)

# kubectl busca ~/.kube/config
# Este archivo lo genera kind automáticamente
export KUBECONFIG=~/.kube/config
# O lo podés ver con:
kubectl config view
```

El `kubeconfig` contiene:

- URL del API server (`https://127.0.0.1:xxxxx`)
- Certificados TLS para autenticarse
- Contexto (qué cluster, qué usuario, qué namespace)

### Comandos básicos (y qué hacen realmente)

#### kubectl get — Listar recursos

```bash
kubectl get pods
# → Lista todos los pods en el namespace actual (default)

# ¿Qué pasa internamente?
# 1. kubectl lee ~/.kube/config
# 2. Hace GET https://api-server:6443/api/v1/namespaces/default/pods
# 3. El API server consulta etcd
# 4. Devuelve JSON con todos los pods
# 5. kubectl formatea la salida en tabla

kubectl get pods -A
# -A = --all-namespaces, pods de TODOS los namespaces

kubectl get pods -n nimbuscore
# -n = namespace específico

kubectl get pods -o wide
# -o wide = más columnas (IP del pod, nodo donde corre)

kubectl get pods -o yaml
# -o yaml = salida en YAML (todo el objeto completo)

kubectl get pods -w
# -w = watch, mantiene la conexión abierta y muestra cambios en tiempo real
```

#### kubectl describe — Ver detalle

```bash
kubectl describe pod nimbuscore-api-7f8c9
# → Muestra TODO sobre ese pod:
#   - Events (útil para debugging)
#   - Conditions (Ready, Initialized)
#   - Containers (image, ports, resources)
#   - Volumes montados
#   - Labels y Annotations
```

**¿Diferencia entre `get` y `describe`?**

- `get`: estado actual (running, pending, etc.)
- `describe`: historia y detalles (events, condiciones, errores)

#### kubectl logs — Ver logs

```bash
kubectl logs nimbuscore-api-7f8c9
# → Logs del contenedor (stdout/stderr)

kubectl logs -f nimbuscore-api-7f8c9
# -f = follow, como docker logs -f

kubectl logs -l app=nimbuscore-api
# -l = selector por label, logs de TODOS los pods con esa label

kubectl logs --previous nimbuscore-api-7f8c9
# → Logs de la instancia ANTERIOR del pod (si se reinició)
```

#### kubectl apply — Crear/actualizar recursos

```bash
kubectl apply -f deployment.yaml
# → Envía el YAML al API server
# → Si el recurso NO existe, lo CREA
# → Si el recurso SÍ existe, lo ACTUALIZA (merge)

# ¿Qué pasa internamente?
# 1. kubectl lee deployment.yaml
# 2. HACE POST https://api-server/.../deployments (si no existe)
#    o PATCH (si ya existe)
# 3. API server valida el YAML
# 4. Almacena en etcd
# 5. Controller Manager detecta el cambio
# 6. Crea los pods necesarios
```

**Diferencia entre `apply`, `create`, `replace`:**

- `apply`: declarativo. "Esto es lo que quiero". K8s hace el merge.
- `create`: imperativo. "Creá esto". Falla si ya existe.
- `replace`: imperativo. "Reemplazá todo". Borra y recrea.

**Siempre usar `apply`.** Es la forma declarativa.

#### kubectl delete — Eliminar

```bash
kubectl delete pod nimbuscore-api-7f8c9
# → Elimina el pod
# → El Deployment LO RECREA porque replicas=3

kubectl delete deployment nimbuscore-api
# → Elimina el Deployment y TODOS los pods que creó

kubectl delete -f deployment.yaml
# → Elimina todo lo que está en ese archivo YAML
```

#### kubectl exec — Ejecutar dentro de un pod

```bash
kubectl exec -it nimbuscore-api-7f8c9 -- sh
# → Abre una shell DENTRO del contenedor (como docker exec)

kubectl exec nimbuscore-api-7f8c9 -- ls /
# → Ejecuta un comando y devuelve la salida
```

### Flags útiles

| Flag               | Significado          | Ejemplo                                         |
| ------------------ | -------------------- | ----------------------------------------------- |
| `-n`               | Namespace            | `kubectl get pods -n nimbuscore`                |
| `-A`               | Todos los namespaces | `kubectl get pods -A`                           |
| `-o wide`          | Más columnas         | `kubectl get pods -o wide`                      |
| `-o yaml`          | Salida YAML          | `kubectl get pod x -o yaml`                     |
| `-o json`          | Salida JSON          | `kubectl get pod x -o json`                     |
| `-w`               | Watch (tiempo real)  | `kubectl get pods -w`                           |
| `-l`               | Label selector       | `kubectl get pods -l app=api`                   |
| `--all`            | Todos los recursos   | `kubectl delete pods --all`                     |
| `-f`               | From file            | `kubectl apply -f file.yaml`                    |
| `--force`          | Forzar eliminación   | `kubectl delete pod x --force`                  |
| `--grace-period=0` | Sin gracia           | `kubectl delete pod x --grace-period=0 --force` |

### ¿Qué pasa cuando ejecutamos estos comandos en NimbusCore?

```bash
# 1. Ver que el cluster esté vivo
kubectl get nodes
# → Muestra el nodo de kind

# 2. Ver qué instalamos
kubectl get pods -n default
# → Los pods de NimbusCore (api, operator, manager)

# 3. Ver CRDs (recursos personalizados)
kubectl get crd | grep nimbuscore
# → workspaces.nimbuscore.io
# → prebuilds.nimbuscore.io

# 4. Ver workspaces creados
kubectl get ws -A
# → Lista todos los workspaces (CRDs)
```

---

## 5. Helm — El instalador de paquetes de Kubernetes

### ¿Qué es?

Helm es como `apt` (Ubuntu), `brew` (macOS) o `pnpm` (Node.js), pero para
Kubernetes.

- **Chart** = paquete (como un `.deb` o `package.json`)
- **Release** = una instalación de un chart (como un proceso corriendo)
- **Repository** = lugar donde se almacenan charts (como npm registry)

### ¿Por qué no usar solo YAMLs sueltos?

Sin Helm:

```bash
kubectl apply -f deployment-api.yaml
kubectl apply -f service-api.yaml
kubectl apply -f configmap-api.yaml
kubectl apply -f secret.yaml
kubectl apply -f hpa.yaml
kubectl apply -f ingress.yaml
kubectl apply -f clusterrole.yaml
kubectl apply -f clusterrolebinding.yaml
# ... 15 archivos, 15 comandos
# Y ni hablar de actualizar configuraciones
```

Con Helm:

```bash
helm upgrade --install nimbuscore ./chart --set api.replicas=3
# → UN SOLO COMANDO instala TODO
# → --set permite cambiar config sin editar archivos
```

### Anatomía de un chart

```
nimbuscore/
├── Chart.yaml         # Metadatos (nombre, versión, dependencias)
├── values.yaml        # Valores por defecto (como default props en React)
├── values.dev.yaml    # Override para dev (como .env.dev)
├── values.prod.yaml   # Override para prod
├── crds/              # CRDs (se instalan antes que todo)
├── templates/         # Plantillas Go con {{ .Values.xxx }}
│   ├── deployment-api.yaml
│   ├── service.yaml
│   └── ...
```

### ¿Qué son las plantillas (templates)?

Es Go con {{ llaves }}. Helm reemplaza las variables antes de enviar a
Kubernetes.

```yaml
# templates/deployment-api.yaml (plantilla)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "nimbuscore.fullname" . }}-api
spec:
  replicas: {{ .Values.api.replicas }}    # → Se reemplaza con el valor
  template:
    spec:
      containers:
        - name: api
          image: "{{ .Values.image.registry }}/nimbuscore-api:{{ .Values.image.tag }}"
```

Cuando ejecutás:

```bash
helm template . --values values.dev.yaml
# → Genera el YAML final reemplazando las variables
# → NO envía a K8s, solo muestra el resultado
```

Salida:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nimbuscore-api
spec:
  replicas: 1 # Desde values.dev.yaml
  template:
    spec:
      containers:
        - name: api
          image: ghcr.io/nimbuscore/nimbuscore-api:latest
```

### Valores (values)

```yaml
# values.yaml (valores por defecto — produccion)
api:
  replicas: 3
  resources:
    cpu: '500m'
    memory: '512Mi'

monitoring:
  serviceMonitor:
    enabled: true
```

```yaml
# values.dev.yaml (override para dev)
api:
  replicas: 1 # Solo 1 réplica en dev

monitoring:
  serviceMonitor:
    enabled: false # Sin Prometheus en dev
```

**Cómo se mergean:**

```bash
helm install ... --values values.dev.yaml
# → Helm combina values.yaml + values.dev.yaml
# → values.dev.yaml gana sobre values.yaml (como Object.assign)
# → api.replicas → 1 (de dev), no 3 (de prod)
```

### Dependencias (subcharts)

```yaml
# Chart.yaml
dependencies:
  - name: postgresql
    version: '16.x'
    repository: 'https://charts.bitnami.com/bitnami'
```

Esto permite que NimbusCore instale PostgreSQL automáticamente como parte del
chart.

```bash
# Antes del primer install, descargar dependencias
helm dependency update deploy/helm/nimbuscore/
# → Descarga postgresql-16.x.tgz a charts/
```

### Instalación completa explicada

```bash
helm upgrade --install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.dev.yaml \
  --set secrets.jwt="dev-jwt-secret" \
  --set secrets.oidcClientSecret="dev-oidc-secret" \
  --set secrets.resticPassword="dev-restic-password" \
  --set global.ingress.host="nimbuscore.local"
```

**Paso a paso de lo que hace:**

1. **`helm upgrade --install`**: "Si no existe el release `nimbuscore`, créalo.
   Si existe, actualízalo."

2. **`nimbuscore`**: Nombre del release. Después de instalarlo, lo referenciás
   por este nombre.

3. **`deploy/helm/nimbuscore/`**: Ruta al chart (el directorio con Chart.yaml).

4. **`--values values.dev.yaml`**: Usá estos valores (1 réplica, sin
   monitoring).

5. **`--set secrets.jwt="..."`**: Seteá valores específicos. Útil para secretos
   que no querés en el repo.

6. **Lo que Helm hace internamente**:

```
Helm
  │
  ├── 1. Lee Chart.yaml → nombre, versión, dependencias
  │
  ├── 2. Mergea values.yaml + values.dev.yaml + --set flags
  │     api.replicas = 1 (dev)
  │     secrets.jwt = "dev-jwt-secret" (--set)
  │
  ├── 3. Procesa templates/ reemplazando {{ .Values.xxx }}
  │     deployment-api.yaml → YAML con replicas: 1
  │     service.yaml → YAML con puerto 8080
  │     secret.yaml → YAML con jwt en base64
  │     ...
  │
  ├── 4. Envía los YAMLs a Kubernetes (kubectl apply)
  │     POST deployment-api → K8s crea el Deployment
  │     POST service → K8s crea el Service
  │     POST secret → K8s crea el Secret
  │     ...
  │
  └── 5. Guarda el release en Helm (secrets en K8s)
      → Podés hacer helm rollback después
```

### helm template (simular sin instalar)

```bash
# Ver qué YAML generaría Helm sin instalar nada
helm template nimbuscore deploy/helm/nimbuscore \
  --values values.dev.yaml \
  --set secrets.jwt="test" | head -50
```

**Usá esto cuando quieras ver qué va a instalar antes de ejecutarlo.**

### helm list, status, rollback

```bash
# Listar releases instalados
helm list
# NAME       NAMESPACE  REVISION  STATUS    CHART
# nimbuscore default    1         deployed  nimbuscore-1.0.0

# Ver estado del release
helm status nimbuscore
# → Muestra recursos instalados, notas, etc.

# Ver historial de revisiones
helm history nimbuscore
# REVISION  STATUS     CHART
# 1         deployed   nimbuscore-1.0.0
# 2         superseded nimbuscore-1.0.0 (cuando upgradeás)

# Volver a una versión anterior
helm rollback nimbuscore 1
# → Revierte a la revisión 1

# Eliminar release (desinstalar todo)
helm uninstall nimbuscore
# → Borra TODOS los recursos K8s que Helm creó
```

---

## 6. CRDs — Extensiones de Kubernetes

### ¿Qué son?

Custom Resource Definitions (CRDs) son EXACTAMENTE lo que su nombre dice:
definiciones de recursos personalizados. Le decís a Kubernetes "che, además de
Pods, Services, Deployments, quiero un recurso llamado **Workspace**".

### ¿Por qué necesitamos CRDs?

Kubernetes viene con recursos built-in:

- Pod, Service, Deployment, Ingress, ConfigMap, Secret, etc.

Pero Kubernetes NO sabe qué es un "Workspace" de NimbusCore. Necesitamos
enseñarle.

### Cómo se define un CRD

```yaml
# deploy/helm/nimbuscore/crds/workspace.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: workspaces.nimbuscore.io # IMPORTANTE: plural.grupo
spec:
  group: nimbuscore.io
  names:
    kind: Workspace # kubectl get workspace
    plural: workspaces # kubectl get workspaces
    singular: workspace # kubectl get workspace
    shortNames:
      - ws # kubectl get ws
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                image:
                  type: string
                cpu:
                  type: string
                ports:
                  type: array
                  items:
                    type: object
                    properties:
                      port:
                        type: integer
```

**Después de aplicar esto**, Kubernetes acepta:

```bash
kubectl get workspaces
# No resources found in default namespace.
# → Ahora K8s sabe qué es un workspace, aunque no haya ninguno

kubectl get crd | grep nimbuscore
# workspaces.nimbuscore.io
```

### Cómo se usa un CRD

```yaml
# Un workspace específico (se crea automáticamente cuando creás uno via API)
apiVersion: nimbuscore.io/v1
kind: Workspace
metadata:
  name: mi-dev-container
spec:
  image: ghcr.io/nimbuscore/code-server:latest
  cpu: '2'
  memory: 4Gi
  ports:
    - port: 3000
      protocol: tcp
      subdomain: app
```

```bash
kubectl apply -f mi-workspace.yaml
# → Crea un recurso de tipo Workspace (CRD) en Kubernetes
```

### CRD vs Recurso normal

```
CRD = La DEFINICIÓN del nuevo recurso (solo se aplica UNA vez)
 │
 └── Workspace (el recurso en sí, se aplica CADA vez que creás uno)
```

**Analogía:**

- CRD = La tabla `CREATE TABLE workspaces (...)` en SQL
- Recurso = El `INSERT INTO workspaces VALUES (...)` con datos específicos

### CRDs de NimbusCore

| CRD                        | Propósito                               | shortName |
| -------------------------- | --------------------------------------- | --------- |
| `workspaces.nimbuscore.io` | Representa un DevContainer en ejecución | `ws`      |
| `prebuilds.nimbuscore.io`  | Representa una Prebuild de DevContainer | `pb`      |

### ¿Quién crea los CRDs?

El OPERADOR (controller-runtime) es el que REACCIONA cuando se
crea/modifica/elimina un CRD.

```
Alguien crea un CRD Workspace
  │
  ▼
Operator detecta el cambio
  │
  ▼
Operator consulta: ¿Qué hay que hacer?
  ├── Si status = pending → crear Pod, Service, Ingress
  ├── Si status = stopping → eliminar Pod, Service, Ingress
  └── Si status = deleting → eliminar todo
```

---

## 7. Operator — El cerebro autónomo

### ¿Qué es un Operator?

Un **Operator** es un programa que corre dentro de Kubernetes y automatiza
tareas. Extiende el comportamiento de Kubernetes para recursos específicos.

### ¿Por qué necesitamos un Operator?

Sin operator:

```
1. API recibe POST /workspaces
2. API inserta en PostgreSQL (status: pending)
3. ... y ahora qué? ¿Quién crea el Pod?
```

La API NO debería crear Pods directamente (está fuera de K8s o tiene permisos
limitados). Necesitamos algo que:

- Mire la base de datos periódicamente
- Cree los recursos K8s correspondientes
- Actualice el estado

Con operator:

```
1. API recibe POST /workspaces
2. API inserta en PostgreSQL (status: pending)
3. API crea CRD Workspace en K8s (vía kubectl)
4. OPERATOR detecta el nuevo CRD
5. OPERATOR crea Pod, Service, Ingress
6. OPERATOR actualiza status del CRD a "running"
```

### Reconciliation loop (bucle de reconciliación)

El operator NO se ejecuta cada N segundos como un cron. Usa **informers** que
escuchan cambios en tiempo real via WebSocket.

```
1. Algo cambia (se crea un Workspace CRD)
   │
2. Informer (WebSocket) avisa al operator
   │
3. Operator llama a Reconcile(ctx, request)
   │
4. Reconcile:
   a. GET el recurso (ej: workspace mi-dev-container)
   b. Compara spec (lo que QUIERO) vs status (lo que TENGO)
   c. Si spec.image cambió → actualizar Pod
   d. Si status es "pending" → crear Pod, Service, Ingress
   e. Si status es "stopping" → eliminar Pod, Service, Ingress
   f. Actualiza status del CRD
   │
5. El cambio en status dispara OTRO reconcile
   (para verificar que todo quedó bien)
```

### Código del reconciler de Workspace

```go
func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Obtener el workspace CRD de Kubernetes
    var ws nimbusv1.Workspace
    if err := r.Get(ctx, req.NamespacedName, &ws); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Según el estado, hacer algo
    switch ws.Status.Phase {
    case "pending", "building":
        // 2a. Crear o actualizar Pod
        pod := buildPod(ws)
        if err := r.Create(ctx, pod); err != nil && !errors.IsAlreadyExists(err) {
            return ctrl.Result{}, err
        }

        // 2b. Crear Service
        svc := buildService(ws)
        if err := r.Create(ctx, svc); err != nil && !errors.IsAlreadyExists(err) {
            return ctrl.Result{}, err
        }

        // 2c. Crear Ingress por cada puerto
        for port := range ws.Spec.Ports {
            ing := buildIngress(ws, port)
            if err := r.Create(ctx, ing); err != nil && !errors.IsAlreadyExists(err) {
                return ctrl.Result{}, err
            }
        }

        // 2d. Actualizar estado
        ws.Status.Phase = "running"
        r.Status().Update(ctx, &ws)

    case "stopping":
        // Eliminar Pod, Service, Ingresses
        r.Delete(ctx, pod)
        r.Delete(ctx, svc)
        for port := range ws.Spec.Ports {
            r.Delete(ctx, ingress)
        }
        ws.Status.Phase = "stopped"
        r.Status().Update(ctx, &ws)
    }

    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

### ¿Cómo corre el operator?

```bash
# Opción 1: Como un deployment en K8s (producción)
kubectl get pods | grep operator
# nimbuscore-operator-7f8c9   1/1 Running

# Opción 2: Localmente (desarrollo)
cd apps/operator && go run ./cmd/
# Usa el kubeconfig de ~/.kube/config
# Útil para debuggear sin deployar
```

### Diferencia: Operator vs Workspace Manager

| Componente        | Operator                               | Workspace Manager                  |
| ----------------- | -------------------------------------- | ---------------------------------- |
| Corre en          | K8s (Deployment) o local               | Docker o K8s                       |
| Qué hace          | Crea/elimina Pods, Services, Ingresses | Snapshots con Restic, idle watcher |
| Cómo se entera    | Informers de K8s (WebSocket)           | RabbitMQ (cola de mensajes)        |
| Estado que maneja | pending → running → stopped            | Snapshots, backups                 |
| Depende de K8s?   | SÍ                                     | NO (usa kubectl exec para Restic)  |

---

## 8. Cómo encaja todo en NimbusCore

### Diagrama de flujo completo

```
TU MÁQUINA (Linux)
  │
  ├── Docker Compose (servicios de infraestructura)
  │   ├── PostgreSQL :5432
  │   ├── Redis :6379
  │   ├── RabbitMQ :5672
  │   └── Dex :5556
  │
  ├── kind (Kubernetes dentro de Docker)
  │   │
  │   └── Cluster kind
  │       │
  │       ├── NGINX Ingress Controller (balanceador)
  │       │
  │       ├── Helm install: nimbuscore
  │       │   ├── Deployment: nimbuscore-api
  │       │   ├── Deployment: nimbuscore-operator
  │       │   ├── Deployment: nimbuscore-manager
  │       │   ├── Service: nimbuscore-api
  │       │   ├── ConfigMaps (config)
  │       │   ├── Secrets (JWT, OIDC)
  │       │   ├── ClusterRole + ClusterRoleBinding (permisos)
  │       │   └── HPA (autoescalado)
  │       │
  │       └── CRDs (recursos personalizados)
  │           ├── workspaces.nimbuscore.io
  │           └── prebuilds.nimbuscore.io
  │
  ├── Frontend Astro :5173
  │
  └── Browser → http://nimbuscore.local
```

### ¿Qué corre dónde?

| Componente        | Corre en               | Cómo se inicia                |
| ----------------- | ---------------------- | ----------------------------- |
| PostgreSQL        | Docker (compose)       | `docker compose up -d`        |
| Redis             | Docker (compose)       | `docker compose up -d`        |
| RabbitMQ          | Docker (compose)       | `docker compose up -d`        |
| Dex               | Docker (compose)       | `docker compose up -d`        |
| API               | Docker (compose) o K8s | `docker compose up -d` o Helm |
| Frontend          | Tu máquina (node)      | `pnpm dev`                    |
| Operator          | K8s                    | Helm                          |
| Workspace Manager | K8s                    | Helm                          |

**¿Por qué algunos servicios están en Docker y otros en K8s?**

En producción TODO estaría en K8s. Pero para desarrollo local:

- PostgreSQL, Redis, RabbitMQ, Dex son más fáciles de manejar con Docker Compose
  (volúmenes, healthchecks, logs simples)
- La API puede correr en Docker (para desarrollo rápido sin Helm) o en K8s (para
  probar el deploy completo)
- El operator y workspace manager solo tienen sentido en K8s o apuntando a un
  cluster

### ¿Qué hace cada componente de NimbusCore?

| Componente            | Archivo                   | Responsabilidad                             |
| --------------------- | ------------------------- | ------------------------------------------- |
| **api**               | `apps/api/`               | CRUD HTTP + auth + publicar eventos         |
| **operator**          | `apps/operator/`          | Reconciliar CRDs contra K8s real            |
| **workspace-manager** | `apps/workspace-manager/` | Snapshots Restic + idle watcher + scheduler |
| **cli**               | `apps/cli/`               | CLI (futuro)                                |

---

## 9. Escenario completo: crear un workspace

### Paso 1: Todo funcionando

```bash
# Terminal 1: infraestructura
docker compose up -d

# Terminal 2: frontend
cd web/dashboard && pnpm dev

# Terminal 3: Kubernetes listo
kubectl get nodes
# NAME                 STATUS   ROLES           AGE
# kind-control-plane   Ready    control-plane   5m

kubectl get pods -n default
# NAME                                  READY   STATUS
# nimbuscore-api-7f8c9d9b6f-abc12      1/1     Running
# nimbuscore-operator-6f8c9d9b6f-def34 1/1     Running
# nimbuscore-manager-5g7h8j9k0l-mno56  1/1     Running

kubectl get crd | grep nimbuscore
# prebuilds.nimbuscore.io
# workspaces.nimbuscore.io
```

### Paso 2: Usuario crea workspace via Dashboard

```
Dashboard → Click "New Workspace"
  ├── Name: mi-proyecto
  ├── Image: ghcr.io/nimbuscore/code-server:latest
  ├── CPU: 2, Memory: 4Gi
  └── Ports: 3000 (app)

Click "Create"

POST /api/workspaces → API
```

### Paso 3: La API procesa

```
API recibe POST /api/workspaces
  │
  ├── 1. Valida JWT (Authorization header)
  │     → Extrae user_id: "uuid-del-user"
  │
  ├── 2. Verifica quota del team (si aplica)
  │     → SELECT FROM quotas WHERE team_id = X
  │     → Si excede → 409 Conflict
  │
  ├── 3. Inserta en PostgreSQL
  │     → INSERT INTO workspaces (id, name, user_id, image, status, ...)
  │     → status = "pending"
  │
  ├── 4. Publica evento en RabbitMQ
  │     → Exchange: nimbuscore.workspace
  │     → Routing key: "workspace.create"
  │     → Body: { id, name, image, ... }
  │
  ├── 5. Crea CRD Workspace en Kubernetes
  │     → kubectl apply -f workspace-crd.yaml
  │     → (la API tiene permisos para crear CRDs via service account)
  │
  └── 6. Devuelve workspace JSON al frontend
       → { id, name, status: "pending", ... }
```

### Paso 4: El operator detecta el CRD

```
Operator (controller-runtime) corriendo en K8s
  │
  ├── Informer detecta: NUEVO Workspace CRD
  │
  ├── Llama a Reconcile(ctx, {Name: "mi-proyecto", Namespace: "default"})
  │
  ├── 4a. GET workspace CRD
  │     → status.phase = "pending"
  │
  ├── 4b. Crea Pod
  │     → Container: ghcr.io/nimbuscore/code-server:latest
  │     → Resources: cpu 2, memory 4Gi
  │     → Ports: 3000
  │
  ├── 4c. Crea Service
  │     → ClusterIP, puerto 3000 → pod puerto 3000
  │
  ├── 4d. Crea Ingress
  │     → app-mi-proyecto.nimbuscore.local → Service:3000
  │
  └── 4e. Actualiza status del CRD
       → status.phase = "running"
       → status.pod_name = "ws-mi-proyecto-abc123"
       → status.port_urls = ["http://app-mi-proyecto.nimbuscore.local"]
```

### Paso 5: Usuario accede al workspace

```
Browser → http://app-mi-proyecto.nimbuscore.local
  │
  ├── DNS → ¿a dónde va?
  │     En kind, agregaste 127.0.0.1 nimbuscore.local en /etc/hosts
  │     El wildcard *.nimbuscore.local apunta a localhost
  │
  ├── El request llega a localhost:80
  │     → NGINX Ingress Controller (corriendo en kind)
  │
  ├── NGINX busca Ingress con host "app-mi-proyecto.nimbuscore.local"
  │     → Lo encuentra → reenvía al Service ws-mi-proyecto:3000
  │
  ├── Service balancea al Pod
  │     → Pod ws-mi-proyecto:3000 (code-server)
  │
  └── Usuario ve VS Code Server en el browser
```

### Paso 6: Auto-stop

```
Usuario cierra el browser. Pasan 20 minutos.
  │
  ├── workspace-manager (idle watcher):
  │     → Cada 5 minutos, chequea last_activity de cada workspace
  │     → last_activity se actualiza con heartbeats (POST /heartbeat)
  │     → Si last_activity > 20 min → publica "workspace.timeout"
  │
  ├── Operator recibe el timeout
  │     → status.phase = "stopping"
  │     → Elimina Pod, Service, Ingress
  │     → status.phase = "stopped"
  │
  └── Dashboard muestra workspace como "Inactivo"
```

---

## 10. Glosario de términos

### Kubernetes

| Término              | Definición                                         | Analogía                                    |
| -------------------- | -------------------------------------------------- | ------------------------------------------- |
| **Cluster**          | Conjunto de nodos (máquinas) que corren Kubernetes | Una ciudad                                  |
| **Node**             | Máquina física o virtual que corre pods            | Un edificio                                 |
| **Pod**              | Unidad mínima: uno o más contenedores              | Un departamento                             |
| **Deployment**       | Plantilla que define cómo crear y escalar pods     | Un plano de construcción                    |
| **Service**          | IP estable + balanceo entre pods                   | La recepción del edificio                   |
| **Ingress**          | Expone servicios HTTP al mundo exterior            | La puerta principal                         |
| **Namespace**        | Partición lógica del cluster                       | Un barrio                                   |
| **ConfigMap**        | Configuración en texto plano                       | Una pizarra con instrucciones               |
| **Secret**           | Configuración sensible (base64)                    | Una caja fuerte                             |
| **PersistentVolume** | Disco persistente (no se borra al eliminar el pod) | Un depósito externo                         |
| **CRD**              | Extensión de Kubernetes para recursos nuevos       | Un nuevo tipo de construcción               |
| **Operator**         | Programa que automatiza tareas en K8s              | Un administrador del edificio               |
| **Controller**       | Código que reconcilia estado actual vs deseado     | Un inspector que compara planos vs realidad |
| **Informer**         | Escucha cambios en K8s via WebSocket               | Un sistema de notificaciones                |
| **Reconcile**        | Bucle que compara spec (deseado) vs status (real)  | El ciclo de planificar→ejecutar→verificar   |
| **Scheduler**        | Decide en qué nodo corre cada pod                  | Un asignador de oficinas                    |
| **kubelet**          | Agente de K8s en cada nodo, corre los pods         | El conserje del edificio                    |
| **etcd**             | Base de datos clave-valor del cluster              | El archivo central                          |
| **kubeconfig**       | Archivo de configuración para kubectl              | Un carnet de identidad                      |

### kind

| Término                    | Definición                                                      |
| -------------------------- | --------------------------------------------------------------- |
| **kind**                   | Kubernetes IN Docker: cluster K8s dentro de contenedores Docker |
| **kind create cluster**    | Crea un cluster nuevo con un nodo control-plane                 |
| **kind delete cluster**    | Elimina el cluster entero (todos los contenedores)              |
| **kind load docker-image** | Copia una imagen Docker local al cluster sin registry           |
| **kind-config.yaml**       | Configuración del cluster (número de nodos, puertos, etc.)      |

### kubectl

| Término              | Definición                                         |
| -------------------- | -------------------------------------------------- |
| **kubectl get**      | Lista recursos                                     |
| **kubectl describe** | Muestra detalle de un recurso (incluyendo eventos) |
| **kubectl apply**    | Crea o actualiza recursos desde un archivo         |
| **kubectl delete**   | Elimina recursos                                   |
| **kubectl logs**     | Muestra logs de un pod                             |
| **kubectl exec**     | Ejecuta un comando dentro de un pod                |
| **kubectl -n**       | Especifica namespace                               |
| **kubectl -A**       | Todos los namespaces                               |
| **kubectl -w**       | Watch (mantiene conexión, muestra cambios)         |

### Helm

| Término                    | Definición                                          |
| -------------------------- | --------------------------------------------------- |
| **Helm**                   | Gestor de paquetes para Kubernetes                  |
| **Chart**                  | Paquete con plantillas YAML + valores por defecto   |
| **Release**                | Una instalación de un chart (puede haber múltiples) |
| **values.yaml**            | Valores por defecto del chart                       |
| **values.dev.yaml**        | Override para desarrollo                            |
| **--set**                  | Setear valores específicos desde CLI                |
| **helm upgrade --install** | Crear o actualizar un release (idempotente)         |
| **helm uninstall**         | Eliminar release y todos sus recursos               |
| **helm template**          | Generar YAML sin instalar (simulación)              |
| **helm rollback**          | Volver a una versión anterior                       |
| **helm dependency**        | Manejar dependencias (subcharts Bitnami)            |

### NimbusCore específico

| Término               | Definición                                                             |
| --------------------- | ---------------------------------------------------------------------- |
| **Workspace CRD**     | `workspaces.nimbuscore.io` — representa un DevContainer                |
| **Prebuild CRD**      | `prebuilds.nimbuscore.io` — representa un prebuild                     |
| **Operator**          | `apps/operator/` — reconcilia CRDs contra K8s                          |
| **Workspace Manager** | `apps/workspace-manager/` — snapshots + idle watcher                   |
| **Reconciler**        | Código que maneja las transiciones de estado (pending→running→stopped) |

### Comandos que NO tenés que entender, solo memorizar

```bash
# Para desarrollo local:
kind create cluster --config kind-config.yaml
# → Crea el cluster de pruebas (hacelo una vez)

# Para instalar NimbusCore:
kubectl apply -f deploy/helm/nimbuscore/crds/
# → Instala CRDs (hacelo una vez)

helm upgrade --install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.dev.yaml \
  --set secrets.jwt="dev-jwt-secret" \
  --set secrets.oidcClientSecret="dev-oidc-secret" \
  --set secrets.resticPassword="dev-restic-password" \
  --set global.ingress.host="nimbuscore.local"
# → Instala todo en K8s

# Para limpiar:
helm uninstall nimbuscore
kind delete cluster
docker compose down
```

---

## Resumen visual

```
┌─────────────────────────────────────────────────────────────┐
│                   ¿QUÉ ES CADA COSA?                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  kind         = Docker que simula ser Kubernetes            │
│  kubectl      = Control remoto para hablar con Kubernetes   │
│  Helm         = Instalador de paquetes para Kubernetes      │
│  CRD          = Enseñarle a Kubernetes un recurso nuevo     │
│  Operator     = Programa que automáticamente maneja CRDs    │
│  Pod          = Un contenedor (ej: code-server)             │
│  Service      = Dirección fija + balanceo para pods         │
│  Ingress      = Puerta de entrada desde el browser          │
│  Deployment   = Plantilla que mantiene N copias de un pod   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

```bash
# Flujo de comandos para entender:
kind create cluster                  # 1. Crear cluster K8s falso
kubectl get nodes                    # 2. Ver que existe
helm install ...                     # 3. Instalar NimbusCore
kubectl get pods                     # 4. Ver qué se instaló
kubectl get crd | grep nimbuscore    # 5. Ver extensiones
# ... crear workspace ...
kubectl get ws                       # 6. Ver workspace como CRD
kubectl get pods -o wide             # 7. Ver pods creados por operator
kubectl describe ws mi-proyecto      # 8. Ver detalle del workspace
```
