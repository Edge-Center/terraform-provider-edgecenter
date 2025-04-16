//go:build cloud_resource

package edgecenter_test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const checkResellerImagesEntityID = 936337
const checkResellerImagesEntityType = edgecloudV2.ResellerType

func TestAccEdgecenterResellerImagesV2Resource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	t.Parallel()

	resourceName := "edgecenter_reseller_imagesV2.rimgs"

	checkImageIDs0 := "0052a312-e6d8-4177-8e29-b017a3a6b588"
	checkImageIDs1 := "b5b4d65d-945f-4b98-ab6f-332319c724ef"
	checkRegionID := 8

	resellerImagesTemplate := fmt.Sprintf(`
			resource "edgecenter_reseller_images" "rimgs" {
  					entity_id = %[1]d
  					entity_type = %[2]s
					options {
  					region_id = %[3]d
  					image_ids = ["%[4]s","%[5]s"]
				}
			}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, checkRegionID, checkImageIDs0, checkImageIDs1)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccResellerImagesV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: resellerImagesTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "2"),
				),
			},
		},
	})
}

func testAccResellerImagesV2Destroy(_ *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	clientV2, err := config.NewCloudClient()
	if err != nil {
		return err
	}

	_, resp, err := clientV2.ResellerImageV2.List(context.Background(), checkResellerImagesEntityType, checkResellerImagesEntityID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("ResellerImage.List error: %w", err)
	}

	return fmt.Errorf("reseller images still exist")
}
