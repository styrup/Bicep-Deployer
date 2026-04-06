# Copilot Instructions — Bicep Deployer

## Build & Run

```bash
# Run the server (loads .env automatically)
make run                        # or: go run ./cmd/server/main.go

# Build binary
make build                      # produces ./bicep-deployer

# Tidy dependencies
make tidy                       # go mod tidy

# Docker
docker build -t bicep-deployer .
```

```bash
# Run all tests
go test ./...

# Run a single test by name
go test ./internal/handler/ -run TestParseModuleRefs
```

## Architecture

Bicep Deployer is a Go web app that lets users deploy Azure Bicep templates from a browser using their own Azure AD identity. The key architectural insight is **pass-through authentication**: the backend never holds ARM credentials — the frontend acquires tokens via MSAL.js and passes them to the backend, which proxies requests to the Azure Resource Manager API.

### Request flow

1. **Server startup** (`cmd/server/main.go`): loads config from env vars (`.env` in dev), creates a blob storage client wrapped in `CachedStore` (2 min TTL), sets up the middleware chain (request logging → security headers → rate limiting), configures `slog` for structured JSON logging, and starts the server with graceful shutdown on SIGINT/SIGTERM.
2. **Frontend auth** (`web/js/auth.js`): MSAL.js v3 authenticates against Azure AD. The server injects `TenantID`, `ClientID`, `AppTitle`, and `AppIcon` into `index.html` via Go templates at serve time.
3. **Template discovery** (`internal/handler/templates.go` → `internal/storage/blob.go`): lists `.bicep` files from Azure Blob Storage, filters to only `metadata published = 'true'`, extracts `metadata name` for display, and groups by directory prefix. These endpoints are **public** (no token required).
4. **Parameter parsing** (`internal/bicep/parser.go`): line-by-line parser extracts `param` declarations, decorators (`@description`, `@allowed`), metadata, and `targetScope`. Bicep expressions (e.g. `resourceGroup().location`) are detected and stored as hints, not literal defaults.
5. **Deployment** (`internal/handler/deploy.go`): validates input (including path traversal checks) → downloads template from blob → recursively downloads referenced local modules → writes all files to a temp directory preserving relative paths → compiles via Bicep CLI → for subscription-scope, extracts `location` from params for the ARM payload → sends PUT to ARM API with the user's Bearer token → returns the deployment URL. The frontend then polls `/api/deploy/status` every 3 seconds.
6. **ARM proxy** (`internal/handler/azure.go`): `/api/subscriptions` and `/api/resource-groups` forward the user's Bearer token to `https://management.azure.com`.

### Embedded frontend

Static assets in `web/` are embedded into the binary via `//go:embed web` in `assets.go` (root-level package `bicepdeployer`). The server serves `index.html` through Go's `html/template` for config injection; all other files are served as static assets.

### Middleware chain (`internal/middleware/`)

Applied in order: `RequestLogger` → `SecurityHeaders` → `RateLimiter`. Rate limiter is per-IP with stale entry cleanup.

### Caching (`internal/handler/cache.go`)

`CachedStore` wraps `TemplateStore` with in-memory TTL cache for both the template list and individual template content, avoiding repeated blob downloads.

### Logging (`internal/logging/`)

Uses `log/slog` with a `MultiHandler` that fans out to multiple backends. Default: JSON to stdout. Optional: file output via `LOG_FILE`. Extensible by adding `slog.Handler` implementations.

## API Endpoints

| Endpoint | Auth | Purpose |
|---|---|---|
| `GET /healthz` | No | Health check (liveness/readiness probe) |
| `GET /api/templates` | No | List published templates grouped by directory |
| `GET /api/templates/{name}` | No | Get parsed parameters for a template |
| `GET /api/subscriptions` | Bearer | Proxy to ARM subscriptions list |
| `GET /api/resource-groups?subscriptionId=...` | Bearer | Proxy to ARM resource groups |
| `POST /api/deploy` | Bearer | Compile & deploy a template |
| `GET /api/deploy/status?url=...` | Bearer | Poll ARM deployment status (restricted to ARM deployment URLs) |

## Key Conventions

- **Config**: all configuration via environment variables (see `.env.example`). `godotenv` loads `.env` in dev. Two storage auth modes: connection string or Managed Identity.
- **Logging**: use `slog.Info()`, `slog.Error()`, etc. — never `log.Printf`. Include structured context (e.g. `"template", name, "error", err`).
- **Error responses**: JSON `{ "error": "..." }` with appropriate HTTP status codes, via `writeError()` in `internal/handler/helpers.go`.
- **JSON helpers**: use `writeJSON()` and `writeError()` from `helpers.go` for all HTTP responses.
- **Template visibility**: only templates with `metadata published = 'true'` appear in listings. `metadata name` provides the display name.
- **Deployment mode**: always ARM Incremental. Empty parameters are omitted so ARM uses template defaults. Parameter values are wrapped as `{ "value": <raw> }`. Subscription-scope deployments include top-level `location` extracted from params.
- **Module resolution**: `compileBicep` creates a temp directory, writes the main template preserving its blob path, then recursively downloads referenced local modules (parsed via regex, skipping `br:`/`ts:` registry refs). Cycle detection via visited set.
- **Security**: template names are validated against path traversal. Deploy status URL is validated via regex to only allow ARM deployment resources. Rate limiting and security headers are applied via middleware.
- **Bicep compilation**: uses the Bicep CLI as an external process — it must be installed on the host (or container). Temp directories are cleaned up with `defer os.RemoveAll`.
- **Frontend state**: vanilla JS (no framework), managed through DOM manipulation. `web/js/auth.js` handles MSAL, `web/js/app.js` handles all UI logic.
- **CSS**: dark theme with a defined color palette in `web/css/styles.css`. Kebab-case class names.
- **Language**: UI text and README are in Danish. Code (variables, functions, comments) is in English.
