param resourceGroupName string
param monthlyBudgetUSD int = 0
param budgetAlertEmail string = ''

param startDate string = '${utcNow('yyyy-MM')}-01'

resource budget 'Microsoft.Consumption/budgets@2023-11-01' = {
  name: '${resourceGroupName}-budget'
  properties: {
    category: 'Cost'
    amount: monthlyBudgetUSD
    timeGrain: 'Monthly'
    timePeriod: {
      startDate: startDate
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
