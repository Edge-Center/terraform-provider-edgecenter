//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/servergroup/v1/servergroups"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccServerGroupDataSource(t *testing.T) {
	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.ServerGroupsPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := servergroups.CreateOpts{Name: "name", Policy: servergroups.AntiAffinityPolicy}
	serverGroup, err := servergroups.Create(client, opts).Extract()
	if err != nil {
		t.Fatal(err)
	}

	resourceName := "data.edgecenter_servergroup.acctest"
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
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", serverGroup.Name),
					resource.TestCheckResourceAttr(resourceName, "id", serverGroup.ServerGroupID),
					resource.TestCheckResourceAttr(resourceName, "policy", serverGroup.Policy.String()),
				),
			},
		},
	})
}
