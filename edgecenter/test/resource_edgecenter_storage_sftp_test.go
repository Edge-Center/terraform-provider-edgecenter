//go:build !cloud
// +build !cloud

package edgecenter_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/storage"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccStorageSFTP(t *testing.T) {
	t.Parallel()
	random := time.Now().Nanosecond()
	alias := fmt.Sprintf("terraformtestalias%dsftp", random)
	resourceName := fmt.Sprintf("edgecenter_storage_sftp.terraformtest%d_sftp", random)

	templateCreate := func() string {
		return fmt.Sprintf(`
resource "edgecenter_storage_sftp" "terraformtest%d_sftp" {
  name = "terraformtest%d"
  location = "mia"
}
		`, random, random)
	}

	templateUpdate := func() string {
		return fmt.Sprintf(`
resource "edgecenter_storage_sftp" "terraformtest%d_sftp" {
  name = "terraformtest%d"
  location = "mia"
  http_servername_alias = "%s"
}
		`, random, random, alias)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_STORAGE_URL_VAR)
		},
		CheckDestroy: func(s *terraform.State) error {
			config := testAccProvider.Meta().(*edgecenter.Config)
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
					resource.TestCheckResourceAttr(resourceName, edgecenter.StorageSchemaLocation, "mia"),
				),
			},
			{
				Config: templateUpdate(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.StorageSFTPSchemaServerAlias, alias),
				),
			},
		},
	})
}