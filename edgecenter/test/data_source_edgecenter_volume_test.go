//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/volume/v1/volumes"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccVolumeDataSource(t *testing.T) {
	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.VolumesPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := volumes.CreateOpts{
		Name:     volumeTestName,
		Size:     volumeSizeTest,
		Source:   volumes.NewVolume,
		TypeName: volumes.Standard,
	}

	volumeID, err := createTestVolume(client, opts)
	if err != nil {
		t.Fatal(err)
	}

	defer volumes.Delete(client, volumeID, volumes.DeleteOpts{})

	resourceName := "data.edgecenter_volume.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_volume" "acctest" {
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
				Config: tpl(opts.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", opts.Name),
					resource.TestCheckResourceAttr(resourceName, "id", volumeID),
					resource.TestCheckResourceAttr(resourceName, "size", strconv.Itoa(opts.Size)),
				),
			},
		},
	})
}
