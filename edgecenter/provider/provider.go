package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func Provider() *schema.Provider {
	return edgecenter.Provider()
}
