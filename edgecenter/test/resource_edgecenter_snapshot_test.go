//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/volume/v1/volumes"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccSnapshot(t *testing.T) {
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

	type Params struct {
		Name        string
		Description string
		Status      string
		Size        int
		VolumeID    string
	}

	create := Params{
		Name:     "test",
		VolumeID: volumeID,
	}

	update := Params{
		Name:     "test",
		VolumeID: volumeID,
	}

	resourceName := "edgecenter_snapshot.acctest"
	importStateIDPrefix := fmt.Sprintf("%s:%s:", os.Getenv("TEST_PROJECT_ID"), os.Getenv("TEST_REGION_ID"))

	SnapshotTemplate := func(params *Params) string {
		additional := fmt.Sprintf("%s\n        %s", regionInfo(), projectInfo())

		template := fmt.Sprintf(`
		resource "edgecenter_snapshot" "acctest" {
			name = "%s"
			volume_id = "%s"
			%s
		`, params.Name, params.VolumeID, additional)

		return template + "\n}"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccSnapshotDestroy,
		Steps: []resource.TestStep{
			{
				Config: SnapshotTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "volume_id", create.VolumeID),
				),
			},
			{
				Config: SnapshotTemplate(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "volume_id", update.VolumeID),
				),
			},
			{
				ImportStateIdPrefix: importStateIDPrefix,
				ResourceName:        resourceName,
				ImportState:         true,
			},
		},
	})
}

func testAccSnapshotDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.SnapshotsPoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_snapshot" {
			continue
		}

		_, err := networks.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("snapshot still exists")
		}
	}

	return nil
}
