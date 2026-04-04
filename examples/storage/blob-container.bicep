metadata description = 'Creates a Blob container with optional lifecycle policy for auto-cleanup'
metadata author = 'Platform Team'
metadata version = '1.0'
metadata category = 'Storage'

@description('Name of the existing storage account')
param storageAccountName string

@description('Name of the blob container')
param containerName string = 'data'

@description('Public access level')
@allowed([
  'None'
  'Blob'
  'Container'
])
param publicAccess string = 'None'

@description('Enable lifecycle policy to auto-delete blobs after N days (0 = disabled)')
param autoDeleteAfterDays int = 0

resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' existing = {
  name: storageAccountName
}

resource blobService 'Microsoft.Storage/storageAccounts/blobServices@2023-05-01' existing = {
  parent: storageAccount
  name: 'default'
}

resource container 'Microsoft.Storage/storageAccounts/blobServices/containers@2023-05-01' = {
  parent: blobService
  name: containerName
  properties: {
    publicAccess: publicAccess
  }
}

resource lifecyclePolicy 'Microsoft.Storage/storageAccounts/managementPolicies@2023-05-01' = if (autoDeleteAfterDays > 0) {
  parent: storageAccount
  name: 'default'
  properties: {
    policy: {
      rules: [
        {
          name: 'auto-delete'
          enabled: true
          type: 'Lifecycle'
          definition: {
            actions: {
              baseBlob: {
                delete: {
                  daysAfterModificationGreaterThan: autoDeleteAfterDays
                }
              }
            }
            filters: {
              blobTypes: [ 'blockBlob' ]
              prefixMatch: [ '${containerName}/' ]
            }
          }
        }
      ]
    }
  }
}

output containerId string = container.id
