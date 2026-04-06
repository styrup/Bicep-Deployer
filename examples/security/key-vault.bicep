metadata description = 'Creates an Azure Key Vault with configurable access policies and secrets'
metadata name = 'Key Vault'
metadata author = 'Security Team'
metadata version = '1.1'
metadata category = 'Security'
metadata published = 'true'

@description('Name of the Key Vault (must be globally unique)')
param keyVaultName string

@description('Azure region')
param location string = resourceGroup().location

@description('Azure AD tenant ID for access policies')
param tenantId string = subscription().tenantId

@description('Object ID of the user/group/SP to grant access')
param accessPolicyObjectId string

@description('SKU for the Key Vault')
@allowed([
  'standard'
  'premium'
])
param skuName string = 'standard'

@description('Enable soft delete')
param enableSoftDelete bool = true

@description('Soft delete retention in days')
param softDeleteRetentionDays int = 90

@description('Enable purge protection (cannot be disabled once enabled)')
param enablePurgeProtection bool = true

@description('Initial secrets to create (name → value pairs)')
@secure()
param secrets object = {}

@description('Tags')
param tags object = {
  environment: 'dev'
  managedBy: 'bicep-deployer'
}

resource keyVault 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: keyVaultName
  location: location
  tags: tags
  properties: {
    sku: {
      family: 'A'
      name: skuName
    }
    tenantId: tenantId
    enableSoftDelete: enableSoftDelete
    softDeleteRetentionInDays: softDeleteRetentionDays
    enablePurgeProtection: enablePurgeProtection ? true : null
    enableRbacAuthorization: false
    accessPolicies: [
      {
        tenantId: tenantId
        objectId: accessPolicyObjectId
        permissions: {
          keys: [ 'get', 'list', 'create', 'delete' ]
          secrets: [ 'get', 'list', 'set', 'delete' ]
          certificates: [ 'get', 'list', 'create', 'delete' ]
        }
      }
    ]
    networkAcls: {
      defaultAction: 'Allow'
      bypass: 'AzureServices'
    }
  }
}

resource kvSecrets 'Microsoft.KeyVault/vaults/secrets@2023-07-01' = [for item in items(secrets): {
  parent: keyVault
  name: item.key
  properties: {
    value: item.value
  }
}]

output keyVaultId string = keyVault.id
output keyVaultUri string = keyVault.properties.vaultUri
