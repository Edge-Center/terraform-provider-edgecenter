//go:build cloud_reseller_resource

package edgecenter_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const checkResellerImagesEntityID = 976100
const checkResellerImagesEntityType = edgecloudV2.ResellerType

func TestAccResellerImagesV2Resource(t *testing.T) {
	resourceName := "edgecenter_reseller_imagesV2.rimgs"

	checkImageIDs0 := "0052a312-e6d8-4177-8e29-b017a3a6b588"
	checkImageIDs1 := "b5b4d65d-945f-4b98-ab6f-332319c724ef"
	checkRegionID := 8

	resellerImagesTemplate := fmt.Sprintf(`
			resource "edgecenter_reseller_imagesV2" "rimgs" {
				entity_id = %[1]d
				entity_type = "%[2]s"
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

func TestAccResellerImagesV2Resource_ImageIDsIsNull(t *testing.T) {
	resourceName := "edgecenter_reseller_imagesV2.rimgs"
	checkRegionID := 8

	// Step 1: image_ids = [] && image_ids_is_null = false
	cfgEmptyFalse := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
			region_id          = %d
				image_ids          = []
				image_ids_is_null  = false
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, checkRegionID,
	)

	// Step 2: image_ids_is_null = true (no image_ids provided)
	cfgNullTrue := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id          = %d
				image_ids_is_null  = true
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, checkRegionID,
	)

	// Step 3: image_ids = [] (no image_ids_is_null provided)
	cfgEmptyOnly := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id          = %d
				image_ids          = []
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, checkRegionID,
	)

	// Step 4: image_ids_is_null=true and image_ids=[] both are set
	cfgInvalid := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id          = %d
				image_ids          = []
				image_ids_is_null  = true
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, checkRegionID,
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccResellerImagesV2Destroy,
		Steps: []resource.TestStep{
			// 1) image_ids = [] && image_ids_is_null = false -> image_ids = []
			{
				Config: cfgEmptyFalse,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsIsNullField, "false"),
				),
			},
			// 2) image_ids_is_null = true -> image_ids = [] (nil on API, empty in state)
			{
				Config: cfgNullTrue,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsIsNullField, "true"),
				),
			},
			// 3) image_ids = [] -> image_ids = [] (and image_ids_is_null becomes false)
			{
				Config: cfgEmptyOnly,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsIsNullField, "false"),
				),
			},
			// 4) image_ids_is_null=true and image_ids=[] -> Error (both cannot be set)
			{
				Config: cfgInvalid,
				ExpectError: regexp.MustCompile(
					fmt.Sprintf(
						"%s must not be set when %s is true",
						edgecenter.ImageIDsField,
						edgecenter.ImageIDsIsNullField,
					),
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
