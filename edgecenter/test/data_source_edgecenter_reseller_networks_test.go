//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccResellerNetworksDataSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.NetworksPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts1 := networks.CreateOpts{
		Name:     "test-reseller-network1",
		Metadata: map[string]string{"key1": "val1", "key2": "val2"},
	}

	network1ID, err := createTestNetwork(client, opts1)
	if err != nil {
		t.Fatal(err)
	}

	defer deleteTestNetwork(client, network1ID)

	resourceName := "data.edgecenter_reseller_network.acctest"
	tpl1 := func() string {
		return fmt.Sprintf(`
			data "edgecenter_reseller_network" "acctest" {
              metadata_k=["key1"]
			}
		`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl1(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.NetworksField+".0."+edgecenter.NameField, opts1.Name),
					resource.TestCheckResourceAttr(resourceName, edgecenter.NetworksField+".0."+edgecenter.IDField, network1ID),
				),
			},
		},
	})
}
