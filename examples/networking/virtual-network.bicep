metadata description = 'Creates a Virtual Network with configurable subnets and NSG'
metadata author = 'Platform Team'
metadata version = '1.2'
metadata category = 'Networking'
metadata published = 'true'

@description('Name of the virtual network')
param vnetName string

@description('Azure region')
param location string = resourceGroup().location

@description('Address space for the VNet')
param addressPrefix string = '10.0.0.0/16'

@description('Subnet configuration')
param subnets array = [
  {
    name: 'default'
    addressPrefix: '10.0.1.0/24'
  }
  {
    name: 'backend'
    addressPrefix: '10.0.2.0/24'
  }
]

@description('Enable DDoS protection (costs extra)')
param enableDdosProtection bool = false

@description('Tags')
param tags object = {
  environment: 'dev'
  managedBy: 'bicep-deployer'
}

resource nsg 'Microsoft.Network/networkSecurityGroups@2023-11-01' = {
  name: '${vnetName}-nsg'
  location: location
  tags: tags
  properties: {
    securityRules: [
      {
        name: 'AllowHTTPS'
        properties: {
          priority: 100
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourcePortRange: '*'
          destinationPortRange: '443'
          sourceAddressPrefix: '*'
          destinationAddressPrefix: '*'
        }
      }
      {
        name: 'DenyAllInbound'
        properties: {
          priority: 4096
          direction: 'Inbound'
          access: 'Deny'
          protocol: '*'
          sourcePortRange: '*'
          destinationPortRange: '*'
          sourceAddressPrefix: '*'
          destinationAddressPrefix: '*'
        }
      }
    ]
  }
}

resource vnet 'Microsoft.Network/virtualNetworks@2023-11-01' = {
  name: vnetName
  location: location
  tags: tags
  properties: {
    addressSpace: {
      addressPrefixes: [ addressPrefix ]
    }
    enableDdosProtection: enableDdosProtection
    subnets: [for subnet in subnets: {
      name: subnet.name
      properties: {
        addressPrefix: subnet.addressPrefix
        networkSecurityGroup: {
          id: nsg.id
        }
      }
    }]
  }
}

output vnetId string = vnet.id
output subnetIds array = [for (subnet, i) in subnets: vnet.properties.subnets[i].id]
