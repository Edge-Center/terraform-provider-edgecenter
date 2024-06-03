//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccProject(t *testing.T) {
	t.Parallel()
	random := time.Now().Nanosecond()
	project_test_name := fmt.Sprintf("terraformtestkey%d", random)
	test_description_1 := "test_description_1"
	test_description_2 := "test_description_2"
	resourceName := fmt.Sprintf("%s.%s", edgecenter.ProjectResource, project_test_name)

	type Params struct {
		Name        string
		Description string
	}

	create := Params{
		Name:        project_test_name,
		Description: test_description_1,
	}

	update := Params{
		Name:        project_test_name,
		Description: test_description_2,
	}

	template := func(p *Params) string {
		return fmt.Sprintf(`
resource "%s" "%s" {
  name = "%s"
  description ="%s"
}
		`, edgecenter.ProjectResource, p.Name, p.Name, p.Description)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckVars(t, EC_USERNAME_VAR, EC_PASSWORD_VAR)
		},
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: template(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.NameField, create.Name),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DescriptionField, create.Description),
				),
			},
			{
				Config: template(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, edgecenter.NameField, update.Name),
					resource.TestCheckResourceAttr(resourceName, edgecenter.DescriptionField, update.Description),
				),
			},
		},
	})
}
