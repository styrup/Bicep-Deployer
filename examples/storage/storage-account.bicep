metadata description = 'Creates a Storage Account with configurable SKU and access tier'
metadata author = 'Platform Team'
metadata version = '1.0'
metadata category = 'Storage'

@description('Name of the storage account (must be globally unique)')
param storageAccountName string

@description('Azure region for the storage account')
param location string = resourceGroup().location

@description('SKU for the storage account')
@allowed([
  'Standard_LRS'
  'Standard_GRS'
  'Standard_RAGRS'
  'Standard_ZRS'
  'Premium_LRS'
])
param sku string = 'Standard_LRS'

@description('Access tier for the storage account')
@allowed([
  'Hot'
  'Cool'
])
param accessTier string = 'Hot'

@description('Enable blob versioning')
param enableVersioning bool = false

@description('Tags to apply to the resource')
param tags object = {
  environment: 'dev'
  managedBy: 'bicep-deployer'
}

resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' = {
  name: storageAccountName
  location: location
  sku: {
    name: sku
  }
  kind: 'StorageV2'
  tags: tags
  properties: {
    accessTier: accessTier
    supportsHttpsTrafficOnly: true
    minimumTlsVersion: 'TLS1_2'
    allowBlobPublicAccess: false
  }
}

resource blobService 'Microsoft.Storage/storageAccounts/blobServices@2023-05-01' = {
  parent: storageAccount
  name: 'default'
  properties: {
    isVersioningEnabled: enableVersioning
  }
}

output storageAccountId string = storageAccount.id
output primaryEndpoint string = storageAccount.properties.primaryEndpoints.blob
