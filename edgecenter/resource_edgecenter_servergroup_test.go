//go:build cloud
// +build cloud

package edgecenter

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/servergroup/v1/servergroups"
)

func TestAccServerGroupResource(t *testing.T) {
	type Params struct {
		Name   string
		Policy string
	}

	create := Params{
		Name:   "test",
		Policy: servergroups.AntiAffinityPolicy.String(),
	}

	fullName := "edgecenter_servergroup.acctest"

	kpTemplate := func(params *Params) string {
		return fmt.Sprintf(`
			resource "edgecenter_servergroup" "acctest" {
			  %s
              %s
			  name = "%s"
			  policy = "%s"
			}
		`, projectInfo(), regionInfo(), params.Name, params.Policy)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccServerGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: kpTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", create.Name),
					resource.TestCheckResourceAttr(fullName, "policy", create.Policy),
				),
			},
		},
	})
}

func testAccServerGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	client, err := CreateTestClient(config.Provider, serverGroupsPoint, versionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_servergroup" {
			continue
		}

		_, err := servergroups.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("ServerGroup %s still exists", rs.Primary.ID)
		}
	}

	return nil
}
