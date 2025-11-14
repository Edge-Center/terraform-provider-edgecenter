//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func GetMockProvider(httpClient *http.Client, baseUrl string) *schema.Provider {
	p := edgecenter.Provider()
	originFunc := p.ConfigureContextFunc
	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		config, diags := originFunc(ctx, d)
		config.(*edgecenter.Config).HTTPClient = httpClient
		config.(*edgecenter.Config).CloudBaseURL = baseUrl
		fmt.Println("Set HTTPClient")
		return config, diags
	}
	return p
}

func TestAccResellerImagesV2DataSource(t *testing.T) {
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
	checkEntityID := 936337
	checkEntityType := edgecloudV2.ResellerType

	client.ResellerImageV2.Delete(ctx, checkEntityType, checkEntityID, nil)

	riuReq := &edgecloudV2.ResellerImageV2UpdateRequest{
		ImageIDs:   &checkImageIDs,
		RegionID:   checkRegionID,
		EntityType: checkEntityType,
		EntityID:   checkEntityID,
	}

	_, _, err = client.ResellerImageV2.Update(ctx, riuReq)
	if err != nil {
		t.Error(err)
	}

	defer client.ResellerImageV2.Delete(ctx, checkEntityType, checkEntityID, nil)

	datasourceName := "data.edgecenter_reseller_imagesV2.rimgs"

	resellerImagesTemplate := fmt.Sprintf(`
			data "edgecenter_reseller_images" "rimgs" {
			entity_id = %d
			entity_type = %s
			}
		`, checkEntityID, checkEntityType)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: resellerImagesTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(datasourceName),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(checkRegionID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.EntityIDField, strconv.Itoa(checkEntityID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "2"),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsIsNullField, "false"),
				),
			},
		},
	})
}

func TestAccResellerImagesV2DataSource_ImageIDsIsNull_Null(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	checkImageIDs := nil
	checkRegionID := 8
	checkEntityID := 936337
	checkEntityType := edgecloudV2.ResellerType

	client.ResellerImageV2.Delete(ctx, checkEntityType, checkEntityID, nil)

	riuReq := &edgecloudV2.ResellerImageV2UpdateRequest{
		ImageIDs:   &checkImageIDs,
		RegionID:   checkRegionID,
		EntityType: checkEntityType,
		EntityID:   checkEntityID,
	}

	_, _, err = client.ResellerImageV2.Update(ctx, riuReq)
	if err != nil {
		t.Error(err)
	}

	defer client.ResellerImageV2.Delete(ctx, checkEntityType, checkEntityID, nil)

	datasourceName := "data.edgecenter_reseller_imagesV2.rimgs"
	resellerImagesTemplate := fmt.Sprintf(`
			data "edgecenter_reseller_imagesV2" "rimgs" {
				entity_id = %d
				entity_type = "%s"
			}
		`, resellerImage.EntityID, resellerImage.EntityType,
	)

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"edgecenter": func() (*schema.Provider, error) {
				return p, nil
			},
		},
		Steps: []resource.TestStep{
			{
				Config: resellerImagesTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(datasourceName),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(resellerImage.RegionID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.EntityIDField, strconv.Itoa(resellerImage.EntityID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsIsNullField, "true"),
				),
			},
		},
	})
}

func TestAccResellerImagesV2DataSource_ImageIDsIsNull_Empty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Error(err)
	}

	checkImageIDs := edgecloudV2.ImageIDs{}
	checkRegionID := 8
	checkEntityID := 936337
	checkEntityType := edgecloudV2.ResellerType

	client.ResellerImageV2.Delete(ctx, checkEntityType, checkEntityID, nil)

	riuReq := &edgecloudV2.ResellerImageV2UpdateRequest{
		ImageIDs:   &checkImageIDs,
		RegionID:   checkRegionID,
		EntityType: checkEntityType,
		EntityID:   checkEntityID,
	}

	_, _, err = client.ResellerImageV2.Update(ctx, riuReq)
	if err != nil {
		t.Error(err)
	}

	defer client.ResellerImageV2.Delete(ctx, checkEntityType, checkEntityID, nil)

	datasourceName := "data.edgecenter_reseller_imagesV2.rimgs"
	resellerImagesTemplate := fmt.Sprintf(`
			data "edgecenter_reseller_imagesV2" "rimgs" {
				entity_id = %d
				entity_type = "%s"
			}
		`, resellerImage.EntityID, resellerImage.EntityType,
	)

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"edgecenter": func() (*schema.Provider, error) {
				return p, nil
			},
		},
		Steps: []resource.TestStep{
			{
				Config: resellerImagesTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(datasourceName),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.RegionIDField, strconv.Itoa(resellerImage.RegionID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.EntityIDField, strconv.Itoa(resellerImage.EntityID)),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsField+".#", "0"),
					resource.TestCheckResourceAttr(datasourceName, edgecenter.ResellerImagesOptionsField+".0."+edgecenter.ImageIDsIsNullField, "false"),
				),
			},
		},
	})
}
