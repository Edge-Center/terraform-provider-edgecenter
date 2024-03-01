//go:build cloud_data_source

package edgecenter_test

import (
	"testing"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccSnapshotDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.VolumesPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := edgecenter.CreateVolumeOpts{
		Name:     volumeTestName,
		Size:     volumeSizeTest,
		Source:   edgecenter.NewVolume,
		TypeName: edgecenter.Standard,
	}

	volumeID, err := createTestVolume(client, opts)
	if err != nil {
		t.Fatal(err)
	}

	defer edgecenter.DeleteVolume(client, volumeID, edgecenter.DeleteVolumeOpts{})

	snapshotOpts := edgecenter.CreateSnapshotOpts{
		Name:     "snapshot_" + volumeTestName,
		VolumeID: volumeID,
	}

	snapshotID, err := createTestSnapshot(client, snapshotOpts)
	if err != nil {
		t.Fatal(err)
	}

	defer edgecenter.DeleteSnapshot(client, snapshotID, edgecenter.DeleteSnapshotOpts{})

	resourceName := "data.edgecenter_snapshot.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_snapshot" "acctest" {
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
				Config: tpl(snapshotOpts.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", snapshotOpts.Name),
					resource.TestCheckResourceAttr(resourceName, "volume_id", volumeID),
				),
			},
		},
	})
}
