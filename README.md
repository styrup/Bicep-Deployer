# Bicep Deployer

Et webbaseret system til at deploye Azure Bicep templates direkte fra browseren via din Azure-identitet.

## Features

- 🔐 **Azure AD login** via MSAL.js — ingen server-side ARM-credentials nødvendige
- 📦 **Central template-lager** i Azure Blob Storage
- ⚙️ **Auto-genererede formularer** ud fra `param`-deklarationer i `.bicep` filer
- 🚀 **Deployment på Resource Group eller Subscription niveau**
- 🔗 **Modul-support** — templates der refererer lokale moduler downloades automatisk
- 🏷️ **Template-styring** — vis/skjul templates via `metadata published` og vis pæne navne via `metadata name`
- 🎨 **Konfigurerbar branding** — titel og ikon kan ændres via env vars
- 🔒 **Sikkerhedshærdet** — rate limiting, security headers, SSRF-beskyttelse, path traversal validering
- 📊 **Struktureret logging** — JSON-logs via `slog` med multi-handler support
- 🌑 **Mørkt, minimalistisk nordisk design**

## Forudsætninger

1. **Go 1.21+**
2. **Bicep CLI** — installeres med `winget install Microsoft.Bicep` (Windows) eller `brew install bicep` (macOS)
3. En **Azure App Registration** (se setup nedenfor)
4. En **Azure Blob Storage container** med `.bicep` filer

## Azure App Registration

1. Gå til [Entra ID → App Registrations](https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps)
2. Opret en ny registrering
3. Tilføj Redirect URI: `http://localhost:8080` (type: **Single-page application**)
4. Under **API permissions** → Add permission → **Azure Service Management** → `user_impersonation`
5. Under **Authentication** → enable **Allow public client flows** = Yes
6. Kopiér **Tenant ID** og **Application (client) ID**

## Konfiguration

```bash
cp .env.example .env
# Rediger .env med dine værdier
```

### Environment variables

| Variabel | Beskrivelse | Påkrævet |
|---|---|---|
| `AZURE_TENANT_ID` | Azure AD Tenant ID | ✅ |
| `AZURE_CLIENT_ID` | App Registration Client ID | ✅ |
| `AZURE_STORAGE_CONNECTION_STRING` | Storage connection string | En af to |
| `STORAGE_ACCOUNT_NAME` | Storage account (bruger Managed Identity) | En af to |
| `STORAGE_CONTAINER_NAME` | Blob container med `.bicep` filer (default: `bicep`) | ✅ |
| `PORT` | HTTP port (default: `8080`) | ❌ |
| `APP_TITLE` | Appens titel (default: `Bicep Deployer`) | ❌ |
| `APP_ICON` | Emoji (`🔧`) eller billed-URL (`https://...`) | ❌ |
| `LOG_LEVEL` | Log niveau: `debug`, `info`, `warn`, `error` (default: `info`) | ❌ |
| `LOG_FILE` | Valgfri fil-sti for log-output (ud over stdout) | ❌ |

## Kørsel

```bash
make tidy     # go mod tidy
make run      # go run ./cmd/server/main.go
# Åbn http://localhost:8080
```

```bash
# Build
make build    # producerer ./bicep-deployer

# Test
go test ./...
```

## Bicep template format

Templates skal ligge som `.bicep` filer i din Blob Storage container.

### Synlighed og navngivning

Kun templates med `metadata published = 'true'` vises i UI'et. Brug `metadata name` til at styre det viste navn:

```bicep
metadata name = 'Storage Account'
metadata description = 'Creates a Storage Account with configurable SKU'
metadata author = 'Platform Team'
metadata version = '1.0'
metadata category = 'Storage'
metadata published = 'true'
```

Templates uden `metadata published = 'true'` (f.eks. moduler) skjules automatisk.

### Parametre

Parametre parses automatisk fra `param`-deklarationer:

```bicep
@description('Azure region to deploy resources into')
param location string = 'westeurope'

@allowed(['Standard_LRS', 'Premium_LRS'])
param sku string = 'Standard_LRS'

param storageAccountName string   // required — vises med *
param instanceCount int = 2
param enableDiagnostics bool = false
```

### Moduler

Templates kan referere lokale moduler med relative stier. Modulerne downloades automatisk fra Blob Storage under kompilering:

```bicep
module budget './modules/budget.bicep' = if (monthlyBudgetUSD > 0) {
  scope: rg
  name: 'budgetModule'
  params: { ... }
}
```

## Projektstruktur

```
bicep-deployer/
├── cmd/server/main.go              # HTTP server, middleware chain, graceful shutdown
├── internal/
│   ├── config/config.go            # Konfiguration fra env vars
│   ├── bicep/parser.go             # Parser Bicep param-deklarationer og metadata
│   ├── storage/blob.go             # Azure Blob Storage klient
│   ├── logging/logging.go          # slog setup, MultiHandler, log levels
│   ├── middleware/middleware.go     # Security headers, rate limiting, request logging
│   └── handler/
│       ├── templates.go            # GET /api/templates (filtrering + display names)
│       ├── azure.go                # GET /api/subscriptions, /api/resource-groups
│       ├── deploy.go               # POST /api/deploy (modul-download, kompilering)
│       ├── cache.go                # CachedStore — TTL cache for templates
│       └── helpers.go              # JSON/auth utilities
├── web/                            # Embedded frontend (HTML/CSS/JS)
├── examples/                       # Eksempel Bicep templates
├── deploy/                         # Azure Container Apps deployment
│   ├── main.bicep                  # Infrastructure-as-code
│   └── main.bicepparam             # Parameter fil
├── Dockerfile
└── .dockerignore
```

## Deploy til Azure Container Apps

### 1. Opret Azure Container Registry og byg image

```bash
# Opret resource group og ACR
az group create -n rg-bicep-deployer -l westeurope
az acr create -n mybicepregistry -g rg-bicep-deployer --sku Basic --admin-enabled true

# Byg og push image
az acr build -r mybicepregistry -t bicep-deployer:latest .
```

### 2. Deploy med Bicep

```bash
# Rediger deploy/main.bicepparam med dine værdier, derefter:
az deployment group create \
  -g rg-bicep-deployer \
  -f deploy/main.bicep \
  -p deploy/main.bicepparam
```

### 3. Opdater App Registration

Tilføj den nye Container App URL som Redirect URI (type: **Single-page application**) i din App Registration.

### Hvad Bicep-templaten opretter

- **Log Analytics Workspace** — til logs
- **Container App Environment** — hosting-miljø
- **Container App** — selve appen med Managed Identity, scale-to-zero, HTTPS
- **Role Assignment** — giver appen `Storage Blob Data Reader` på din Storage Account

## Sikkerhed

- **Rate limiting** — 20 req/s per IP med burst 40
- **Security headers** — CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy
- **Request timeouts** — read 10s, write 60s, idle 120s
- **Path traversal beskyttelse** — template-navne valideres mod `..` og absolutte stier
- **SSRF-hærdning** — deploy status proxy accepterer kun ARM deployment-URLs
- **Graceful shutdown** — SIGINT/SIGTERM med 10s drain
- **Health check** — `GET /healthz` til liveness/readiness probes
