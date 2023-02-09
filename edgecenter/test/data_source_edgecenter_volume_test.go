//go:build cloud

package edgecenter_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/volume/v1/volumes"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	volumeTestName = "test-volume"
	volumeTestSize = 1
)

func TestAccVolumeDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := CreateTestClient(cfg.Provider, edgecenter.VolumesPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := volumes.CreateOpts{
		Name:     volumeTestName,
		Size:     volumeTestSize,
		Source:   volumes.NewVolume,
		TypeName: volumes.Standard,
	}

	volumeID, err := createTestVolume(client, opts)
	if err != nil {
		t.Fatal(err)
	}

	defer volumes.Delete(client, volumeID, volumes.DeleteOpts{})

	fullName := "data.edgecenter_volume.acctest"
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
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", opts.Name),
					resource.TestCheckResourceAttr(fullName, "id", volumeID),
					resource.TestCheckResourceAttr(fullName, "size", strconv.Itoa(opts.Size)),
				),
			},
		},
	})
}

func createTestVolume(client *edgecloud.ServiceClient, opts volumes.CreateOpts) (string, error) {
	res, err := volumes.Create(client, opts).Extract()
	if err != nil {
		return "", err
	}

	taskID := res.Tasks[0]
	volumeID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, edgecenter.VolumeCreatingTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		volumeID, err := volumes.ExtractVolumeIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve volume ID from task info: %w", err)
		}
		return volumeID, nil
	},
	)
	if err != nil {
		return "", err
	}
	return volumeID.(string), nil
}