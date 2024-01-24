//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/image/v1/images"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/instances"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/types"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/volume/v1/volumes"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccInstanceDataSource(t *testing.T) {
	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	clientVolume, err := createTestClient(cfg.Provider, edgecenter.VolumesPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	clientImage, err := createTestClient(cfg.Provider, ImagesPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	imgs, err := images.ListAll(clientImage, nil)
	if err != nil {
		t.Fatal(err)
	}

	var img images.Image
	for _, i := range imgs {
		if i.OsDistro == osDistroTest {
			img = i
			break
		}
	}
	if img.ID == "" {
		t.Fatalf("images with os_distro='%s' does not exist", osDistroTest)
	}

	optsV := volumes.CreateOpts{
		Name:     volumeTestName,
		Size:     volumeSizeTest * 5,
		Source:   volumes.Image,
		TypeName: volumes.Standard,
		ImageID:  img.ID,
	}
	volumeID, err := createTestVolume(clientVolume, optsV)
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.InstancePoint, edgecenter.VersionPointV2)
	if err != nil {
		t.Fatal(err)
	}

	clientV1, err := createTestClient(cfg.Provider, edgecenter.InstancePoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := instances.CreateOpts{
		Names:  []string{instanceTestName},
		Flavor: flavorTest,
		Volumes: []instances.CreateVolumeOpts{{
			Source:    types.ExistingVolume,
			BootIndex: 0,
			VolumeID:  volumeID,
		}},
		Interfaces: []instances.InterfaceInstanceCreateOpts{
			{
				InterfaceOpts:  instances.InterfaceOpts{Type: types.ExternalInterfaceType},
				SecurityGroups: []edgecloud.ItemID{},
			},
		},
	}

	res, err := instances.Create(client, opts).Extract()
	if err != nil {
		t.Fatal(err)
	}

	taskID := res.Tasks[0]
	instanceID, err := tasks.WaitTaskAndReturnResult(clientVolume, taskID, true, edgecenter.InstanceCreatingTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(clientVolume, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		id, err := instances.ExtractInstanceIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve instance ID from task info: %w", err)
		}
		return id, nil
	},
	)
	if err != nil {
		t.Fatal(err)
	}

	defer instances.Delete(clientV1, instanceID.(string), instances.DeleteOpts{Volumes: []string{volumeID}})

	resourceName := "data.edgecenter_instance.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_instance" "acctest" {
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
				Config: tpl(instanceTestName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", instanceTestName),
					resource.TestCheckResourceAttr(resourceName, "id", instanceID.(string)),
				),
			},
		},
	})
}
