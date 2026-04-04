metadata description = 'Creates an Azure Container Apps environment with a container app'
metadata author = 'Platform Team'
metadata version = '1.0'
metadata category = 'Compute'

@description('Name of the container app')
param appName string

@description('Azure region')
param location string = resourceGroup().location

@description('Container image to deploy')
param containerImage string = 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'

@description('CPU cores allocated to the container')
@allowed([
  '0.25'
  '0.5'
  '1.0'
  '2.0'
])
param cpuCores string = '0.25'

@description('Memory allocated to the container')
@allowed([
  '0.5Gi'
  '1.0Gi'
  '2.0Gi'
  '4.0Gi'
])
param memory string = '0.5Gi'

@description('Minimum number of replicas (0 = scale to zero)')
param minReplicas int = 0

@description('Maximum number of replicas')
param maxReplicas int = 3

@description('Container port to expose')
param targetPort int = 80

@description('Environment variables for the container')
param envVars array = []

@description('Tags')
param tags object = {
  environment: 'dev'
  managedBy: 'bicep-deployer'
}

resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2023-09-01' = {
  name: '${appName}-logs'
  location: location
  tags: tags
  properties: {
    sku: { name: 'PerGB2018' }
    retentionInDays: 30
  }
}

resource containerAppEnv 'Microsoft.App/managedEnvironments@2024-03-01' = {
  name: '${appName}-env'
  location: location
  tags: tags
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

resource containerApp 'Microsoft.App/containerApps@2024-03-01' = {
  name: appName
  location: location
  tags: tags
  properties: {
    managedEnvironmentId: containerAppEnv.id
    configuration: {
      ingress: {
        external: true
        targetPort: targetPort
        transport: 'http'
        allowInsecure: false
      }
    }
    template: {
      containers: [
        {
          name: appName
          image: containerImage
          resources: {
            cpu: json(cpuCores)
            memory: memory
          }
          env: envVars
        }
      ]
      scale: {
        minReplicas: minReplicas
        maxReplicas: maxReplicas
        rules: [
          {
            name: 'http-scaling'
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

output appUrl string = 'https://${containerApp.properties.configuration.ingress.fqdn}'
output appId string = containerApp.id
