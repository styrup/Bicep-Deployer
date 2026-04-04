// Package bicepdeployer provides embedded web assets for the Bicep Deployer server.
package bicepdeployer

import "embed"

//go:embed web
var WebFS embed.FS
