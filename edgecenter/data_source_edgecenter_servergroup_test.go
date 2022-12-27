//go:build cloud
// +build cloud

package edgecenter

import (
	"fmt"
	"testing"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/servergroup/v1/servergroups"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccServerGroupDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := CreateTestClient(cfg.Provider, serverGroupsPoint, versionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := servergroups.CreateOpts{Name: "name", Policy: servergroups.AntiAffinityPolicy}
	serverGroup, err := servergroups.Create(client, opts).Extract()
	if err != nil {
		t.Fatal(err)
	}

	fullName := "data.edgecenter_servergroup.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_servergroup" "acctest" {
			  %s
              %s
              name = "%s"
			}
		`, projectInfo(), regionInfo(), name)
	}

	defer servergroups.Delete(client, serverGroup.ServerGroupID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(opts.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", serverGroup.Name),
					resource.TestCheckResourceAttr(fullName, "id", serverGroup.ServerGroupID),
					resource.TestCheckResourceAttr(fullName, "policy", serverGroup.Policy.String()),
				),
			},
		},
	})
}
