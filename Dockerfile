# ── Build stage ───────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS build
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bicep-deployer ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────
FROM mcr.microsoft.com/dotnet/runtime-deps:8.0-alpine

# Install Bicep CLI
RUN apk add --no-cache curl \
    && curl -Lo /usr/local/bin/bicep https://github.com/Azure/bicep/releases/latest/download/bicep-linux-musl-x64 \
    && chmod +x /usr/local/bin/bicep \
    && bicep --version

COPY --from=build /bicep-deployer /usr/local/bin/bicep-deployer

EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["bicep-deployer"]
