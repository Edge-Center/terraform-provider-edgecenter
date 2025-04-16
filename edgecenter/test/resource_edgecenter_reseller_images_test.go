//go:build cloud_resource

package edgecenter_test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const checkResellerImagesResellerID = 936337

func TestAccEdgecenterResellerImagesResource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	t.Parallel()

	resourceName := "edgecenter_reseller_images.rimgs"

	checkImageIDs0 := "0052a312-e6d8-4177-8e29-b017a3a6b588"
	checkImageIDs1 := "b5b4d65d-945f-4b98-ab6f-332319c724ef"
	checkRegionID := 8

	resellerImagesTemplate := fmt.Sprintf(`
			resource "edgecenter_reseller_images" "rimgs" {
  					reseller_id = %d
					options {
  					region_id = %d
  					image_ids = ["%s","%s"]
				}
			}
		`, checkResellerImagesResellerID, checkRegionID, checkImageIDs0, checkImageIDs1)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccResellerImagesDestroy,
		Steps: []resource.TestStep{
			{
				Config: resellerImagesTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerIDField, strconv.Itoa(checkResellerImagesResellerID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "2"),
				),
			},
		},
	})
}

func testAccResellerImagesDestroy(_ *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	clientV2, err := config.NewCloudClient()
	if err != nil {
		return err
	}

	_, resp, err := clientV2.ResellerImage.List(context.Background(), checkResellerImagesResellerID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("ResellerImage.List error: %w", err)
	}

	return fmt.Errorf("reseller images still exist")
}
