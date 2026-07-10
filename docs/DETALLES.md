# NimbusCore — Documentación Detallada

## Índice

1. [Arquitectura general](#1-arquitectura-general)
2. [Go — Lenguaje del backend](#2-go--lenguaje-del-backend)
3. [Chi — Router HTTP](#3-chi--router-http)
4. [controller-runtime — Operador Kubernetes](#4-controller-runtime--operador-kubernetes)
5. [Astro 7 — Framework del frontend](#5-astro-7--framework-del-frontend)
6. [React 19 — Islas interactivas](#6-react-19--islas-interactivas)
7. [Tailwind CSS 4 — Estilos](#7-tailwind-css-4--estilos)
8. [PostgreSQL 17 — Base de datos](#8-postgresql-17--base-de-datos)
9. [Redis 7 — Caché](#9-redis-7--caché)
10. [RabbitMQ 4 — Cola de mensajes](#10-rabbitmq-4--cola-de-mensajes)
11. [Dex — Proveedor OIDC](#11-dex--proveedor-oidc)
12. [pnpm — Gestor de paquetes](#12-pnpm--gestor-de-paquetes)
13. [Docker / Docker Compose](#13-docker--docker-compose)
14. [Helm — Paquete Kubernetes](#14-helm--paquete-kubernetes)
15. [Taskfile — Runner de tareas](#15-taskfile--runner-de-tareas)
16. [Estructura del monorepo](#16-estructura-del-monorepo)
17. [Flujo completo: login → workspace](#17-flujo-completo-login--workspace)
18. [Glosario de comandos](#18-glosario-de-comandos)

---

## 1. Arquitectura general

NimbusCore es una plataforma de entornos de desarrollo on-demand (como GitHub
Codespaces) para correr en infraestructura propia (self-hosted) sobre
Kubernetes.

```
┌──────────────────────────────────────────────────┐
│                  Browser (UI)                      │
│         http://localhost:5173 (dev)                │
└──────────┬───────────────────────────┬────────────┘
           │                           │
           │ /api/* (proxy)            │ /auth/* (proxy)
           ▼                           ▼
┌──────────────────────┐   ┌──────────────────────┐
│   API (Chi) :8080     │   │   Dex (OIDC) :5556   │
│   Go 1.26             │   │                      │
│   auth, crud, eventos │   │   Proveedor OpenID   │
└───────┬──────┬───────┘   └──────────────────────┘
        │      │
        ▼      ▼
┌──────────┐ ┌──────────┐
│PostgreSQL│ │  Redis   │
│    :5432 │ │   :6379  │
└──────────┘ └──────────┘
        │
        ▼
┌──────────────┐
│  RabbitMQ    │
│    :5672     │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────┐
│        Kubernetes Cluster                 │
│                                           │
│  ┌───────────┐  ┌───────────────────┐    │
│  │  Operator  │  │ Workspace Manager │    │
│  │ CRD recon- │  │ eventos, snap-    │    │
│  │ ciliador   │  │ shots, idle       │    │
│  └─────┬─────┘  └────────┬──────────┘    │
│        │                  │               │
│        ▼                  ▼               │
│  ┌──────────────┐  ┌──────────────┐      │
│  │ Pods, Svcs,  │  │ Jobs de      │      │
│  │ Ingresses    │  │ prebuild     │      │
│  └──────────────┘  └──────────────┘      │
└──────────────────────────────────────────┘
```

**Flujo de datos por capa:**

1. **Browser** → Frontend Astro (React islands) → llama a API via proxy
2. **API** → autentica (OIDC + JWT), CRUD en PostgreSQL, cachea en Redis,
   publica eventos en RabbitMQ
3. **RabbitMQ** → workspace-manager consume eventos (crear, detener, snapshot)
4. **Operator** → reconcilia CRDs contra Kubernetes real (pods, services,
   ingresses)
5. **Workspace-manager** → snapshots con Restic, idle watcher, scheduler de
   backups

---

## 2. Go — Lenguaje del backend

### ¿Por qué Go?

- **Compilado a binario estático** — un solo binario sin dependencias, ideal
  para Docker distroless (imagen de ~15 MB)
- **Concurrencia nativa** — goroutines + channels para manejar miles de
  conexiones simultáneas
- **Tipado fuerte** — detecta errores en compilación, no en runtime
- **Rendimiento** — latency de microsegundos, sin GC pauses largas
- **Ecosistema K8s** — client-go, controller-runtime están escritos en Go

### Versión: Go 1.26.5

Usamos la versión más reciente estable. Se maneja con `mise`:

```bash
mise install go@1.26.5
mise use go@1.26.5
eval "$(mise activate zsh)"
```

### Cómo se usa en el proyecto

Hay **8 módulos Go** separados en un workspace (`go.work`):

```go
// go.work
go 1.26.5

use (
    ./apps/api
    ./apps/operator
    ./apps/workspace-manager
    ./apps/cli
    ./pkg/api
    ./pkg/auth
    ./pkg/operator
    ./pkg/workspace
)
```

Cada `go.mod` tiene `replace` directives para referenciar los paquetes locales:

```go
// apps/api/go.mod
module github.com/nimbuscore/apps/api

go 1.26.5

require (
    github.com/nimbuscore/pkg/api v0.0.0
    github.com/nimbuscore/pkg/auth v0.0.0
    github.com/nimbuscore/pkg/workspace v0.0.0
)

replace (
    github.com/nimbuscore/pkg/api => ../../pkg/api
    github.com/nimbuscore/pkg/auth => ../../pkg/auth
    github.com/nimbuscore/pkg/workspace => ../../pkg/workspace
)
```

**¿Por qué módulos separados en vez de un solo módulo?** Porque cada app se
buildea independientemente en Docker. Si todo estuviera en un solo módulo,
Docker tendría que copiar TODO el código para buildear cualquier app.

### Estructura de un módulo

Tomando `apps/api` como ejemplo:

```
apps/api/
├── cmd/
│   └── main.go              # Punto de entrada
├── internal/
│   ├── cache/               # Redis client
│   ├── handler/             # HTTP handlers (rutas)
│   ├── middleware/          # JWT, audit, RBAC
│   ├── queue/               # RabbitMQ publisher
│   ├── server/              # Router Chi + setup
│   └── store/               # PostgreSQL queries
├── go.mod
└── go.sum
```

### Comandos Go importantes

```bash
go run ./cmd/            # Compila y ejecuta (hot-reload no, pero rápido)
go build -o /bin/api ./cmd/  # Compila a binario
go test ./...            # Tests con race detector
go vet ./...             # Análisis estático (variables no usadas, etc.)
go mod tidy              # Sincroniza go.mod con el código fuente
go mod download          # Descarga dependencias al cache
```

**Diferencia entre `go run` y `go build`:** `go run` compila y ejecuta en un
solo paso, útil para desarrollo. `go build` solo compila, útil para Docker.

---

## 3. Chi — Router HTTP

### ¿Por qué Chi y no Gin o Echo?

- **Ligero** — solo es un router, no un framework completo
- **Compatible con `net/http`** — usa `http.Handler` nativo. Cualquier
  middleware de Go funciona
- **Middleware modular** — cada middleware es una función
  `func(http.Handler) http.Handler`
- **Sin magia** — no usa reflection ni code generation. Lo que ves es lo que hay
- **Chi v5** — maduro, estable, usado en producción

### Cómo se usa

```go
// apps/api/internal/server/routes.go

func (s *Server) Routes() http.Handler {
    r := chi.NewRouter()

    // Middleware global (se ejecuta en TODAS las rutas)
    r.Use(chimw.RequestID)        // Agrega X-Request-ID a cada request
    r.Use(chimw.RealIP)           // Lee X-Forwarded-For para IP real
    r.Use(middleware.Logger)       // Loggea cada request en JSON
    r.Use(chimw.Recoverer)         // Atrapa panics y devuelve 500
    r.Use(chimw.Timeout(30 * time.Second)) // Timeout global

    // CORS (permite requests del frontend)
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins: []string{"*"},
        AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    }))

    // Rutas públicas (sin JWT)
    r.Get("/health", h.Health)
    r.Get("/metrics", expvar.Handler().ServeHTTP)

    // Grupo /auth (público)
    r.Route("/auth", func(r chi.Router) {
        r.Get("/login", h.AuthLogin)    // Redirect a Dex
        r.Get("/callback", h.AuthCallback) // Callback OIDC
    })

    // Grupo /api (protegido con JWT)
    r.Route("/api", func(r chi.Router) {
        r.Use(jwtMiddleware)             // Verifica JWT en cada ruta
        r.Use(auditMiddleware)           // Loggea acciones en audit_log
        r.Use(middleware.RBACInjector)   // Inyecta permisos del usuario

        r.Route("/workspaces", func(r chi.Router) {
            r.Get("/", h.ListWorkspaces)   // GET /api/workspaces
            r.Post("/", h.CreateWorkspace) // POST /api/workspaces
            r.Get("/{id}", h.GetWorkspace) // GET /api/workspaces/{id}
        })
    })

    return r
}
```

**Cómo funciona Chi por dentro:**

1. Recibe un request HTTP
2. Ejecuta middleware global en orden (RequestID → RealIP → Logger → Recoverer →
   Timeout)
3. Matchea la ruta: `GET /api/workspaces/123` → handler `ListWorkspaces`
4. Extrae parámetros de la URL: `chi.URLParam(r, "id")` → `"123"`
5. Ejecuta el handler, que escribe la respuesta

### Diferencia entre middlewares

| Middleware          | Cuándo se ejecuta | Qué hace                                                |
| ------------------- | ----------------- | ------------------------------------------------------- |
| `RequestID`         | Cada request      | Genera UUID para trazabilidad                           |
| `Logger`            | Cada request      | Loggea method, path, status, duration                   |
| `Recoverer`         | Solo en panic     | Captura el panic, loggea stack trace, devuelve 500      |
| `Timeout`           | Cada request      | Cancela el context si supera 30s                        |
| `CORS`              | Cada request      | Agrega headers CORS a la respuesta                      |
| `jwtMiddleware`     | Solo en `/api/*`  | Valida JWT, extrae claims, los inyecta en context       |
| `RequirePermission` | Por ruta          | Verifica que el rol del user tenga el permiso necesario |

### Ejemplo: request completo

```bash
curl -s http://localhost:8080/api/workspaces \
  -H "Authorization: Bearer eyJ..."

# Lo que pasa internamente:
# 1. Chi recibe GET /api/workspaces
# 2. CORS → agrega Access-Control-Allow-Origin: *
# 3. Logger → loggea "request GET /api/workspaces"
# 4. jwtMiddleware → decodifica JWT, extrae user_id
# 5. RBACInjector → mapea rol a permisos
# 6. ListWorkspaces → consulta DB y devuelve JSON
# 7. Logger → loggea "response 200 5.2ms"
```

---

## 4. controller-runtime — Operador Kubernetes

### ¿Por qué controller-runtime?

- **Estándar de facto** para operadores Kubernetes (usado por cert-manager,
  etc.)
- **Reconciliation loop** — el operador constantemente compara el estado actual
  con el deseado
- **Informers + caches** — escucha cambios en tiempo real via watch de la API de
  K8s
- **Leader election** — solo un operador reconcilia a la vez (evita duplicados)
- **Metrics integradas** — expone métricas Prometheus automáticamente (reconcile
  time, errors, etc.)

### Cómo funciona el pattern de operador

```
1. Algo crea un CRD (Custom Resource Definition)
   Ej: kubectl apply -f workspace.yaml

2. Controller-runtime detecta el nuevo recurso via Informer

3. Llama a Reconcile(ctx, Request{NamespacedName: "default/mi-ws"})

4. El reconciler:
   a. Fetch del recurso (mi-ws)
   b. Compara status actual vs deseado
   c. Crea/actualiza/elimina recursos K8s (Pods, Services, Ingresses)
   d. Actualiza el status del CRD
```

### Ejemplo concreto: WorkspaceReconciler

```go
// apps/operator/internal/controller/workspace_controller.go

func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Obtener el workspace CRD
    var ws nimbusv1.Workspace
    if err := r.Get(ctx, req.NamespacedName, &ws); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Decidir qué hacer según el status
    switch ws.Status.Phase {
    case "pending":
        // Crear namespace si no existe
        // Crear Pod con la imagen del DevContainer
        // Crear Service ClusterIP
        // Crear Ingress por cada puerto definido
        ws.Status.Phase = "running"

    case "stopping":
        // Eliminar Pod
        // Eliminar Service
        // Eliminar Ingress
        ws.Status.Phase = "stopped"

    case "deleting":
        // Eliminar todo incluyendo namespace
    }

    // 3. Actualizar status en K8s
    r.Status().Update(ctx, &ws)
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

### CRDs definidos

```yaml
# workspaces.nimbuscore.io — representa un DevContainer en ejecución
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
status:
  phase: running
  port_urls:
    - name: app
      url: http://app-mi-dev-container.nimbuscore.local
  pod_name: ws-mi-dev-container-7f8c9
```

```yaml
# prebuilds.nimbuscore.io — representa una prebuild de DevContainer
apiVersion: nimbuscore.io/v1
kind: Prebuild
metadata:
  name: mi-prebuild
spec:
  repo_url: https://github.com/user/repo
  branch: main
  devcontainer_path: .devcontainer/devcontainer.json
status:
  phase: completed
  job_name: prebuild-mi-prebuild-abc12
```

### Leader election

El operador usa `coordination.k8s.io/Leases` para elegir un líder entre
réplicas. Solo el líder reconcilia. Esto evita que dos operadores creen
duplicados.

### Comandos

```bash
# Correr operador localmente (usa ~/.kube/config)
cd apps/operator && go run ./cmd/

# Ver workspaces como CRDs
kubectl get ws -A
kubectl describe ws mi-dev-container
```

---

## 5. Astro 7 — Framework del frontend

### ¿Por qué Astro?

- **Zero JS por defecto** — la página se renderiza como HTML estático. El JS
  solo se carga para componentes interactivos (islas)
- **Islas (islands)** — puedes usar React, Vue, Svelte, etc. para componentes
  específicos sin cargar un SPA entero
- **Rendimiento** — las páginas cargan instantáneamente porque son HTML puro
- **Mismo equipo de Vite** — usa Vite 7 como build tool internamente

### Diferencia Astro vs Next.js vs SPA

| Característica | Astro                          | Next.js                | SPA (Vite+React)        |
| -------------- | ------------------------------ | ---------------------- | ----------------------- |
| JS en cliente  | Mínimo (islas)                 | Hidrata toda la página | Toda la app             |
| SSR/SSG        | Static por defecto             | SSR o Static           | Solo client-side        |
| Bundle inicial | HTML (0 KB JS)                 | HTML + JS del router   | ~100 KB+ de JS          |
| Frameworks UI  | Múltiples (React, Vue, Svelte) | Solo React             | El que elijas           |
| Ideal para     | Dashboards, docs, contenido    | Apps con mucho SSR     | Apps tipo SPA complejas |

Elegimos Astro porque el dashboard es **90% contenido estático** con algunas
islas interactivas (login, crear workspace). No necesitamos un SPA completo.

### Cómo funciona

Archivo `.astro`:

```astro
---
// 🟢 Código que corre en el servidor (solo en build/SSR)
// Aquí puedes hacer fetch a la API, consultar DB, etc.
import Login from '../components/Login'
import BaseLayout from '../layouts/BaseLayout.astro'

const data = await fetch('http://api/...') // Esto corre en el servidor
---

<!-- 🟢 HTML que se renderiza en el servidor -->
<BaseLayout title='Sign In'>
  <!-- client:load → esta isla se hidrata en el cliente -->
  <Login client:load />
</BaseLayout>

<style>
  /* 🟢 CSS con scope (solo esta página) */
  .login-btn {
    color: theme('colors.neon');
  }
</style>
```

### Directivas de hidratación

| Directiva        | Cuándo se hidrata           | Uso                               |
| ---------------- | --------------------------- | --------------------------------- |
| `client:load`    | Inmediatamente al cargar    | Componentes visibles tipo navbar  |
| `client:idle`    | Cuando el browser está idle | Componentes no críticos           |
| `client:visible` | Cuando entra en viewport    | Lazy loading                      |
| `client:media`   | Según media query           | Componentes responsive            |
| `client:only`    | Solo en cliente, no SSR     | Componentes que usan localStorage |

Usamos `client:load` para `App.tsx`, `Dashboard.tsx`, `AdminPanel.tsx` porque
dependen de datos del usuario (JWT, workspaces).

### SSR vs Static en Astro 7

```js
// astro.config.mjs
export default defineConfig({
  output: 'static' // Genera HTML estático en build
  // output: 'server'  // SSR en cada request (no usado aquí)
})
```

En modo `static`, Astro genera archivos HTML en `dist/` durante el build. Sirve
con cualquier servidor estático (nginx, s3, etc.). No necesita Node.js en
runtime.

### Proxy de desarrollo (Vite)

```js
// astro.config.mjs
vite: {
    server: {
        proxy: {
            '/api': { target: 'http://localhost:8080', changeOrigin: true },
            '/auth': { target: 'http://localhost:8080', changeOrigin: true }
        }
    }
}
```

En desarrollo, Astro/Vite sirve el frontend en `:5173` y proxy los requests
`/api/*` y `/auth/*` al backend en `:8080`. Así el frontend no necesita CORS en
desarrollo.

---

## 6. React 19 — Islas interactivas

### ¿Por qué React?

- **Ecosistema maduro** — hooks, componentes, tools
- **Equipo familiarizado** — el equipo conoce React, no hay curva de aprendizaje
- **React 19** — última versión estable con mejoras en hidratación y concurrent
  mode

### Cómo se integra con Astro

Astro usa `@astrojs/react` para montar componentes React en las páginas:

```tsx
// src/components/App.tsx
function App() {
  const [ready, setReady] = useState(false)
  const [authenticated, setAuthenticated] = useState(false)

  useEffect(() => {
    // Corre en el cliente después de la hidratación
    const token = localStorage.getItem('nimbuscore_token')
    if (token) setAuthenticated(true)
    setReady(true)
  }, [])

  if (!ready) return <Spinner />
  return authenticated ? <Dashboard /> : <Login />
}
```

```astro
---
// index.astro
import App from '../components/App'
---

<App client:load />
<!-- Astro hidrata este componente -->
```

**¿Por qué no hacer todo en React?** Porque la página de login y el layout son
HTML estático. Solo el dashboard y admin panel necesitan interactividad (fetch
de datos, estados, etc.). Con Astro, esas páginas cargan instantáneamente y
React solo se activa donde es necesario.

### Componentes React y sus responsabilidades

| Componente        | Archivo               | Responsabilidad                                  |
| ----------------- | --------------------- | ------------------------------------------------ |
| `App`             | `App.tsx`             | Orquestador: decide si mostrar Login o Dashboard |
| `Login`           | `Login.tsx`           | Botón "Sign in with SSO"                         |
| `Dashboard`       | `Dashboard.tsx`       | Lista de workspaces, crear, detener              |
| `WorkspaceCard`   | `WorkspaceCard.tsx`   | Card individual de workspace                     |
| `CreateWorkspace` | `CreateWorkspace.tsx` | Formulario de creación                           |
| `AdminPanel`      | `AdminPanel.tsx`      | Tabla de usuarios y teams                        |
| `api`             | `api.ts`              | Cliente HTTP con JWT (no es componente)          |

### React 19 novedades que usamos

- **`use()`** — permite leer Promises y Context directamente en render
- **Mejor hidratación** — menos errores de hydration mismatch
- **Nuevo `createRoot`** — API más limpia para montar React

### useState vs useReducer

Usamos `useState` para estados simples (loading, error, modales). Si el estado
fuera más complejo (múltiples workspaces con acciones), usaríamos `useReducer`.

---

## 7. Tailwind CSS 4 — Estilos

### ¿Por qué Tailwind CSS 4?

- **Utility-first** — escribes estilos directamente en HTML, no en archivos CSS
  separados
- **v4 es nativo** — usa `@import` y `@theme` en vez de PostCSS config
- **Bundle pequeño** — purga CSS no usado automáticamente (JIT compiler)
- **Consistencia** — todos usan el mismo design system (colores, spacing, fonts)

### Cómo se configura

```css
/* web/dashboard/src/styles/global.css */
@import 'tailwindcss';
@import 'geist/dist/fonts/geist-sans/style.css';
@import 'geist/dist/fonts/geist-mono/style.css';

@theme {
  --color-neon: #00ff41;
  --color-pitch-900: #0a0a0f;
  --color-pitch-800: #12121a;
  --font-family-pixel: 'Geist Pixel', monospace;
  --font-family-sans: 'Geist Sans', sans-serif;
}
```

```js
// astro.config.mjs
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  vite: {
    plugins: [tailwindcss()]
  }
})
```

### Design tokens

| Token          | Valor         | Uso                               |
| -------------- | ------------- | --------------------------------- |
| `bg-pitch-900` | `#0a0a0f`     | Fondo principal (negro casi puro) |
| `text-neon`    | `#00ff41`     | Verde neón (acentos, títulos)     |
| `font-pixel`   | `Geist Pixel` | Títulos pixelados                 |
| `font-sans`    | `Geist Sans`  | Texto general                     |
| `btn-neon`     |               | Botón con borde neón              |

### Ejemplo de componente con Tailwind

```tsx
function Login() {
  return (
    <div className='flex min-h-screen items-center justify-center'>
      <div className='w-full max-w-md space-y-8 px-6 text-center'>
        <h1 className='font-pixel text-4xl leading-tight'>
          NIMBUS<span className='text-neon'>CORE</span>
        </h1>
        <button
          onClick={api.login}
          className='btn btn-neon w-full px-6 py-3 text-sm font-semibold'
        >
          Sign in with SSO
        </button>
      </div>
    </div>
  )
}
```

### ¿Qué significan esas clases?

| Clase            | Significado                                   |
| ---------------- | --------------------------------------------- |
| `flex`           | `display: flex`                               |
| `min-h-screen`   | `min-height: 100vh`                           |
| `items-center`   | `align-items: center`                         |
| `justify-center` | `justify-content: center`                     |
| `w-full`         | `width: 100%`                                 |
| `max-w-md`       | `max-width: 28rem`                            |
| `space-y-8`      | `> * + * { margin-top: 2rem }`                |
| `px-6`           | `padding-left: 1.5rem; padding-right: 1.5rem` |
| `text-center`    | `text-align: center`                          |
| `font-pixel`     | `font-family: Geist Pixel`                    |
| `text-4xl`       | `font-size: 2.25rem`                          |
| `btn-neon`       | Clase custom con borde y texto neón           |

---

## 8. PostgreSQL 17 — Base de datos

### ¿Por qué PostgreSQL?

- **SQL estándar** — cualquier herramienta funciona (psql, pgAdmin, DBeaver)
- **JSONB** — almacenamos puertos y configuraciones como JSON directamente en
  columnas
- **UUID nativo** — IDs como UUID v4, no auto-increment
- **Transacciones ACID** — garantiza consistencia en upserts, quotas, billing
- **Madurez** — 30+ años de desarrollo, comunidad enorme

### Tablas definidas

| Tabla             | Propósito                 | Columnas clave                                        |
| ----------------- | ------------------------- | ----------------------------------------------------- |
| `users`           | Usuarios autenticados     | id, email, provider, provider_id (unique)             |
| `teams`           | Equipos de usuarios       | id, name, slug (unique)                               |
| `workspaces`      | Entornos dev              | id, user_id (FK), team_id (FK), status, ports (JSONB) |
| `quotas`          | Límites por team          | team_id (FK), max_workspaces, max_cpu                 |
| `snapshots`       | Backups de workspaces     | id, workspace_id (FK), status, size                   |
| `prebuilds`       | Prebuilds de DevContainer | id, user_id (FK), repo_url, status                    |
| `audit_log`       | Auditoría de acciones     | id, action, user_id, ip_address                       |
| `billing_records` | Facturación por uso       | id, workspace_id (FK), cost, duration_min             |
| `clusters`        | Clusters K8s registrados  | id, name, api_endpoint, region                        |

### Esquema de workspaces

```sql
CREATE TABLE workspaces (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    team_id UUID REFERENCES teams(id),
    image TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    cpu TEXT NOT NULL,
    memory TEXT NOT NULL,
    disk TEXT NOT NULL,
    gpu INT NOT NULL DEFAULT 0,
    repo_url TEXT,
    branch TEXT DEFAULT 'main',
    ports JSONB NOT NULL DEFAULT '[]',  -- Array de objetos {port, protocol, subdomain}
    last_activity TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**¿Por qué JSONB para ports?** Porque la estructura de puertos varía por
workspace (distinta cantidad, distintos protocolos). JSONB permite flexibilidad
sin crear una tabla separada.

### Cómo se conecta la API

```go
// apps/api/internal/store/connection.go
pool, err := pgxpool.New(ctx, "postgres://nimbuscore:nimbuscore@postgresql:5432/nimbuscore?sslmode=disable")
```

Usamos `pgxpool` para connection pooling (reutiliza conexiones). Config:

```go
MaxConns: 25   // Máximo de conexiones simultáneas
MinConns: 5    // Mínimo para evitar cold starts
```

### Consultas comunes

```go
// Upsert (insert o update si existe)
func UpsertUser(ctx context.Context, db *pgxpool.Pool, u *api.User) error {
    err := db.QueryRow(ctx, `
        INSERT INTO users (id, email, name, role, provider, provider_id, avatar_url, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT (provider, provider_id) DO UPDATE SET
            email = EXCLUDED.email,
            name = EXCLUDED.name,
            updated_at = EXCLUDED.updated_at
        RETURNING id
    `, u.ID, u.Email, u.Name, u.Role, u.Provider, u.ProviderID, u.AvatarURL, u.CreatedAt, u.UpdatedAt).Scan(&u.ID)
    return err
}
```

**`RETURNING id`** — después del upsert, PostgreSQL devuelve el ID real (sea el
insertado o el existente). Asi el JWT siempre tiene el ID correcto.

---

## 9. Redis 7 — Caché

### ¿Por qué Redis?

- **En memoria** — lecturas en microsegundos (< 1 ms)
- **Estructuras de datos** — strings, hashes, sets, sorted sets, streams
- **TTL automático** — expiración de claves sin necesidad de cron
- **Pub/Sub** — para notificaciones en tiempo real (aunque usamos RabbitMQ para
  colas)

### Cómo lo usamos

Actualmente Redis se usa para:

```go
// apps/api/internal/cache/redis.go
rdb := redis.NewClient(&redis.Options{
    Addr: "redis:6379",
})

// Ping para verificar conexión
err := rdb.Ping(ctx).Err()

// Uso futuro planeado:
// - Cache de sesiones OIDC (state)
// - Rate limiting (INCR + EXPIRE)
// - Cache de consultas DB pesadas
// - Almacenamiento temporal de state OAuth
```

### Conexión

```go
func NewRedis(url, password string, db int) *redis.Client {
    opts, err := redis.ParseURL(url)
    if err != nil {
        opts = &redis.Options{
            Addr:     url,
            Password: password,
            DB:       db,
        }
    }
    return redis.NewClient(opts)
}
```

**`redis://redis:6379`** vs **`redis:6379`**: El primero es una URL válida que
`ParseURL` interpreta correctamente. El segundo (sin `//`) hace que `url.Parse`
interprete `redis:` como scheme y `6379` como opaque, resultando en `localhost`
como host.

**Error común:** `rdb.Ping(ctx)` devuelve `*StatusCmd`, no `error`. Siempre usar
`.Err()`:

```go
// ❌ INCORRECTO — siempre entra al if
if err := rdb.Ping(ctx); err != nil { ... }

// ✅ CORRECTO
if err := rdb.Ping(ctx).Err(); err != nil { ... }
```

---

## 10. RabbitMQ 4 — Cola de mensajes

### ¿Por qué RabbitMQ y no NATS/Kafka/Redis PubSub?

| Feature          | RabbitMQ                | NATS               | Kafka    | Redis PubSub |
| ---------------- | ----------------------- | ------------------ | -------- | ------------ |
| Persistencia     | Sí (disco)              | No por defecto     | Sí       | No           |
| Encolamiento     | Sí                      | No (fire & forget) | Sí       | No           |
| Routing flexible | Topics, headers, direct | Subjects           | Topics   | Channels     |
| ACK/reintentos   | Sí                      | Solo en JetStream  | Sí       | No           |
| Madurez          | 15+ años                | 5+ años            | 10+ años | 10+ años     |

Elegimos RabbitMQ porque necesitamos **garantía de entrega** (ACK) y **colas
persistentes** (si el workspace-manager está caído, los eventos no se pierden).
NATS es más rápido pero no garantiza entrega. Kafka es overkill para nuestro
volumen.

### Cómo funciona

```
                  Exchange (topic)
              nimbuscore.workspace
                   │
        ┌──────────┼──────────┐
        │          │          │
        ▼          ▼          ▼
   Queue:      Queue:      Queue:
  create      stop        snapshot
        │          │          │
        ▼          ▼          ▼
   workspace-manager consume ACK
```

### Eventos definidos

```go
// pkg/workspace/events.go
const (
    EventCreate   = "workspace.create"
    EventStart    = "workspace.start"
    EventStop     = "workspace.stop"
    EventDelete   = "workspace.delete"
    EventSnapshot = "workspace.snapshot"
    EventTimeout  = "workspace.timeout"
    EventPrebuild = "workspace.prebuild"
)
```

### Código: publicar evento

```go
// apps/api/internal/queue/rabbitmq.go
ch.Publish(
    "nimbuscore.workspace", // exchange
    "workspace.create",     // routing key
    true,                   // mandatory
    false,
    amqp.Publishing{
        ContentType: "application/json",
        Body:        json.Marshal(event),
        DeliveryMode: amqp.Persistent, // persiste en disco
    },
)
```

### Código: consumir evento

```go
// apps/workspace-manager/internal/manager/events.go
msgs, _ := ch.Consume(
    queue.Name, // cola
    "",         // consumer tag
    false,      // auto-ack → MANUAL ACK
    false, false, false, nil,
)

for msg := range msgs {
    switch msg.RoutingKey {
    case "workspace.create":
        handleCreate(msg.Body)
    case "workspace.stop":
        handleStop(msg.Body)
    }
    msg.Ack(false) // confirmo que procesé
}
```

**ACK manual** — si el workspace-manager crashea mientras procesa, el mensaje
queda en la cola y otro consumidor lo retoma. Sin ACK, el mensaje se pierde.

### RabbitMQ Admin UI

`http://localhost:15672` — user `user`, pass `changeme`. Muestra colas,
conexiones, mensajes en tiempo real.

---

## 11. Dex — Proveedor OIDC

### ¿Por qué Dex?

- **OIDC estándar** — implementa OpenID Connect, compatible con cualquier
  cliente OIDC
- **Múltiples backends** — LDAP, SAML, GitHub, Google, etc. Dex es un "proxy"
  que traduce cualquier auth a OIDC
- **Local dev** — con `enablePasswordDB` podemos tener usuarios locales sin
  depender de un provider externo
- **Gratuito y self-hosted** — a diferencia de Auth0, Okta, etc.

### Configuración

```yaml
# deploy/dex/config.yaml
issuer: http://host.docker.internal:5556/dex # Cómo se llama Dex a sí mismo

storage:
  type: sqlite3
  config:
    file: /var/dex/dex.db

web:
  http: 0.0.0.0:5556

staticClients:
  - id: nimbuscore # Client ID (público)
    redirectURIs:
      - http://localhost:8080/auth/callback # Callback de la API
      - http://localhost:5173 # Callback del frontend
    name: NimbusCore
    secret: nimbuscore-secret

enablePasswordDB: true
staticPasswords:
  - email: admin@nimbuscore.io
    hash: '$2b$10$...' # Hash bcrypt de "admin123"
    username: admin
    userID: '11111111-1111-1111-1111-111111111111'
```

### ¿Por qué `host.docker.internal`?

Dentro de Docker, los contenedores se comunican por nombre de servicio (dex,
api, postgresql). Pero Dex devuelve su `issuer` en el
`.well-known/openid-configuration`, y la API verifica que coincida con el que
espera.

Si el `issuer` es `http://dex:5556/dex`, la API conecta bien. Pero **el
browser** necesita una URL que pueda resolver:

- Dex usa `host.docker.internal` para que coincida con lo que la API usa
  internamente, Y
- La API reemplaza el host en `AuthURL` para que el browser reciba
  `localhost:5556/dex/auth`

### Flujo OIDC detallado

```
1. Browser click "Sign in with SSO"
2. Frontend (App.tsx) llama a API: GET /auth/login
3. API crea provider con OIDC_ISSUER, genera state UUID
4. API redirige browser a:
   http://localhost:5556/dex/auth?
     client_id=nimbuscore&
     redirect_uri=http://localhost:5173&
     response_type=code&
     state=<uuid>
5. Browser llega a Dex, ve formulario de login
6. Usuario ingresa admin@nimbuscore.io / admin123
7. Dex verifica password contra bcrypt en SQLite
8. Dex redirige browser a:
   http://localhost:5173/?code=<auth-code>&state=<uuid>
9. App.tsx detecta `code` en URL, llama a:
   fetch /auth/callback?code=<auth-code>
   (proxied por Vite a API :8080)
10. API intercambia code por token en Dex
11. API upserta usuario en PostgreSQL
12. API emite JWT firmado con JWT_SECRET
13. API devuelve {token, user_id}
14. App.tsx guarda token en localStorage
15. App.tsx redirige a / (autenticado)
```

### Generar hash bcrypt para Dex

```bash
docker run --rm ghcr.io/dexidp/dex:v2.42.0 dex hash -p admin123
```

---

## 12. pnpm — Gestor de paquetes

### ¿Por qué pnpm y no npm/yarn?

- **Disk space** — usa hard links, no duplica `node_modules` por proyecto
- **Velocidad** — hasta 2-3x más rápido que npm
- **Strict** — no permite imports de paquetes no declarados en `package.json`
- **Workspaces nativos** — manejo de monorepo integrado

### Configuración

```yaml
# pnpm-workspace.yaml
packages:
  - 'web/*'
onlyBuiltDependencies:
  - esbuild # Necesario para Vite/Astro
  - sharp # Necesario para imágenes en Astro
```

`onlyBuiltDependencies` — pnpm 11 por defecto no ejecuta scripts de post-install
de dependencias. Hay que aprobar explícitamente `esbuild` y `sharp` que
necesitan compilar binarios nativos.

### Comandos

```bash
pnpm install          # Instala dependencias (lockfile)
pnpm add react        # Agrega dependencia
pnpm remove react     # Elimina dependencia
pnpm dev              # Ejecuta script dev del package.json
pnpm build            # Ejecuta script build
pnpm dlx             # Ejecuta binario sin instalarlo (como npx)
```

### pnpm-lock.yaml vs package-lock.json

pnpm genera `pnpm-lock.yaml` en vez de `package-lock.json`. Cumple la misma
función: lockea versiones exactas de dependencias para builds reproducibles.

---

## 13. Docker / Docker Compose

### ¿Por qué Docker?

- **Entornos idénticos** — lo que funciona en tu máquina funciona en producción
- **Distroless** — imágenes mínimas (~15 MB) sin shell ni vulnerabilidades
- **Compose** — orquestación local con un solo comando

### Dockerfiles multi-stage

```dockerfile
# deploy/Dockerfile.api
FROM golang:1.26.5-alpine AS builder
WORKDIR /build
COPY apps/api/ apps/api/
COPY pkg/api/ pkg/api/
COPY pkg/auth/ pkg/auth/
COPY pkg/workspace/ pkg/workspace/
RUN cd apps/api && go mod download && go build -o /bin/api ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/api /api
ENTRYPOINT ["/api"]
```

**Stage 1 (builder):** Alpine con Go, compila el binario. **Stage 2 (runtime):**
Distroless (solo libc, sin shell, sin package manager). **Resultado:** Imagen de
~15 MB.

**¿Por qué NO copiar `go.work`?** Porque `go.work` es para desarrollo local. En
Docker, cada app se buildea independientemente con `replace` directives en su
`go.mod`.

**¿Por qué copiar `pkg/` pero no `apps/operator`?** Porque la API solo depende
de `pkg/api`, `pkg/auth`, y `pkg/workspace`. Copiar solo lo necesario minimiza
la capa de Docker.

### Docker Compose

```yaml
# compose.yml
services:
  postgresql:
    image: postgres:17-alpine
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./deploy/db/init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U nimbuscore']

  redis:
    image: redis:7-alpine
    healthcheck:
      test: ['CMD', 'redis-cli', 'ping']

  rabbitmq:
    image: rabbitmq:4-management-alpine
    environment:
      RABBITMQ_DEFAULT_USER: user
      RABBITMQ_DEFAULT_PASS: changeme

  dex:
    image: ghcr.io/dexidp/dex:v2.42.0
    volumes:
      - ./deploy/dex/config.yaml:/etc/dex/config.yaml:ro
    command: ['dex', 'serve', '/etc/dex/config.yaml']

  api:
    build:
      context: .
      dockerfile: deploy/Dockerfile.api
    ports:
      - '8080:8080'
    extra_hosts:
      - 'host.docker.internal:host-gateway'
    depends_on:
      postgresql: { condition: service_healthy }
      redis: { condition: service_healthy }
      rabbitmq: { condition: service_healthy }
      dex: { condition: service_started }
```

**`depends_on` con `condition: service_healthy`:** Espera a que el servicio esté
listo (healthcheck pasa) antes de iniciar la API. Sin esto, la API intentaría
conectar antes de que PostgreSQL/Redis estén listos.

**`extra_hosts`:** Permite que `host.docker.internal` resuelva al host Docker
(Linux). Dex necesita esto para que la API y el browser puedan acceder a él con
diferentes URLs.

### Comandos Docker

```bash
docker compose up -d          # Levanta servicios en background
docker compose up -d --build  # Reconstruye imágenes y levanta
docker compose logs -f api    # Sigue logs de la API
docker compose down           # Detiene servicios
docker compose down -v        # Detiene y borra volúmenes
docker compose exec api sh    # Ejecuta shell dentro del container
docker compose ps             # Lista servicios y su estado
```

---

## 14. Helm — Paquete Kubernetes

### ¿Por qué Helm?

- **Plantillas parametrizables** — un mismo chart se despliega en dev, staging,
  prod con diferentes values
- **Dependencias** — PostgreSQL, Redis, RabbitMQ como subcharts de Bitnami
- **Rollback** — `helm rollback` vuelve a versión anterior
- **Ecosistema** — charts público, ArgoCD compatible

### Estructura del chart

```
deploy/helm/nimbuscore/
├── Chart.yaml              # Metadatos (nombre, versión, dependencias)
├── values.yaml             # Valores por defecto (producción)
├── values.dev.yaml         # Override para desarrollo (menos réplicas, sin monitoring)
├── values.prod.yaml        # Override para producción
├── crds/                   # CRDs de NimbusCore (workspaces, prebuilds)
├── dashboards/             # Grafana dashboards + Prometheus rules
│   ├── grafana-dashboard.json
│   └── prometheus-rules.yaml
├── argocd/                 # ApplicationSet + Project para ArgoCD
│   ├── application-set.yaml
│   └── project.yaml
└── templates/
    ├── _helpers.tpl            # Funciones auxiliares (labels, names)
    ├── configmap-api.yaml      # ConfigMap con env vars de la API
    ├── configmap-operator.yaml # ConfigMap del operator
    ├── configmap-manager.yaml  # ConfigMap del workspace-manager
    ├── deployment-api.yaml     # Deployment de la API
    ├── deployment-operator.yaml # Deployment del operator
    ├── deployment-workspace-manager.yaml # Deployment del manager
    ├── hpa.yaml               # Horizontal Pod Autoscaler
    ├── ingress.yaml           # Ingress para la API
    ├── service.yaml           # Services (api, operator, manager)
    ├── service-monitor.yaml   # ServiceMonitor para Prometheus (deshabilitado en dev)
    ├── clusterrole.yaml       # ClusterRole para RBAC
    ├── clusterrolebinding.yaml # ClusterRoleBinding
    ├── secret.yaml            # Secrets (JWT, OIDC, Restic)
    └── pdb.yaml               # PodDisruptionBudget
```

### Instalación

```bash
# 1. Preparar dependencias (Bitnami charts)
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update deploy/helm/nimbuscore/

# 2. Instalar/actualizar
helm upgrade --install nimbuscore deploy/helm/nimbuscore \
  --values deploy/helm/nimbuscore/values.dev.yaml \
  --set global.ingress.host="nimbuscore.local"
```

**`--install`:** Si el release no existe, lo crea. Si existe, lo actualiza.

### Values explicados

```yaml
# values.yaml (producción)
api:
  replicas: 3 # 3 réplicas para alta disponibilidad
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilization: 70 # Escala cuando CPU > 70%

postgresql:
  enabled: true # Usar subchart de Bitnami
  auth:
    database: nimbuscore
    username: nimbuscore
  primary:
    persistence:
      size: 20Gi # Disco persistente

monitoring:
  serviceMonitor:
    enabled: true # Crear ServiceMonitor para Prometheus

ingress:
  hosts:
    - host: api.nimbuscore.io
      paths:
        - path: /
          pathType: Prefix
```

```yaml
# values.dev.yaml (desarrollo)
api:
  replicas: 1
  autoscaling:
    enabled: false # Sin autoscaling en dev

monitoring:
  serviceMonitor:
    enabled: false # Sin Prometheus en dev
```

---

## 15. Taskfile — Runner de tareas

### ¿Por qué Taskfile en vez de Makefile?

- **YAML en vez de Make** — más legible y fácil de mantener
- **Task runner moderno** — dependencias entre tasks, variables, dotenv
- **Multiplataforma** — funciona igual en Linux, macOS, Windows (sin bash)

### Tasks definidas

```yaml
# Taskfile.yml
version: '3'

tasks:
  default:
    desc: Listar todas las tareas disponibles
    cmd: task --list

  go:test:
    desc: Ejecutar tests Go con race detector
    cmd:
      for dir in apps/* pkg/*; do (cd "$dir" && go test -race -count=1 ./...);
      done

  go:vet:
    desc: Ejecutar go vet en todos los módulos
    cmd: for dir in apps/* pkg/*; do (cd "$dir" && go vet ./...); done

  web:dev:
    desc: Iniciar servidor de desarrollo Astro
    dir: web/dashboard
    cmd: pnpm dev

  docker:build:
    desc: Construir todas las imágenes Docker
    cmds:
      - docker compose build

  ci:
    desc: Ejecutar CI completo
    cmds:
      - task: go:vet
      - task: go:lint
      - task: go:test
      - task: format
      - task: web:build
```

### Cómo ejecutar

```bash
task              # Lista todas las tasks
task go:test      # Ejecuta tests
task web:dev      # Inicia Astro
task ci           # CI completo
```

### Diferencia con `go run ./cmd/`

```bash
# Directo
cd apps/api && go run ./cmd/

# Via Taskfile
task api:run      # Si estuviera definido (no lo tenemos, pero es patrón común)
```

Taskfile no es obligatorio — es solo azúcar. Puedes ejecutar `go run`,
`pnpm dev`, `docker compose` directamente.

---

## 16. Estructura del monorepo

```
nimbuscore/
├── apps/
│   ├── api/                    # API REST (Chi)
│   │   ├── cmd/main.go         # Entry point
│   │   ├── internal/
│   │   │   ├── cache/          # Redis
│   │   │   ├── handler/        # HTTP handlers
│   │   │   │   ├── admin.go    # Clusters, audit log
│   │   │   │   ├── auth.go     # OIDC login/callback
│   │   │   │   ├── quota.go    # Quotas CRUD
│   │   │   │   ├── team.go     # Teams CRUD
│   │   │   │   ├── user.go     # Users CRUD + me
│   │   │   │   └── workspace.go # Workspaces CRUD
│   │   │   ├── middleware/     # JWT, audit, RBAC
│   │   │   ├── queue/          # RabbitMQ publisher
│   │   │   ├── server/         # Router Chi
│   │   │   └── store/          # PostgreSQL queries
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   ├── operator/               # Operador Kubernetes
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   └── controller/     # Reconcilers (workspace, prebuild)
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   ├── workspace-manager/      # Event loop + snapshots
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── manager/        # Event handlers
│   │   │   ├── snapshot/       # Restic operations
│   │   │   └── watcher/        # Idle timeout
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   └── cli/                    # CLI (futuro)
│       ├── go.mod
│       └── go.sum
│
├── pkg/
│   ├── api/                    # Tipos compartidos (User, Workspace, WorkspaceStatus, etc.)
│   ├── auth/                   # OIDC provider, JWT issuer, middleware
│   ├── operator/               # CRD tipos + scheme + deepcopy
│   └── workspace/              # Lifecycle state machine + eventos + config
│
├── web/
│   └── dashboard/              # Frontend Astro
│       ├── src/
│       │   ├── components/     # React components (App, Login, Dashboard, etc.)
│       │   ├── layouts/        # Astro layouts
│       │   ├── pages/          # Astro pages (index, login, admin)
│       │   ├── styles/         # CSS global + Tailwind
│       │   └── types/          # TypeScript interfaces
│       ├── public/
│       │   └── fonts/          # Geist fonts
│       ├── astro.config.mjs
│       ├── package.json
│       └── tsconfig.json
│
├── deploy/
│   ├── compose.yml             # Docker Compose (stack local)
│   ├── Dockerfile.api
│   ├── Dockerfile.operator
│   ├── Dockerfile.workspace-manager
│   ├── db/
│   │   └── init.sql            # Schema PostgreSQL
│   ├── dex/
│   │   └── config.yaml         # Configuración Dex
│   └── helm/nimbuscore/        # Helm chart completo
│
├── .github/workflows/
│   └── ci.yml                  # CI pipeline
│
├── Taskfile.yml
├── go.work
└── pnpm-workspace.yaml
```

### ¿Por qué `apps/` y `pkg/`?

| Directorio | Qué contiene                         | Quién lo usa           |
| ---------- | ------------------------------------ | ---------------------- |
| `apps/*`   | Aplicaciones ejecutables (binarios)  | El usuario final       |
| `pkg/*`    | Librerías compartidas                | Las apps entre sí      |
| `web/*`    | Frontend (independiente del backend) | El usuario vía browser |

Cada `apps/*` y `pkg/*` es un módulo Go independiente con su propio `go.mod`. Se
comunican via `replace` directives.

### ¿Por qué `web/` no está adentro de `apps/`?

Porque `apps/` es para Go (microservicios) y `web/` es TypeScript. Tienen
toolchains diferentes. Mezclarlos en el mismo directorio sería confuso.

---

## 17. Flujo completo: login → workspace

### Paso 1: Infraestructura

```bash
# Terminal 1: Levantar servicios
docker compose up -d
# → PostgreSQL :5432, Redis :6379, RabbitMQ :5672, Dex :5556

# Terminal 2: Frontend
cd web/dashboard && pnpm dev
# → Astro :5173

# Terminal 3: API (opcional, también via compose)
# docker compose up -d --build api
# → API :8080
```

### Paso 2: Autenticación

```
Browser → http://localhost:5173/
  ↓ Ve pantalla de login con botón "Sign in with SSO"
  ↓ Click
  ↓ GET /auth/login (proxied a API :8080)
  ↓ 307 Redirect a http://localhost:5556/dex/auth?...
  ↓ Login con admin@nimbuscore.io / admin123
  ↓ 302 Redirect a http://localhost:5173/?code=xxx
  ↓ App.tsx detecta code, llama a GET /auth/callback?code=xxx
  ↓ API intercambia code por token en Dex
  ↓ API upserta usuario en PostgreSQL
  ↓ API devuelve {token, user_id}
  ↓ App.tsx guarda en localStorage
  ↓ Redirige a /
  ↓ Muestra Dashboard con "0 Running" y "0 Total"
```

### Paso 3: Crear workspace

```
Dashboard → Click "New Workspace"
  ↓ Formulario con name, image, repo, resources, ports
  ↓ Click "Create"
  ↓ POST /api/workspaces (con JWT en Authorization)
  ↓ API valida JWT, extrae user_id
  ↓ API upserta workspace en PostgreSQL (status: "pending")
  ↓ API publica evento "workspace.create" en RabbitMQ
  ↓ API devuelve workspace JSON
  ↓ Frontend agrega a la lista
```

### Paso 4: Kubernetes reconcilia (si hay cluster)

```
Operator detecta nuevo workspace via Informer
  ↓ Crea Namespace (nimbuscore-ws-<uuid>)
  ↓ Crea Pod DevContainer
  ↓ Crea Service ClusterIP
  ↓ Crea Ingress por cada puerto
  ↓ Actualiza status del CRD a "running"
  ↓ API recibe el cambio via watch o polling
  ↓ Frontend muestra workspace como "running"
```

### Paso 5: Uso y limpieza

```
Usuario accede a http://app-mi-ws.nimbuscore.local
  ↓ Trabaja en su DevContainer
  ↓ Heartbeat cada pocos minutos (POST /heartbeat)
  ↓
  │ Si idle > 20 min:
  │   workspace-manager publica "workspace.timeout"
  │   Operator elimina Pod/Service/Ingress
  │   status → "stopped"
  │
  │ Si usuario hace click "Stop":
  │   POST /api/workspaces/{id}/stop
  │   API publica "workspace.stop"
  │   workspace-manager hace backup Restic
  │   Operator elimina recursos K8s
  │   status → "stopped"
```

---

## 18. Glosario de comandos

### Go

| Comando                        | Explicación                                                                                                    |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------- |
| `go run ./cmd/`                | Compila y ejecuta el paquete `main` en `./cmd/`. Útil para desarrollo. No deja binario.                        |
| `go build -o /bin/api ./cmd/`  | Compila el paquete `main` en `./cmd/` y escribe el binario en `/bin/api`.                                      |
| `go test -race -count=1 ./...` | Ejecuta todos los tests con race detector (`-race`) sin cache (`-count=1`).                                    |
| `go vet ./...`                 | Análisis estático: detecta variables no usadas, llamadas sospechosas, etc.                                     |
| `go mod tidy`                  | Sincroniza `go.mod` con el código fuente. Agrega dependencias faltantes, elimina no usadas, regenera `go.sum`. |
| `go mod download`              | Descarga todas las dependencias al cache local sin compilar. Útil en Docker para cachear capas.                |

### pnpm

| Comando          | Explicación                                                                      |
| ---------------- | -------------------------------------------------------------------------------- |
| `pnpm install`   | Instala dependencias según `pnpm-lock.yaml`. Crea `node_modules` con hard links. |
| `pnpm dev`       | Ejecuta el script `dev` del `package.json` (Astro dev server).                   |
| `pnpm build`     | Ejecuta el script `build` (Astro build a HTML estático).                         |
| `pnpm add react` | Agrega `react` como dependencia al `package.json` y lockfile.                    |
| `pnpm dlx`       | Ejecuta un binario sin instalarlo globalmente (similar a `npx`).                 |

### Docker

| Comando                        | Explicación                                                        |
| ------------------------------ | ------------------------------------------------------------------ |
| `docker compose up -d`         | Levanta servicios definidos en `compose.yml` en background (`-d`). |
| `docker compose up -d --build` | Reconstruye imágenes (`--build`) y levanta servicios.              |
| `docker compose logs -f api`   | Sigue logs del servicio `api` en tiempo real.                      |
| `docker compose down`          | Detiene y elimina containers, networks. NO elimina volúmenes.      |
| `docker compose down -v`       | Lo mismo que `down` pero también elimina volúmenes (pierde datos). |
| `docker compose exec api sh`   | Ejecuta `sh` dentro del container `api`. Útil para debug.          |
| `docker compose ps`            | Muestra estado de los servicios (running, exited, etc.).           |

### Kubernetes / Helm

| Comando                                       | Explicación                                                                                       |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `helm upgrade --install nimbuscore ...`       | Crea o actualiza un release de Helm. Idempotente: si no existe, lo crea; si existe, lo actualiza. |
| `helm uninstall nimbuscore`                   | Elimina el release y todos los recursos K8s asociados.                                            |
| `helm dependency update`                      | Descarga dependencias (subcharts Bitnami) según `Chart.yaml`.                                     |
| `kubectl apply -f file.yaml`                  | Crea o actualiza recursos K8s desde un archivo YAML.                                              |
| `kubectl get pods -A`                         | Lista pods en todos los namespaces (`-A`).                                                        |
| `kubectl describe ws mi-ws`                   | Muestra detalle del CRD Workspace.                                                                |
| `kubectl delete crd workspaces.nimbuscore.io` | Elimina el CRD y todos los recursos de ese tipo.                                                  |

### Taskfile

| Comando        | Explicación                                                     |
| -------------- | --------------------------------------------------------------- |
| `task`         | Lista todas las tareas disponibles con descripción.             |
| `task go:test` | Ejecuta tests Go en todos los módulos.                          |
| `task go:vet`  | Ejecuta go vet en todos los módulos.                            |
| `task format`  | Formatea código con Prettier.                                   |
| `task web:dev` | Inicia servidor de desarrollo Astro.                            |
| `task ci`      | Ejecuta CI completo (vet + lint + test + build + format + web). |

### Utilidades

| Comando                                                           | Explicación                                                                                    |
| ----------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `curl -s http://localhost:8080/health`                            | Health check de la API. Responde `{"status":"ok"}`                                             |
| `curl -s http://localhost:8080/metrics`                           | Métricas expvar (JSON). Muestra memstats, contadores, gauges.                                  |
| `curl http://localhost:5556/dex/.well-known/openid-configuration` | Configuración OIDC de Dex. Verifica que Dex responde.                                          |
| `docker compose logs -f api \| grep ERROR`                        | Filtra solo errores de la API.                                                                 |
| `eval "$(mise activate zsh)"`                                     | Activa mise (gestor de versiones) en la shell actual. Necesario para encontrar Go, Node, pnpm. |
