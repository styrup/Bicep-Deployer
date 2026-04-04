using './main.bicep'

param containerImage = ''           // e.g. myregistry.azurecr.io/bicep-deployer:latest
param azureTenantId  = ''           // Azure AD Tenant ID
param azureClientId  = ''           // App Registration Client ID
param storageAccountName = ''       // Existing Storage Account
param registryServer   = ''        // e.g. myregistry.azurecr.io
param registryUsername = ''
param registryPassword = ''
