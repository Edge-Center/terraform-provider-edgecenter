package main

import (
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

var version = "dev"

func main() {
	var debug bool
	var address string

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.StringVar(&address, "address", "provider", "this value is used in the TF_REATTACH_PROVIDERS environment variable during debugging")
	flag.Parse()

	opts := &plugin.ServeOpts{
		Debug:        debug,
		ProviderAddr: address,
		ProviderFunc: func() *schema.Provider {
			return provider.ProviderWithVersion(version)
		},
	}

	plugin.Serve(opts)
}
