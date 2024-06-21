//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

func TestAccSnapshotDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	volumeOpts := edgecloudV2.VolumeCreateRequest{
		Name:     "test-snapshot-volume",
		Size:     volumeSizeTest,
		Source:   edgecloudV2.VolumeSourceNewVolume,
		TypeName: edgecloudV2.VolumeTypeStandard,
	}

	volumeID, err := createTestVolumeV2(ctx, cfg.CloudClient, &volumeOpts)
	if err != nil {
		t.Fatal(err)
	}

	snapshotOpts := edgecloudV2.SnapshotCreateRequest{
		Name:     "snapshot-" + volumeTestName,
		VolumeID: volumeID,
	}

	taskResultCreate, err := utilV2.ExecuteAndExtractTaskResult(ctx, cfg.CloudClient.Snapshots.Create, &snapshotOpts, &cfg.CloudClient)
	if err != nil {
		t.Fatal(err)
	}

	snapshotID := taskResultCreate.Snapshots[0]

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
					resource.TestCheckResourceAttr(resourceName, "snapshot_id", snapshotID),
				),
			},
		},
	})

	taskSnapshotsDelete, _, err := cfg.CloudClient.Snapshots.Delete(ctx, snapshotID)
	if err != nil {
		t.Fatal(err)
	}
	err = utilV2.WaitForTaskComplete(ctx, &cfg.CloudClient, taskSnapshotsDelete.Tasks[0])
	if err != nil {
		t.Fatal(err)
	}

	if err := utilV2.ResourceIsDeleted(ctx, cfg.CloudClient.Snapshots.Get, snapshotID); err != nil {
		t.Fatal(err)
	}

	taskVolumesDelete, _, err := cfg.CloudClient.Volumes.Delete(ctx, volumeID)
	if err != nil {
		t.Fatal(err)
	}
	err = utilV2.WaitForTaskComplete(ctx, &cfg.CloudClient, taskVolumesDelete.Tasks[0])
	if err != nil {
		t.Fatal(err)
	}

	if err := utilV2.ResourceIsDeleted(ctx, cfg.CloudClient.Volumes.Get, volumeID); err != nil {
		t.Fatal(err)
	}
}
