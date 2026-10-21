//go:build websec

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccWebsecResource(t *testing.T) {
	t.Parallel()

	resourceName := "edgecenter_websec_resource.acctest"
	groupName := testName("websec-group")
	domain := testName("websec") + ".example.com"

	template := func(subdomains, extraOptions string) string {
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
  subdomains   = [%s]

  options {
    tls_versions {
      value = ["TLSv1.2", "TLSv1.3"]
    }
%s
  }
}
`, groupName, domain, subdomains, extraOptions)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_WEBSEC_URL_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: template("", ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "domain", domain),
					resource.TestCheckResourceAttr(resourceName, "active", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "protected_ip"),
					resource.TestCheckResourceAttr(resourceName, "subdomains.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "options.0.tls_versions.0.value.#", "2"),
				),
			},
			{
				Config: template(fmt.Sprintf("%q", "www."+domain), `
    www_redirects {
      value = true
    }
    blacklist {
      value = ["198.51.100.7"]
    }`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "subdomains.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "options.0.www_redirects.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "options.0.blacklist.0.value.#", "1"),
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
