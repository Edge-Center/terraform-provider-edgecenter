//go:build !cloud
// +build !cloud

package edgecenter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccCDNResource(t *testing.T) {
	t.Parallel()
	fullName := "edgecenter_cdn_resource.acctest"

	type Params struct {
		Proto string
	}

	cname := fmt.Sprintf("cdn.terraform-%d.acctest", time.Now().Nanosecond())
	secondaryHostname := "secondary-" + cname

	create := Params{"HTTP"}
	update := Params{"MATCH"}

	template := func(params *Params) string {
		return fmt.Sprintf(`
resource "edgecenter_cdn_resource" "acctest" {
  cname = "%s"
  origin_group = %s
  origin_protocol = "%s"
  secondary_hostnames = ["%s"]
}
		`, cname, EC_CDN_ORIGINGROUP_ID, params.Proto, secondaryHostname)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_CDN_URL_VAR, EC_CDN_ORIGINGROUP_ID_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: template(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "cname", cname),
					resource.TestCheckResourceAttr(fullName, "origin_protocol", create.Proto),
				),
			},
			{
				Config: template(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "cname", cname),
					resource.TestCheckResourceAttr(fullName, "origin_protocol", update.Proto),
				),
			},
		},
	})
}
