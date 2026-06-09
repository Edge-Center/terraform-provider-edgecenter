//go:build cdn

package edgecenter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccCDNResource(t *testing.T) {
	t.Parallel()
	resourceName := "edgecenter_cdn_resource.acctest"

	type Params struct {
		Proto string
	}

	cname := fmt.Sprintf("cdn.terraform-%d.acctest", time.Now().Nanosecond())
	secondaryHostname := "secondary-" + cname

	create := Params{"HTTP"}
	update := Params{"MATCH"}

	groupName := fmt.Sprintf("terraform_acctest_group-%d", time.Now().Nanosecond())

	template := func(params *Params) string {
		return fmt.Sprintf(`
resource "edgecenter_cdn_origingroup" "acctest" {
  name                 = "%s"
  use_next             = true
  consistent_balancing = true

  origin {
    source  = "google.com"
    enabled = true
  }
}

resource "edgecenter_cdn_resource" "acctest" {
  cname               = "%s"
  origin_group        = tonumber(edgecenter_cdn_origingroup.acctest.id)
  origin_protocol     = "%s"
  secondary_hostnames = ["%s"]
}
		`, groupName, cname, params.Proto, secondaryHostname)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_CDN_URL_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: template(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "cname", cname),
					resource.TestCheckResourceAttr(resourceName, "origin_protocol", create.Proto),
				),
			},
			{
				Config: template(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "cname", cname),
					resource.TestCheckResourceAttr(resourceName, "origin_protocol", update.Proto),
				),
			},
		},
	})
}
