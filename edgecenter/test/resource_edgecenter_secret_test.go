//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/secret/v1/secrets"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccSecret(t *testing.T) {
	t.Parallel()
	resourceName := "edgecenter_secret.acctest"
	kpTemplate := fmt.Sprintf(`
	resource "edgecenter_secret" "acctest" {
	  %s
      %s
      name = "%s"
      private_key = %q
      certificate = %q
      certificate_chain = %q
      expiration = "2030-12-28T19:14:44.213"
	}
	`, projectInfo(), regionInfo(), secretTestName, privateKey, certificate, certificateChain)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: kpTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", secretTestName),
				),
			},
		},
	})
}

func testAccSecretDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.SecretPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_secret" {
			continue
		}

		_, err := secrets.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("secret still exists")
		}
	}

	return nil
}
