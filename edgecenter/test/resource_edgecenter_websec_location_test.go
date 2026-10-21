//go:build websec

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccWebsecLocation(t *testing.T) {
	t.Parallel()

	resourceName := "edgecenter_websec_location.acctest"
	groupName := testName("websec-location-group")
	domain := testName("websec-location") + ".example.com"

	template := func(pattern, extra string) string {
		return fmt.Sprintf(`
resource "edgecenter_websec_source_group" "acctest" {
  name = "%s"

  source {
    source = "example.com"
  }
}

resource "edgecenter_websec_resource" "acctest" {
  domain       = "%s"
  source_group = edgecenter_websec_source_group.acctest.id
}

resource "edgecenter_websec_location" "acctest" {
  resource_id = edgecenter_websec_resource.acctest.id
  name        = "static"
  location    = "%s"
%s
}
`, groupName, domain, pattern, extra)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_WEBSEC_URL_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: template("/static/", ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "resource_id", "edgecenter_websec_resource.acctest", "id"),
					resource.TestCheckResourceAttr(resourceName, "location", "/static/"),
					resource.TestCheckResourceAttr(resourceName, "active", "true"),
					resource.TestCheckResourceAttr(resourceName, "weight", "1"),
					resource.TestCheckResourceAttr(resourceName, "source_group", "0"),
					resource.TestCheckResourceAttr(resourceName, "options.#", "0"),
				),
			},
			{
				Config: template("^/assets/", `
  weight       = 10
  source_group = edgecenter_websec_source_group.acctest.id

  options {
    asn_acl {
      policy_type     = "deny"
      excepted_values = [64496]
    }
  }`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "location", "^/assets/"),
					resource.TestCheckResourceAttr(resourceName, "weight", "10"),
					resource.TestCheckResourceAttrPair(resourceName, "source_group", "edgecenter_websec_source_group.acctest", "id"),
					resource.TestCheckResourceAttr(resourceName, "options.0.asn_acl.0.policy_type", "deny"),
					resource.TestCheckResourceAttr(resourceName, "options.0.asn_acl.0.excepted_values.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					location, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("not found: %s", resourceName)
					}

					return location.Primary.Attributes["resource_id"] + ":" + location.Primary.ID, nil
				},
			},
		},
	})
}
