//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/router/v1/routers"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccRouterDataSource(t *testing.T) {
	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	clientNet, err := createTestClient(cfg.Provider, edgecenter.NetworksPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	clientRouter, err := createTestClient(cfg.Provider, edgecenter.RouterPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := networks.CreateOpts{
		Name:         networkTestName,
		CreateRouter: true,
	}

	networkID, err := createTestNetwork(clientNet, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer networks.Delete(clientNet, networkID)

	rs, err := routers.ListAll(clientRouter, routers.ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	router := rs[0]

	resourceName := "data.edgecenter_router.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_router" "acctest" {
			  %s
              %s
              name = "%s"
			}
		`, projectInfo(), regionInfo(), name)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(router.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", router.Name),
					resource.TestCheckResourceAttr(resourceName, "id", router.ID),
				),
			},
		},
	})
}
