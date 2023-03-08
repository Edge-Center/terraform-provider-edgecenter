//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/region/v1/regions"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccRegionDataSource(t *testing.T) {
	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.RegionPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	rs, err := regions.ListAll(client)
	if err != nil {
		t.Fatal(err)
	}

	if len(rs) == 0 {
		t.Fatal("regions not found")
	}

	region := rs[0]

	resourceName := "data.edgecenter_region.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_region" "acctest" {
              name = "%s"
			}
		`, name)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(region.DisplayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", region.DisplayName),
					resource.TestCheckResourceAttr(resourceName, "id", strconv.Itoa(region.ID)),
				),
			},
		},
	})
}
