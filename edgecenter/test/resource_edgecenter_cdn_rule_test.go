//go:build cdn

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccCDNRule(t *testing.T) {
	t.Parallel()
	resourceName := "edgecenter_cdn_rule.acctest"

	type Params struct {
		Name    string
		Pattern string
		RawPart string
	}

	create := Params{
		Name:    "All images",
		Pattern: "/folder/images/*.png",
	}
	update := Params{
		Name:    "All scripts",
		Pattern: "/folder/scripts/*.js",
		RawPart: `
  options {
    host_header {
      enabled = true
      value = "rule-host.com"
    }
  }
		`,
	}

	template := func(params *Params) string {
		return fmt.Sprintf(`
resource "edgecenter_cdn_rule" "acctest" {
  resource_id = %s
  name = "%s"
  rule = "%s"
  %s
}
		`, EC_CDN_RESOURCE_ID, params.Name, params.Pattern, params.RawPart)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_CDN_URL_VAR, EC_CDN_RESOURCE_ID_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: template(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", create.Name),
					resource.TestCheckResourceAttr(resourceName, "rule", create.Pattern),
				),
			},
			{
				Config: template(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", update.Name),
					resource.TestCheckResourceAttr(resourceName, "rule", update.Pattern),
					resource.TestCheckResourceAttr(resourceName, "options.0.host_header.0.value", "rule-host.com"),
				),
			},
		},
	})
}
