//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccNetworkDataSource(t *testing.T) {
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
		Name:     "test-network1",
		Metadata: map[string]string{"key1": "val1", "key2": "val2"},
	}

	network1ID, err := createTestNetwork(client, opts1)
	if err != nil {
		t.Fatal(err)
	}
	opts2 := networks.CreateOpts{
		Name:     "test-network2",
		Metadata: map[string]string{"key1": "val1", "key3": "val3"},
	}

	network2ID, err := createTestNetwork(client, opts2)
	if err != nil {
		t.Fatal(err)
	}

	defer deleteTestNetwork(client, network1ID)
	defer deleteTestNetwork(client, network2ID)

	resourceName := "data.edgecenter_network.acctest"
	tpl1 := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_network" "acctest" {
			  %s
              %s
              name = "%s"
              metadata_k="key1"
			}
		`, projectInfo(), regionInfo(), name)
	}
	tpl2 := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_network" "acctest" {
			  %s
              %s
              name = "%s"
 			  metadata_kv={
                  key3 = "val3"
			  }
			}
		`, projectInfo(), regionInfo(), name)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl1(opts1.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", opts1.Name),
					resource.TestCheckResourceAttr(resourceName, "id", network1ID),
					testAccCheckMetadata(t, resourceName, true, map[string]string{
						"key1": "val1", "key2": "val2",
					}),
				),
			},
			{
				Config: tpl2(opts2.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", opts2.Name),
					resource.TestCheckResourceAttr(resourceName, "id", network2ID),
					testAccCheckMetadata(t, resourceName, true, map[string]string{
						"key3": "val3",
					}),
				),
			},
		},
	})
}
