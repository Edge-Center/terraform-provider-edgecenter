//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccSubnetDataSource(t *testing.T) {
	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	clientNet, err := createTestClient(cfg.Provider, edgecenter.NetworksPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	clientSubnet, err := createTestClient(cfg.Provider, edgecenter.SubnetPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := networks.CreateOpts{
		Name: networkTestName,
	}

	networkID, err := createTestNetwork(clientNet, opts)
	if err != nil {
		t.Fatal(err)
	}

	defer deleteTestNetwork(clientNet, networkID)

	optsSubnet1 := subnets.CreateOpts{
		Name:      "test-subnet1",
		NetworkID: networkID,
		Metadata:  map[string]string{"key1": "val1", "key2": "val2"},
	}

	subnet1ID, err := createTestSubnet(clientSubnet, optsSubnet1, "192.168.41.0/24")
	if err != nil {
		t.Fatal(err)
	}

	optsSubnet2 := subnets.CreateOpts{
		Name:      "test-subnet2",
		NetworkID: networkID,
		Metadata:  map[string]string{"key1": "val1", "key3": "val3"},
	}

	subnet2ID, err := createTestSubnet(clientSubnet, optsSubnet2, "192.168.43.0/24")
	if err != nil {
		t.Fatal(err)
	}

	resourceName := "data.edgecenter_subnet.acctest"
	tpl1 := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_subnet" "acctest" {
  			%s
			%s
			name = "%s"
			metadata_k="key1"
			}
		`, projectInfo(), regionInfo(), name)
	}

	tpl2 := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_subnet" "acctest" {
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
				Config: tpl1(optsSubnet1.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", optsSubnet1.Name),
					resource.TestCheckResourceAttr(resourceName, "id", subnet1ID),
					resource.TestCheckResourceAttr(resourceName, "network_id", networkID),
					edgecenter.TestAccCheckMetadata(resourceName, true, map[string]string{
						"key1": "val1", "key2": "val2",
					}),
				),
			},
			{
				Config: tpl2(optsSubnet2.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", optsSubnet2.Name),
					resource.TestCheckResourceAttr(resourceName, "id", subnet2ID),
					// resource.TestCheckResourceAttr(resourceName, "network_id", networkID),
					edgecenter.TestAccCheckMetadata(resourceName, true, map[string]string{
						"key3": "val3",
					}),
				),
			},
		},
	})
}
