<div align="center">
  <h1>NimbusCore</h1>
  <p><strong>Plataforma self-hosted de DevContainers on-demand sobre Kubernetes.</strong></p>
  <p>Como GitHub Codespaces, pero corriendo en tu propia infraestructura.</p>
  <br>
  <p>
    <img src="https://img.shields.io/badge/Go-1.26.5-00ADD8?logo=go" alt="Go">
    <img src="https://img.shields.io/badge/Astro-7.0.6-BC52EE?logo=astro" alt="Astro">
    <img src="https://img.shields.io/badge/Node-24.18_LTS-339933?logo=nodedotjs" alt="Node">
    <img src="https://img.shields.io/badge/Vite_8+Rolldown-646CFF?logo=vite" alt="Vite">
    <img src="https://img.shields.io/badge/React_19-61DAFB?logo=react" alt="React">
    <img src="https://img.shields.io/badge/Tailwind_4-06B6D4?logo=tailwindcss" alt="Tailwind">
    <img src="https://img.shields.io/badge/Prettier-3.9.5-F7B93E?logo=prettier" alt="Prettier">
    <img src="https://img.shields.io/badge/Oxlint-1.73-5C2D91?logo=oxlint" alt="Oxlint">
    <img src="https://img.shields.io/badge/Kubernetes-326CE5?logo=kubernetes" alt="K8s">
    <img src="https://img.shields.io/badge/license-MIT-yellow" alt="License">
  </p>
  <br>
</div>

---

## Tabla de Contenidos

- [¿Por qué NimbusCore?](#por-qué-nimbuscore)
- [El Problema](#el-problema)
- [La Solución](#la-solución)
- [Arquitectura](#arquitectura)
- [Stack Tecnológico](#stack-tecnológico)
- [Roadmap](#roadmap)
- [Estructura del Proyecto](#estructura-del-proyecto)
- [Convenciones de Commits](#convenciones-de-commits)
- [Empezando](#empezando)
- [Comandos](#comandos)
- [Licencia](#licencia)

---

## ¿Por qué NimbusCore?

Los entornos de desarrollo en la nube (Codespaces, Gitpod, Coder) transformaron
la forma de trabajar: cero setup, entornos efímeros, colaboración instantánea.
Pero tienen un costo elevado, dependencia de proveedores externos, y
limitaciones de personalización cuando necesitas hardware específico, GPUs, o
integración directa con tu cluster on-premise.

NimbusCore nace para equipos que quieren **el control total** de su
infraestructura de desarrollo sin sacrificar la experiencia de desarrollo
moderna.

---

## El Problema

| Problema                   | Impacto                                                                                                                       |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| **Costo**                  | Codespaces cuesta ~$0.50/h por máquina mediana. Un equipo de 20 devs trabajando 8h/día = ~$2,400/mes. Escalar es prohibitivo. |
| **Dependencia externa**    | Si GitHub/Gitlab tiene una caída, tu equipo no puede desarrollar.                                                             |
| **Hardware específico**    | ¿Necesitas GPUs para ML, ARM para cross-compile, o un kernel personalizado? Los proveedores cloud lo limitan.                 |
| **Integración on-premise** | Tu código, secretos, monitoreo y compliance no pueden salir de tu red.                                                        |
| **Setup local**            | Cada nuevo dev pierde días configurando toolchains, dependencias, y permisos.                                                 |

## La Solución

NimbusCore despliega **entornos de desarrollo completos** (VS Code en el
navegador + terminal + port forwarding) sobre tu propio clúster Kubernetes, bajo
demanda:

- **Cada workspace es efímero y aislado**: un namespace de K8s por
  desarrollador, con recursos limitados y cleanup automático.
- **Arranque en segundos**: imágenes DevContainer pre-cacheadas + init container
  que clona el repo.
- **Persistencia opcional**: snapshots del home directory a S3 con restic,
  restore on demand.
- **Port forwarding**: expone puertos del container como subdominios automáticos
  (`app-8080.user.dev.example.com`).
- **Auto-stop**: idle >20 min → hiberna. Sin consumo cuando no se usa.
- **Multi-cluster**: los workspaces se distribuyen entre clusters (dev, staging,
  spot, on-premise).
- **Seguro**: network policies, OIDC SSO, audit log, y secretos cifrados.

### Casos de uso

- **Equipos remotos** — desarrollo desde cualquier máquina, solo necesitan un
  navegador
- **Formación y workshops** — entornos precargados para 50 asistentes en minutos
- **CI preview environments** — workspace por PR para revisar cambios antes de
  merge
- **Desarrollo con hardware específico** — GPUs para ML, ARM para IoT, kernel
  tuning
- **Open source** — contributors sin setup, solo un clic desde el README

---

## Arquitectura

```
Usuario ──▶ Dashboard (Astro + React)
              │
              ▼
          API Server (Go/Chi)
              │
              ├── PostgreSQL ── Workspaces, Users, Teams, Quotas
              ├── Redis ────── Sessions, Heartbeats, Cache
              ├── RabbitMQ ── Task queue (builds, snapshots)
              │
              ▼
    ┌──────────────────────────┐
    │   K8s Operator (Go)      │
    │   controller-runtime     │
    └──────────────────────────┘
              │
              ▼
    ┌──────────────────────────┐
    │  Namespace por workspace  │
    │                          │
    │  ┌──────────────────┐    │
    │  │ code-server       │    │
    │  │ (VS Code web)     │    │
    │  ├──────────────────┤    │
    │  │ Init container   │    │
    │  │ (git clone repo)  │    │
    │  ├──────────────────┤    │
    │  │ Restic sidecar    │    │
    │  │ (snapshots a S3)  │    │
    │  ├──────────────────┤    │
    │  │ PVC (home dir)    │    │
    │  └──────────────────┘    │
    │                          │
    │  Ingress: user.dev.empresa.com  │
    └──────────────────────────┘
```

**Stack del workspace:**

```
┌────────────────────────────────────────┐
│           code-server                  │
│    VS Code completo en el navegador    │
├────────────────────────────────────────┤
│  Extensiones: Go, Python, Rust, Node   │
│  Runtimes: golang, node, python, rust  │
│  CLI: kubectl, docker, helm, gh        │
├────────────────────────────────────────┤
│  Prebuild: imagen cacheada por repo    │
│  Init: git clone del repositorio       │
│  Persistencia: PVC + restic snapshots  │
│  Networking: Ingress + subdominio      │
└────────────────────────────────────────┘
```

---

## Stack Tecnológico

| Capa                   | Tecnología                          | Versión       | Propósito                                                    |
| ---------------------- | ----------------------------------- | ------------- | ------------------------------------------------------------ |
| **Lenguaje backend**   | Go                                  | 1.26.5        | Concurrencia nativa, excelente para operadores K8s           |
| **Framework API**      | Chi / Fiber                         | latest        | Ligero, alto throughput                                      |
| **K8s operator**       | controller-runtime + kubebuilder    | latest        | CRD + reconciler pattern                                     |
| **Frontend framework** | Astro                               | 7.0.6         | Islands architecture, renderizado estático + islas React     |
| **UI components**      | React                               | 19.2.7        | Componentes interactivos del dashboard                       |
| **Build tool**         | Vite 8 + Rolldown                   | 8.1.4 / 1.1.4 | Rust-native, 10-30x más rápido que Rollup                    |
| **Estilos**            | Tailwind CSS                        | 4             | Utility-first, compilación en Rust                           |
| **Formatter**          | Prettier                            | 3.9.5         | Con plugins para Astro y sort de imports                     |
| **Sort imports**       | @ianvs/prettier-plugin-sort-imports | 4.7.1         | Imports ordenados por capa (builtin → third-party → interno) |
| **Linter JS/TS**       | Oxlint                              | 1.73          | 50-100x más rápido que ESLint                                |
| **Linter Go**          | golangci-lint                       | 2.12          | ~40 linters integrados                                       |
| **Package manager**    | pnpm                                | 11.11         | Disk space eficiente, strict                                 |
| **Task runner**        | Task                                | 3.52          | Makefile moderno, YAML-based                                 |
| **Version manager**    | Mise                                | 2026.7        | Pinning Go/Node/Rust por proyecto                            |
| **Git hooks**          | Husky                               | 9.1.7         | commit-msg + pre-commit hooks                                |
| **Commit lint**        | commitlint                          | 21.2          | Conventional commits validation                              |
| **Lint staged**        | lint-staged                         | 17.0          | Formateo + lint en staged files                              |
| **CI/CD**              | GitHub Actions                      | —             | Lint + test + build + push imágenes                          |
| **Auth**               | Dex + OIDC                          | latest        | SSO con GitHub/GitLab/Google                                 |
| **Observabilidad**     | OpenTelemetry + Prometheus + Loki   | latest        | Trazas, métricas, logs                                       |
| **Despliegue**         | Helm + ArgoCD                       | latest        | GitOps, rollback automático                                  |

---

## Roadmap

```
FASE 0 ✅  Setup
           Monorepo, toolchain, CI/CD, estructura base
           Frontend Astro 7 + Prettier + Oxlint
           Husky + commitlint + lint-staged
           ───────────────────────────────────────────────

FASE 1 🔄  Core (en progreso)
           API server (CRUD workspaces/users/teams)
           K8s Operator (CRD + reconciler del Workspace)
           Auth (Dex + OIDC + JWT)
           ───────────────────────────────────────────────

FASE 2 ⏳  Workspace Engine
           code-server sidecar + init container
           Storage (PVC + Restic snapshots a S3)
           Networking (Ingress + TLS + port-forward relay)
           ───────────────────────────────────────────────

FASE 3 ⏳  Frontend Dashboard
           Login OIDC + lista de workspaces + creación
           Panel de admin (quotas, uso, audit)
           ───────────────────────────────────────────────

FASE 4 ⏳  Infraestructura
           Helm chart completo (api + operator + dex + redis + pg)
           ArgoCD ApplicationSet por entorno
           Dashboards de observabilidad
           ───────────────────────────────────────────────

FASE 5 ⏳  Features avanzadas
           Snapshots programados + restore
           Prebuilds de imágenes DevContainer en CI
           Port forwarding automático (subdominios)
           Auto-stop por idle + quotas por team
           ───────────────────────────────────────────────

FASE 6 ⏳  Hardening producción
           Multi-cluster distribution
           RBAC granular (admin / team-lead / dev)
           Audit log + billing metrics
           Security scanning (Trivy)
           ───────────────────────────────────────────────

           ⏱ Estimado: ~5 meses
```

---

## Estructura del Proyecto

```
nimbuscore/
│
├── apps/                          # Aplicaciones desplegables
│   ├── api/                       # API server REST
│   │   └── cmd/main.go
│   ├── operator/                  # K8s controller
│   │   └── cmd/main.go
│   ├── cli/                       # CLI nimbusctl
│   │   └── cmd/main.go
│   └── workspace-manager/         # Gestor de ciclo de vida
│       └── cmd/main.go
│
├── pkg/                           # Librerías compartidas
│   ├── api/                       # Tipos y clientes API
│   ├── operator/                  # Tipos del operador K8s
│   ├── workspace/                 # Lógica de dominio
│   └── auth/                      # Primitivas de autenticación
│
├── web/
│   └── dashboard/                 # Frontend Astro 7 + React
│       ├── src/
│       │   ├── layouts/           # Layouts Astro
│       │   ├── components/        # Componentes React
│       │   ├── pages/             # Páginas Astro
│       │   ├── styles/            # CSS global
│       │   └── env.d.ts
│       ├── astro.config.mjs
│       └── package.json
│
├── deploy/                        # Despliegue
│   ├── Dockerfile.api
│   ├── Dockerfile.operator
│   └── helm/                      # Charts Helm
│
├── .github/workflows/
│   └── ci.yml                     # CI pipeline
│
├── .husky/                        # Git hooks
│   ├── commit-msg                 # commitlint
│   └── pre-commit                 # lint-staged
│
├── .devcontainer/
│   └── devcontainer.json          # ¡Dogfooding!
│
├── .mise.toml                     # Versiones pinneadas
├── .prettierrc                    # Formatter config
├── .oxlintrc.json                 # Linter config
├── commitlint.config.mjs          # Conventional commits config
├── .lintstagedrc.json             # Lint-staged config
├── go.work                        # Go workspace
├── Taskfile.yml                   # Task runner
├── .golangci.yml                  # Linter Go
│
└── README.md
```

---

## Convenciones de Commits

Usamos **Conventional Commits** para mantener un historial limpio y generar
changelogs automáticos.

### Formato

```
<tipo>(<scope opcional>): <descripción>

[cuerpo opcional]

[pie opcional]
```

### Tipos

| Tipo       | Uso                              |
| ---------- | -------------------------------- |
| `feat`     | Nueva funcionalidad              |
| `fix`      | Corrección de bug                |
| `chore`    | Mantenimiento, tooling, configs  |
| `docs`     | Documentación                    |
| `refactor` | Refactor sin cambios funcionales |
| `test`     | Tests                            |
| `style`    | Formato, estilos (no lógica)     |
| `perf`     | Mejora de rendimiento            |

### Ejemplos

```
feat(api): add workspace CRUD endpoints
fix(operator): handle namespace deletion race condition
chore: update golangci-lint to v2.12.2
docs: add API reference to README
refactor(pkg/auth): extract token validation to middleware
```

### Hooks automáticos

- **pre-commit**: lint-staged ejecuta Prettier + Oxlint solo en archivos
  modificados
- **commit-msg**: commitlint valida que el mensaje siga el formato conventional

---

## Empezando

### Requisitos

- [Mise](https://mise.jdx.dev) — instala y gestiona todas las herramientas
  automáticamente
- Docker (para builds de imágenes)

### Setup

```bash
# 1. Clonar el repo
git clone https://github.com/nimbuscore/nimbuscore
cd nimbuscore

# 2. Activar Mise (una sola vez en tu shell)
echo 'eval "$(/home/jeancdevx/.local/bin/mise activate zsh)"' >> ~/.zshrc
exec zsh

# 3. Instalar toolchain (Go, Node, pnpm, Task, golangci-lint)
mise install

# 4. Instalar dependencias del proyecto
pnpm install

# 5. Probar que todo funciona
task dev

# 6. (Opcional) Frontend dev server
task web:dev
# Abrir http://localhost:5173
```

---

## Comandos

### Generales

```bash
task                  # Listar todas las tareas disponibles
task dev              # Correr todos los checks locales
task ci               # Pipeline CI completo (lint + test + build)
task clean            # Limpiar artefactos de build
task format           # Formatear todo el código con Prettier
task format:check     # Verificar formato
task lint             # Lint Go + oxlint
task oxlint           # Oxlint sobre JS/TS
```

### Go

```bash
task go:lint          # golangci-lint en todos los módulos Go
task go:test          # Tests unitarios con race detector
task go:build         # Compilar todas las aplicaciones
task go:vet           # go vet estático
task go:tidy          # go mod tidy en todos los módulos
```

```bash
# Manual por módulo
go build ./apps/api/cmd
go test -race -count=1 ./pkg/api/...
```

### Frontend

```bash
task web:install      # pnpm install
task web:dev          # Dev server en :5173 con HMR
task web:build        # Build producción con Astro + Rolldown
task web:preview      # Preview del build de producción
```

```bash
pnpm --filter dashboard dev        # Dev desde root
pnpm --filter dashboard build      # Build desde root
```

### Git Hooks

```bash
task hooks:init       # Inicializar husky (si no se ejecutó en install)
```

```bash
git commit -m "feat(api): add workspace CRUD"   # commitlint valida
```

### Docker

```bash
task docker:build:api          # Imagen de la API
task docker:build:operator     # Imagen del operador
```

```bash
DOCKER_BUILDKIT=1 docker build -f deploy/Dockerfile.api -t nimbuscore-api:dev .
```

---

## Licencia

MIT. Ver [LICENSE](LICENSE).

---

<p align="center">
  <sub>Hecho con ❤️ por el equipo de NimbusCore</sub>
</p>
