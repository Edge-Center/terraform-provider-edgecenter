//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const DefaultAvailabilityZone = "nova"

func TestAccAvailabilityZoneDataSource(t *testing.T) {
	t.Parallel()

	resourceName := "data.edgecenter_availability_zone.acctest"
	tpl := func() string {
		return fmt.Sprintf(`
			data "edgecenter_availability_zone" "acctest" {
			  %s
			}
		`, regionInfo())
	}

	regionID, _, err := getRegionIDAndProjectID()
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", strconv.Itoa(regionID)),
					resource.TestCheckResourceAttr(resourceName, "region_id", strconv.Itoa(regionID)),
					resource.TestCheckResourceAttr(resourceName, "availability_zones.0", DefaultAvailabilityZone),
				),
			},
		},
	})
}
