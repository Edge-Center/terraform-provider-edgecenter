//go:build websec

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccWebsecSourceGroup(t *testing.T) {
	t.Parallel()

	resourceName := "edgecenter_websec_source_group.acctest"
	name := testName("websec-sources")

	template := func(weight int, comment string) string {
		return fmt.Sprintf(`
resource "edgecenter_websec_source_group" "acctest" {
  name = "%s"

  source {
    source  = "example.com"
    weight  = %d
    comment = "%s"
  }
}
`, name, weight, comment)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_WEBSEC_URL_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: template(50, "created by an acceptance test"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "use_next", "true"),
					resource.TestCheckResourceAttr(resourceName, "source.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "source.0.source", "example.com"),
					resource.TestCheckResourceAttr(resourceName, "source.0.mode", "primary"),
					resource.TestCheckResourceAttr(resourceName, "source.0.weight", "50"),
					resource.TestCheckResourceAttr(resourceName, "source.0.comment", "created by an acceptance test"),
				),
			},
			{
				Config: template(100, "updated by an acceptance test"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "source.0.weight", "100"),
					resource.TestCheckResourceAttr(resourceName, "source.0.comment", "updated by an acceptance test"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
