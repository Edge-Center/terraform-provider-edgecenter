//go:build !cloud
// +build !cloud

package edgecenter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccDnsZone(t *testing.T) {
	t.Parallel()
	random := time.Now().Nanosecond()
	name := fmt.Sprintf("terraformtestkey%d", random)
	zone := name + ".com"
	resourceName := fmt.Sprintf("%s.%s", edgecenter.DNSZoneResource, name)

	templateCreate := func() string {
		return fmt.Sprintf(`
resource "%s" "%s" {
  name = "%s"
}
		`, edgecenter.DNSZoneResource, name, zone)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_DNS_URL_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: templateCreate(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DNSZoneSchemaName, zone),
				),
			},
		},
	})
}
