package main

import (
	"github.com/kaytu-io/open-governance/pkg/steampipe-plugin-kaytu/kaytu"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{PluginFunc: kaytu.Plugin})
}
