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

var defaultTags = {
  environment: environment
  owner: owner
  managedBy: 'bicep-deployer'
  createdDate: utcNow('yyyy-MM-dd')
}

resource rg 'Microsoft.Resources/resourceGroups@2023-07-01' = {
  name: resourceGroupName
  location: location
  tags: union(defaultTags, tags)
}

resource budget 'Microsoft.Consumption/budgets@2023-11-01' = if (monthlyBudgetUSD > 0) {
  name: '${resourceGroupName}-budget'
  scope: rg
  properties: {
    category: 'Cost'
    amount: monthlyBudgetUSD
    timeGrain: 'Monthly'
    timePeriod: {
      startDate: '${utcNow('yyyy-MM')}-01'
    }
    notifications: {
      actual80pct: {
        enabled: budgetAlertEmail != ''
        operator: 'GreaterThanOrEqualTo'
        threshold: 80
        contactEmails: budgetAlertEmail != '' ? [budgetAlertEmail] : []
        thresholdType: 'Actual'
      }
      forecast100pct: {
        enabled: budgetAlertEmail != ''
        operator: 'GreaterThanOrEqualTo'
        threshold: 100
        contactEmails: budgetAlertEmail != '' ? [budgetAlertEmail] : []
        thresholdType: 'Forecasted'
      }
    }
  }
}

output resourceGroupId string = rg.id
output resourceGroupName string = rg.name
