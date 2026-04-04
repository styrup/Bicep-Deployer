@description('Azure region for all resources')
param location string = resourceGroup().location

@description('Globally unique name for the Container App Environment')
param environmentName string = 'bicep-deployer-env'

@description('Name of the Container App')
param appName string = 'bicep-deployer'

@description('Container image (e.g. myregistry.azurecr.io/bicep-deployer:latest)')
param containerImage string

@description('Azure AD Tenant ID for MSAL')
param azureTenantId string

@description('Azure AD App Registration Client ID for MSAL')
param azureClientId string

@description('Storage Account name holding .bicep templates')
param storageAccountName string

@description('Blob container name with .bicep files')
param storageContainerName string = 'bicep'

@description('Container Registry server (e.g. myregistry.azurecr.io)')
param registryServer string

@description('Container Registry username')
@secure()
param registryUsername string

@description('Container Registry password')
@secure()
param registryPassword string

// ── Log Analytics (required by Container App Environment) ────────────────
resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2023-09-01' = {
  name: '${appName}-logs'
  location: location
  properties: {
    sku: { name: 'PerGB2018' }
    retentionInDays: 30
  }
}

// ── Container App Environment ────────────────────────────────────────────
resource env 'Microsoft.App/managedEnvironments@2024-03-01' = {
  name: environmentName
  location: location
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logAnalytics.properties.customerId
        sharedKey: logAnalytics.listKeys().primarySharedKey
      }
    }
  }
}

// ── Reference existing Storage Account ───────────────────────────────────
resource storage 'Microsoft.Storage/storageAccounts@2023-05-01' existing = {
  name: storageAccountName
}

// ── Container App ────────────────────────────────────────────────────────
resource app 'Microsoft.App/containerApps@2024-03-01' = {
  name: appName
  location: location
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    managedEnvironmentId: env.id
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        transport: 'http'
        allowInsecure: false
      }
      registries: [
        {
          server: registryServer
          username: registryUsername
          passwordSecretRef: 'registry-password'
        }
      ]
      secrets: [
        {
          name: 'registry-password'
          value: registryPassword
        }
      ]
    }
    template: {
      containers: [
        {
          name: appName
          image: containerImage
          resources: {
            cpu: json('0.25')
            memory: '0.5Gi'
          }
          env: [
            { name: 'PORT',                     value: '8080' }
            { name: 'AZURE_TENANT_ID',           value: azureTenantId }
            { name: 'AZURE_CLIENT_ID',           value: azureClientId }
            { name: 'STORAGE_ACCOUNT_NAME',      value: storageAccountName }
            { name: 'STORAGE_CONTAINER_NAME',    value: storageContainerName }
          ]
        }
      ]
      scale: {
        minReplicas: 0
        maxReplicas: 3
        rules: [
          {
            name: 'http-rule'
            http: {
              metadata: {
                concurrentRequests: '50'
              }
            }
          }
        ]
      }
    }
  }
}

// ── Role assignment: Storage Blob Data Reader → Container App ────────────
resource roleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(storage.id, app.id, '2a2b9908-6ea1-4ae2-8e65-a410df84e7d1')
  scope: storage
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '2a2b9908-6ea1-4ae2-8e65-a410df84e7d1')
    principalId: app.identity.principalId
    principalType: 'ServicePrincipal'
  }
}

// ── Outputs ──────────────────────────────────────────────────────────────
output appUrl string = 'https://${app.properties.configuration.ingress.fqdn}'
output principalId string = app.identity.principalId
