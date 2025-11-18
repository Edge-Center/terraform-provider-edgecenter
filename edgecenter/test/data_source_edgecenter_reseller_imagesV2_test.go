//go:build cloud_reseller_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	checkResellerImagesEntityID   = 976100
	checkResellerImagesEntityType = edgecloudV2.ResellerType
)

func TestAccResellerImagesV2DataSource(t *testing.T) {
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

	client.ResellerImageV2.Delete(ctx, checkResellerImagesEntityType, checkResellerImagesEntityID, nil)

	riuReq := &edgecloudV2.ResellerImageV2UpdateRequest{
		ImageIDs:   &checkImageIDs,
		RegionID:   checkRegionID,
		EntityType: checkResellerImagesEntityType,
		EntityID:   checkResellerImagesEntityID,
	}

	_, _, err = client.ResellerImageV2.Update(ctx, riuReq)
	if err != nil {
		t.Error(err)
	}

	defer client.ResellerImageV2.Delete(ctx, checkResellerImagesEntityType, checkResellerImagesEntityID, nil)

	datasourceName := "data.edgecenter_reseller_imagesV2.rimgs"
	resellerImagesTemplate := fmt.Sprintf(`
			data "edgecenter_reseller_imagesV2" "rimgs" {
				entity_id = %d
				entity_type = "%s"
			}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType,
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: resellerImagesTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(datasourceName),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "2"),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsIsNullField, "false"),
				),
			},
		},
	})
}

func TestAccResellerImagesV2DataSource_ImageIDsIsNull_Null(t *testing.T) {
	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	var checkImageIDs *edgecloudV2.ImageIDs = nil
	checkRegionID := 8

	client.ResellerImageV2.Delete(ctx, checkResellerImagesEntityType, checkResellerImagesEntityID, nil)

	riuReq := &edgecloudV2.ResellerImageV2UpdateRequest{
		ImageIDs:   checkImageIDs,
		RegionID:   checkRegionID,
		EntityType: checkResellerImagesEntityType,
		EntityID:   checkResellerImagesEntityID,
	}

	_, _, err = client.ResellerImageV2.Update(ctx, riuReq)
	if err != nil {
		t.Error(err)
	}

	defer client.ResellerImageV2.Delete(ctx, checkResellerImagesEntityType, checkResellerImagesEntityID, nil)

	datasourceName := "data.edgecenter_reseller_imagesV2.rimgs"
	resellerImagesTemplate := fmt.Sprintf(`
			data "edgecenter_reseller_imagesV2" "rimgs" {
				entity_id = %d
				entity_type = "%s"
			}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType,
	)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: resellerImagesTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(datasourceName),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsIsNullField, "true"),
				),
			},
		},
	})
}

func TestAccResellerImagesV2DataSource_ImageIDsIsNull_Empty(t *testing.T) {
	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	checkImageIDs := edgecloudV2.ImageIDs{}
	checkRegionID := 8

	client.ResellerImageV2.Delete(ctx, checkResellerImagesEntityType, checkResellerImagesEntityID, nil)

	riuReq := &edgecloudV2.ResellerImageV2UpdateRequest{
		ImageIDs:   &checkImageIDs,
		RegionID:   checkRegionID,
		EntityType: checkResellerImagesEntityType,
		EntityID:   checkResellerImagesEntityID,
	}

	_, _, err = client.ResellerImageV2.Update(ctx, riuReq)
	if err != nil {
		t.Error(err)
	}

	defer client.ResellerImageV2.Delete(ctx, checkResellerImagesEntityType, checkResellerImagesEntityID, nil)

	datasourceName := "data.edgecenter_reseller_imagesV2.rimgs"
	resellerImagesTemplate := fmt.Sprintf(`
			data "edgecenter_reseller_imagesV2" "rimgs" {
				entity_id = %d
				entity_type = "%s"
			}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType,
	)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: resellerImagesTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(datasourceName),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsIsNullField, "false"),
				),
			},
		},
	})
}
