//go:build !cloud
// +build !cloud

package edgecenter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/storage"
)

func TestStorageSFTPDataSource(t *testing.T) {
	random := time.Now().Nanosecond()
	name := fmt.Sprintf("terraformtestsftp%d", random)
	location := "mia"

	resourceName := fmt.Sprintf("edgecenter_storage_sftp.%s_sftp", name)
	dataSourceName := fmt.Sprintf("data.edgecenter_storage_sftp.%s_sftp_data", name)

	templateCreate := func() string {
		return fmt.Sprintf(`
resource "edgecenter_storage_sftp" "%s_sftp" {
  name = "%s"
  location = "%s"
}
		`, name, name, location)
	}

	templateRead := func() string {
		return fmt.Sprintf(`
data "edgecenter_storage_sftp" "%s_sftp_data" {
  name = "%s"
}
		`, name, name)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_STORAGE_URL_VAR)
		},
		CheckDestroy: func(s *terraform.State) error {
			config := testAccProvider.Meta().(*Config)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			for _, rs := range s.RootModule().Resources {
				if rs.Type != "edgecenter_storage_sftp" {
					continue
				}
				opts := []func(opt *storage.StorageListHTTPV2Params){
					func(opt *storage.StorageListHTTPV2Params) { opt.Context = ctx },
					func(opt *storage.StorageListHTTPV2Params) { opt.ID = &rs.Primary.ID },
				}
				storages, err := config.StorageClient.StoragesList(opts...)
				if err != nil {
					return fmt.Errorf("find storage: %w", err)
				}
				if len(storages) == 0 {
					return nil
				}
				if storages[0].ProvisioningStatus == "ok" {
					return fmt.Errorf("storage #%s wasn't deleted correctrly", rs.Primary.ID)
				}
			}
			return nil
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: templateCreate(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, StorageSchemaLocation, location),
				),
			},
			{
				Config: templateRead(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(dataSourceName),
					resource.TestCheckResourceAttr(dataSourceName, StorageSchemaLocation, location),
				),
			},
		},
	})
}
