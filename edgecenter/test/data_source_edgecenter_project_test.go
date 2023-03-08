//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/project/v1/projects"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccProjectDataSource(t *testing.T) {
	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.ProjectPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	prjs, err := projects.ListAll(client)
	if err != nil {
		t.Fatal(err)
	}

	if len(prjs) == 0 {
		t.Fatal("projects not found")
	}

	project := prjs[0]

	resourceName := "data.edgecenter_project.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_project" "acctest" {
              name = "%s"
			}
		`, name)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(project.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", project.Name),
					resource.TestCheckResourceAttr(resourceName, "id", strconv.Itoa(project.ID)),
				),
			},
		},
	})
}
