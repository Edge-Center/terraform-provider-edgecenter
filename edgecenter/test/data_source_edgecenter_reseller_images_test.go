//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"strconv"
	"testing"
)

func TestAccResellerImagesDataSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	t.Parallel()

	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	checkImageIDs := edgecloudV2.ImageIDs{
		"0052a312-e6d8-4177-8e29-b017a3a6b588",
		"b5b4d65d-945f-4b98-ab6f-332319c724ef",
	}
	checkRegionID := 8
	checkResellerID := 936337

	client.ResellerImage.Delete(ctx, checkResellerID)

	riuReq := &edgecloudV2.ResellerImageUpdateRequest{
		ImageIDs:   &checkImageIDs,
		RegionID:   checkRegionID,
		ResellerID: checkResellerID,
	}

	_, _, err = client.ResellerImage.Update(ctx, riuReq)
	if err != nil {
		t.Error(err)
	}

	defer client.ResellerImage.Delete(ctx, checkResellerID)

	datasourceName := "data.edgecenter_reseller_images.rimgs"

	resellerImagesTemplate := fmt.Sprintf(`
			data "edgecenter_reseller_images" "rimgs" {
			reseller_id = %d
			}
		`, checkResellerID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: resellerImagesTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(datasourceName),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerIDField, strconv.Itoa(checkResellerID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "2"),
				),
			},
		},
	})
}
