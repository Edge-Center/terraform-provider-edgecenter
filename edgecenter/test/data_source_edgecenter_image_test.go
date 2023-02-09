//go:build cloud

package edgecenter_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/image/v1/images"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccImageDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := CreateTestClient(cfg.Provider, edgecenter.ImagesPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	imgs, err := images.ListAll(client, images.ListOpts{})
	if err != nil {
		t.Fatal(err)
	}

	if len(imgs) == 0 {
		t.Fatal("images not found")
	}

	img := imgs[0]

	fullName := "data.edgecenter_image.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_image" "acctest" {
			  %s
              %s
              name = "%s"
			}
		`, projectInfo(), regionInfo(), name)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(img.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", img.Name),
					resource.TestCheckResourceAttr(fullName, "id", img.ID),
					resource.TestCheckResourceAttr(fullName, "min_disk", strconv.Itoa(img.MinDisk)),
					resource.TestCheckResourceAttr(fullName, "min_ram", strconv.Itoa(img.MinRAM)),
					resource.TestCheckResourceAttr(fullName, "os_distro", img.OsDistro),
					resource.TestCheckResourceAttr(fullName, "os_version", img.OsVersion),
				),
			},
		},
	})
}
