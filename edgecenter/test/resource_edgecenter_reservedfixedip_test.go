//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/reservedfixedip/v1/reservedfixedips"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccReservedFixedIP(t *testing.T) {
	t.Parallel()
	type Params struct {
		Type  string
		IsVip bool
	}

	createExternal := Params{
		Type:  "external",
		IsVip: true,
	}

	updateExternal := Params{
		Type:  "external",
		IsVip: false,
	}

	resourceName := "edgecenter_reservedfixedip.acctest"

	ripTemplateExternal := func(params *Params) string {
		return fmt.Sprintf(`
			resource "edgecenter_reservedfixedip" "acctest" {
			  %s
              %s
			  is_vip = %t
			  type = "%s"
			}
		`, projectInfo(), regionInfo(), params.IsVip, params.Type)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccReservedFixedIPDestroy,
		Steps: []resource.TestStep{
			{
				Config: ripTemplateExternal(&createExternal),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", createExternal.Type),
					resource.TestCheckResourceAttr(resourceName, "is_vip", fmt.Sprintf("%t", createExternal.IsVip)),
				),
			},
			{
				Config: ripTemplateExternal(&updateExternal),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "type", updateExternal.Type),
					resource.TestCheckResourceAttr(resourceName, "is_vip", fmt.Sprintf("%t", updateExternal.IsVip)),
				),
			},
		},
	})
}

func testAccReservedFixedIPDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.FloatingIPsPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_reservedfixedip" {
			continue
		}

		_, err := reservedfixedips.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("ReservedFixedIP still exists")
		}
	}

	return nil
}
