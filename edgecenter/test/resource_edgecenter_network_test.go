//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccNetwork(t *testing.T) {
	t.Parallel()
	type Params struct {
		Name        string
		Type        string
		MetadataMap string
	}

	paramsCreate := Params{
		Name: "create_test",
		Type: "vxlan",
		MetadataMap: `{
				key1 = "val1"
				key2 = "val2"
			}`,
	}

	paramsUpdate := Params{
		Name: "update_test",
		MetadataMap: `{
				key3 = "val3"
			  }`,
	}

	resourceName := "edgecenter_network.acctest"
	importStateIDPrefix := fmt.Sprintf("%s:%s:", os.Getenv("TEST_PROJECT_ID"), os.Getenv("TEST_REGION_ID"))

	NetworkTemplate := func(params *Params) string {
		template := fmt.Sprintf(`
		resource "edgecenter_network" "acctest" {
			name = "%s"
	  		metadata_map = %s
			%s
			%s
		`, params.Name, params.MetadataMap, regionInfo(), projectInfo())

		if params.Type != "" {
			template += fmt.Sprintf("type = \"%s\"\n", params.Type)
		}

		return template + "\n}"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: NetworkTemplate(&paramsCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", paramsCreate.Name),
					resource.TestCheckResourceAttr(resourceName, "type", paramsCreate.Type),
					resource.TestCheckResourceAttr(resourceName, "metadata_map.key1", "val1"),
					resource.TestCheckResourceAttr(resourceName, "metadata_map.key2", "val2"),
					testAccCheckMetadata(t, resourceName, true, map[string]string{
						"key1": "val1",
						"key2": "val2",
					}),
				),
			},
			{
				Config: NetworkTemplate(&paramsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", paramsUpdate.Name),
					resource.TestCheckResourceAttr(resourceName, "type", paramsCreate.Type),
					testAccCheckMetadata(t, resourceName, true, map[string]string{
						"key3": "val3",
					}),
					testAccCheckMetadata(t, resourceName, false, map[string]string{
						"key1": "val1",
					}),
					testAccCheckMetadata(t, resourceName, false, map[string]string{
						"key2": "val2",
					}),
				),
			},
			{
				ImportStateIdPrefix: importStateIDPrefix,
				ResourceName:        resourceName,
				ImportState:         true,
			},
		},
	})
}

func testAccNetworkDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.NetworksPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_network" {
			continue
		}

		_, err := networks.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("network still exists")
		}
	}

	return nil
}
