metadata description = 'Creates a Resource Group with enforced tags and optional budget alert'
metadata author = 'Platform Team'
metadata version = '1.0'
metadata category = 'Governance'

targetScope = 'subscription'

@description('Name of the resource group to create')
param resourceGroupName string

@description('Azure region for the resource group')
@allowed([
  'westeurope'
  'northeurope'
  'swedencentral'
  'norwayeast'
])
param location string = 'westeurope'

@description('Environment tag')
@allowed([
  'dev'
  'staging'
  'production'
])
param environment string = 'dev'

@description('Team or cost center that owns this resource group')
param owner string

@description('Monthly budget in USD (0 = no budget alert)')
param monthlyBudgetUSD int = 0

@description('Email to receive budget alerts')
param budgetAlertEmail string = ''

@description('Additional tags to apply')
param tags object = {}

@description('Creation date tag (auto-generated)')
param createdDate string = utcNow('yyyy-MM-dd')

var defaultTags = {
  environment: environment
  owner: owner
  managedBy: 'bicep-deployer'
  createdDate: createdDate
}

resource rg 'Microsoft.Resources/resourceGroups@2023-07-01' = {
  name: resourceGroupName
  location: location
  tags: union(defaultTags, tags)
}

module budget './modules/budget.bicep' = if (monthlyBudgetUSD > 0) {
  scope: rg
  name: 'budgetModule'
  params: {
    resourceGroupName: rg.name
    monthlyBudgetUSD: monthlyBudgetUSD
    budgetAlertEmail: budgetAlertEmail
  }
}

output resourceGroupId string = rg.id
output resourceGroupName string = rg.name
