//go:build cdn

package edgecenter_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOriginGroup(t *testing.T) {
	t.Parallel()
	resourceName := "edgecenter_cdn_origingroup.acctest"

	type Params struct {
		Source  string
		Enabled string
	}

	create := Params{"google.com", "true"}
	update := Params{"tut.by", "false"}

	template := func(params *Params) string {
		return fmt.Sprintf(`
            resource "edgecenter_cdn_origingroup" "acctest" {
			  name = "terraform_acctest_group"
			  use_next = true

			  origin {
			    source = "%s"
				enabled = %s
			  }

			  origin {
			    source = "yandex.ru"
			    enabled = true
			  }
			  
   			  consistent_balancing = true
			}
		`, params.Source, params.Enabled)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR, EC_CDN_URL_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: template(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "terraform_acctest_group"),
					or(
						resource.TestCheckResourceAttr(resourceName, "origin.0.source", create.Source),
						resource.TestCheckResourceAttr(resourceName, "origin.1.source", create.Source),
					),
					or(
						resource.TestCheckResourceAttr(resourceName, "origin.0.enabled", create.Enabled),
						resource.TestCheckResourceAttr(resourceName, "origin.1.enabled", create.Enabled),
					),
				),
			},
			{
				Config: template(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "terraform_acctest_group"),
					or(
						resource.TestCheckResourceAttr(resourceName, "origin.0.source", update.Source),
						resource.TestCheckResourceAttr(resourceName, "origin.1.source", update.Source),
					),
					or(
						resource.TestCheckResourceAttr(resourceName, "origin.0.enabled", update.Enabled),
						resource.TestCheckResourceAttr(resourceName, "origin.1.enabled", update.Enabled),
					),
				),
			},
		},
	})
}

func or(checks ...resource.TestCheckFunc) resource.TestCheckFunc {
	return func(t *terraform.State) error {
		var composed string

		for _, check := range checks {
			err := check(t)
			if err == nil {
				return nil
			}

			composed += err.Error() + "; "
		}

		return errors.New(composed)
	}
}
