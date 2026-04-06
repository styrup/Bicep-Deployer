# Bicep Deployer

A web-based system for deploying Azure Bicep templates directly from the browser using your Azure identity.

## Features

- 🔐 **Azure AD login** via MSAL.js — no server-side ARM credentials required
- 📦 **Central template store** in Azure Blob Storage
- ⚙️ **Auto-generated forms** from `param` declarations in `.bicep` files
- 🚀 **Deployment at Resource Group or Subscription scope**
- 🔗 **Module support** — templates referencing local modules are downloaded automatically
- 🏷️ **Template management** — show/hide templates via `metadata published` and set display names via `metadata name`
- 🎨 **Configurable branding** — title and icon customizable via environment variables
- 🔒 **Security hardened** — rate limiting, security headers, SSRF protection, path traversal validation
- 📊 **Structured logging** — JSON logs via `slog` with multi-handler support
- 🌑 **Dark, minimalist Nordic design**

## Prerequisites

1. **Go 1.21+**
2. **Bicep CLI** — install with `winget install Microsoft.Bicep` (Windows) or `brew install bicep` (macOS)
3. An **Azure App Registration** (see setup below)
4. An **Azure Blob Storage container** with `.bicep` files

## Azure App Registration

1. Go to [Entra ID → App Registrations](https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps)
2. Create a new registration
3. Add Redirect URI: `http://localhost:8080` (type: **Single-page application**)
4. Under **API permissions** → Add permission → **Azure Service Management** → `user_impersonation`
5. Under **Authentication** → enable **Allow public client flows** = Yes
6. Copy the **Tenant ID** and **Application (client) ID**

## Configuration

```bash
cp .env.example .env
# Edit .env with your values
```

### Environment variables

| Variable | Description | Required |
|---|---|---|
| `AZURE_TENANT_ID` | Azure AD Tenant ID | ✅ |
| `AZURE_CLIENT_ID` | App Registration Client ID | ✅ |
| `AZURE_STORAGE_CONNECTION_STRING` | Storage connection string | One of two |
| `STORAGE_ACCOUNT_NAME` | Storage account (uses Managed Identity) | One of two |
| `STORAGE_CONTAINER_NAME` | Blob container with `.bicep` files (default: `bicep`) | ✅ |
| `PORT` | HTTP port (default: `8080`) | ❌ |
| `APP_TITLE` | App title (default: `Bicep Deployer`) | ❌ |
| `APP_ICON` | Emoji (`🔧`) or image URL (`https://...`) | ❌ |
| `LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` (default: `info`) | ❌ |
| `LOG_FILE` | Optional file path for log output (in addition to stdout) | ❌ |

## Running

```bash
make tidy     # go mod tidy
make run      # go run ./cmd/server/main.go
# Open http://localhost:8080
```

```bash
# Build
make build    # produces ./bicep-deployer

# Test
go test ./...
```

## Bicep template format

Templates must be `.bicep` files in your Blob Storage container.

### Visibility and naming

Only templates with `metadata published = 'true'` are shown in the UI. Use `metadata name` to control the display name:

```bicep
metadata name = 'Storage Account'
metadata description = 'Creates a Storage Account with configurable SKU'
metadata author = 'Platform Team'
metadata version = '1.0'
metadata category = 'Storage'
metadata published = 'true'
```

Templates without `metadata published = 'true'` (e.g. modules) are hidden automatically.

### Parameters

Parameters are automatically parsed from `param` declarations:

```bicep
@description('Azure region to deploy resources into')
param location string = 'westeurope'

@allowed(['Standard_LRS', 'Premium_LRS'])
param sku string = 'Standard_LRS'

param storageAccountName string   // required — shown with *
param instanceCount int = 2
param enableDiagnostics bool = false
```

### Modules

Templates can reference local modules with relative paths. Modules are automatically downloaded from Blob Storage during compilation:

```bicep
module budget './modules/budget.bicep' = if (monthlyBudgetUSD > 0) {
  scope: rg
  name: 'budgetModule'
  params: { ... }
}
```

## Project structure

```
bicep-deployer/
├── cmd/server/main.go              # HTTP server, middleware chain, graceful shutdown
├── internal/
│   ├── config/config.go            # Configuration from env vars
│   ├── bicep/parser.go             # Parses Bicep param declarations and metadata
│   ├── storage/blob.go             # Azure Blob Storage client
│   ├── logging/logging.go          # slog setup, MultiHandler, log levels
│   ├── middleware/middleware.go     # Security headers, rate limiting, request logging
│   └── handler/
│       ├── templates.go            # GET /api/templates (filtering + display names)
│       ├── azure.go                # GET /api/subscriptions, /api/resource-groups
│       ├── deploy.go               # POST /api/deploy (module download, compilation)
│       ├── cache.go                # CachedStore — TTL cache for templates
│       └── helpers.go              # JSON/auth utilities
├── web/                            # Embedded frontend (HTML/CSS/JS)
├── examples/                       # Example Bicep templates
├── deploy/                         # Azure Container Apps deployment
│   ├── main.bicep                  # Infrastructure-as-code
│   └── main.bicepparam             # Parameter file
├── Dockerfile
└── .dockerignore
```

## Deploy to Azure Container Apps

### 1. Create Azure Container Registry and build image

```bash
# Create resource group and ACR
az group create -n rg-bicep-deployer -l westeurope
az acr create -n mybicepregistry -g rg-bicep-deployer --sku Basic --admin-enabled true

# Build and push image
az acr build -r mybicepregistry -t bicep-deployer:latest .
```

### 2. Deploy with Bicep

```bash
# Edit deploy/main.bicepparam with your values, then:
az deployment group create \
  -g rg-bicep-deployer \
  -f deploy/main.bicep \
  -p deploy/main.bicepparam
```

### 3. Update App Registration

Add the new Container App URL as a Redirect URI (type: **Single-page application**) in your App Registration.

### What the Bicep template creates

- **Log Analytics Workspace** — for logs
- **Container App Environment** — hosting environment
- **Container App** — the app with Managed Identity, scale-to-zero, HTTPS
- **Role Assignment** — grants the app `Storage Blob Data Reader` on your Storage Account

## Security

- **Rate limiting** — 20 req/s per IP with burst 40
- **Security headers** — CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy
- **Request timeouts** — read 10s, write 60s, idle 120s
- **Path traversal protection** — template names validated against `..` and absolute paths
- **SSRF hardening** — deploy status proxy only accepts ARM deployment URLs
- **Graceful shutdown** — SIGINT/SIGTERM with 10s drain
- **Health check** — `GET /healthz` for liveness/readiness probes
