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

const (
	checkResellerImagesEntityID   = 976100
	checkResellerImagesEntityType = edgecloudV2.ResellerType
)

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

func TestAccResellerImagesV2Resource_AllPublicImagesAreAvailable(t *testing.T) {
	resourceName := "edgecenter_reseller_imagesV2.rimgs"
	checkRegionID := 8

	// Step 1: image_ids = [] && all_public_images_are_available = false
	cfgEmptyFalse := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id                       = %d
				image_ids                       = []
				all_public_images_are_available = false
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, checkRegionID,
	)

	// Step 2: all_public_images_are_available = true (no image_ids provided)
	cfgNullTrue := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id                       = %d
				all_public_images_are_available = true
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, checkRegionID,
	)

	// Step 3: image_ids = [] (no all_public_images_are_available provided)
	cfgEmptyOnly := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id = %d
				image_ids = []
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, checkRegionID,
	)

	// Step 4: all_public_images_are_available=true and image_ids=[] both are set
	cfgInvalid := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id                       = %d
				image_ids                       = []
				all_public_images_are_available = true
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, checkRegionID,
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccResellerImagesV2Destroy,
		Steps: []resource.TestStep{
			// 1) image_ids=[] && all_public_images_are_available=false -> image_ids=[]
			{
				Config: cfgEmptyFalse,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.AllPublicImagesAreAvailableField, "false"),
				),
			},
			// 2) all_public_images_are_available=true -> image_ids=[] (nil on API, empty in state)
			{
				Config: cfgNullTrue,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.AllPublicImagesAreAvailableField, "true"),
				),
			},
			// 3) image_ids=[] -> image_ids=[] (and all_public_images_are_available becomes false)
			{
				Config: cfgEmptyOnly,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.AllPublicImagesAreAvailableField, "false"),
				),
			},
			// 4) all_public_images_are_available=true and image_ids=[] -> Error (both cannot be set)
			{
				Config: cfgInvalid,
				ExpectError: regexp.MustCompile(
					fmt.Sprintf(
						"%s must not be set when %s is true",
						edgecenter.ImageIDsField,
						edgecenter.AllPublicImagesAreAvailableField,
					),
				),
			},
		},
	})
}

func TestAccResellerImagesV2Resource_AllPublicImagesAreAvailable_Ordering(t *testing.T) {
	resourceName := "edgecenter_reseller_imagesV2.rimgs"
	lowRegionID := 8
	highRegionID := 2403

	cfgValid := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id                       = %d
				all_public_images_are_available = true
			}
			options {
				region_id = %d
				image_ids = []
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, lowRegionID, highRegionID,
	)

	cfgValidReversed := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id = %d
				image_ids = []
			}
			options {
				region_id                       = %d
				all_public_images_are_available = true
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, highRegionID, lowRegionID,
	)

	cfgInvalid := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id                       = %d
				all_public_images_are_available = true
			}
			options {
				region_id                       = %d
				image_ids                       = []
				all_public_images_are_available = true
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, lowRegionID, highRegionID,
	)

	cfgInvalidReversed := fmt.Sprintf(`
		resource "edgecenter_reseller_imagesV2" "rimgs" {
			entity_id   = %d
			entity_type = "%s"
			options {
				region_id          				= %d
				image_ids          				= []
				all_public_images_are_available = true
			}
			options {
				region_id         				= %d
				all_public_images_are_available = true
			}
		}
		`, checkResellerImagesEntityID, checkResellerImagesEntityType, highRegionID, lowRegionID,
	)

	expectedError := regexp.MustCompile(
		fmt.Sprintf(
			"%s must not be set when %s is true",
			edgecenter.ImageIDsField,
			edgecenter.AllPublicImagesAreAvailableField,
		),
	)
	checkFunc := resource.ComposeTestCheckFunc(
		testAccCheckResourceExists(resourceName),
		resource.TestCheckResourceAttr(resourceName, edgecenter.EntityIDField, strconv.Itoa(checkResellerImagesEntityID)),
		resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(highRegionID)),
		resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
		resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.AllPublicImagesAreAvailableField, "false"),
		resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".1."+edgecenter.RegionIDField, strconv.Itoa(lowRegionID)),
		resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".1."+edgecenter.ImageIDsField+".#", "0"),
		resource.TestCheckResourceAttr(resourceName, edgecenter.ResellerImagesOptionsField+".1."+edgecenter.AllPublicImagesAreAvailableField, "true"),
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccResellerImagesV2Destroy,
		Steps: []resource.TestStep{
			{Config: cfgValid, Check: checkFunc},
			{Config: cfgValidReversed, Check: checkFunc},
			{Config: cfgInvalid, ExpectError: expectedError},
			{Config: cfgInvalidReversed, ExpectError: expectedError},
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
