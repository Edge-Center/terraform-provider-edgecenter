//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/keypair/v2/keypairs"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccKeyPair(t *testing.T) {
	t.Parallel()
	type Params struct {
		Name string
		PK   string
	}

	create := Params{
		Name: "test",
		PK:   pkTest,
	}

	resourceName := "edgecenter_keypair.acctest"

	kpTemplate := func(params *Params) string {
		return fmt.Sprintf(`
			resource "edgecenter_keypair" "acctest" {
			  %s
			  public_key = "%s"
			  sshkey_name = "%s"
			}
		`, projectInfo(), params.PK, params.Name)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccKeypairDestroy,
		Steps: []resource.TestStep{
			{
				Config: kpTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "sshkey_name", create.Name),
					resource.TestCheckResourceAttr(resourceName, "public_key", create.PK),
				),
			},
		},
	})
}

func testAccKeypairDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.KeypairsPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_keypair" {
			continue
		}

		_, err := keypairs.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("KeyPair still exists")
		}
	}

	return nil
}
