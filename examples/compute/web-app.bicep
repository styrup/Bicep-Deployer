metadata description = 'Creates a Linux web app on Azure App Service with Application Insights'
metadata name = 'Web App (Linux)'
metadata author = 'Platform Team'
metadata version = '2.0'
metadata category = 'Compute'
metadata published = 'true'

@description('Name of the web app (must be globally unique)')
param webAppName string

@description('Azure region')
param location string = resourceGroup().location

@description('App Service Plan SKU')
@allowed([
  'F1'
  'B1'
  'B2'
  'S1'
  'S2'
  'P1v3'
  'P2v3'
])
param skuName string = 'B1'

@description('Runtime stack')
@allowed([
  'NODE|20-lts'
  'PYTHON|3.12'
  'DOTNETCORE|8.0'
  'JAVA|17-java17'
  'GO|1.22'
])
param linuxFxVersion string = 'NODE|20-lts'

@description('Enable Application Insights monitoring')
param enableAppInsights bool = true

@description('App settings as key-value pairs')
param appSettings object = {}

@description('Tags')
param tags object = {
  environment: 'dev'
  managedBy: 'bicep-deployer'
}

resource appServicePlan 'Microsoft.Web/serverfarms@2023-12-01' = {
  name: '${webAppName}-plan'
  location: location
  tags: tags
  kind: 'linux'
  sku: {
    name: skuName
  }
  properties: {
    reserved: true
  }
}

resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2023-09-01' = if (enableAppInsights) {
  name: '${webAppName}-logs'
  location: location
  tags: tags
  properties: {
    sku: { name: 'PerGB2018' }
    retentionInDays: 30
  }
}

resource appInsights 'Microsoft.Insights/components@2020-02-02' = if (enableAppInsights) {
  name: '${webAppName}-ai'
  location: location
  tags: tags
  kind: 'web'
  properties: {
    Application_Type: 'web'
    WorkspaceResourceId: enableAppInsights ? logAnalytics.id : ''
  }
}

var baseAppSettings = [
  { name: 'WEBSITE_NODE_DEFAULT_VERSION', value: '~20' }
]

var aiSettings = enableAppInsights ? [
  { name: 'APPINSIGHTS_INSTRUMENTATIONKEY', value: appInsights.properties.InstrumentationKey }
  { name: 'APPLICATIONINSIGHTS_CONNECTION_STRING', value: appInsights.properties.ConnectionString }
] : []

var customSettings = [for item in items(appSettings): {
  name: item.key
  value: item.value
}]

resource webApp 'Microsoft.Web/sites@2023-12-01' = {
  name: webAppName
  location: location
  tags: tags
  properties: {
    serverFarmId: appServicePlan.id
    httpsOnly: true
    siteConfig: {
      linuxFxVersion: linuxFxVersion
      alwaysOn: skuName != 'F1'
      minTlsVersion: '1.2'
      ftpsState: 'Disabled'
      appSettings: concat(baseAppSettings, aiSettings, customSettings)
    }
  }
}

output webAppUrl string = 'https://${webApp.properties.defaultHostName}'
output webAppId string = webApp.id
