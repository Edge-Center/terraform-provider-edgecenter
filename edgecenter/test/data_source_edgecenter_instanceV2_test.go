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

func TestAccInstanceV2DataSource(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	imgs, _, err := cfg.CloudClient.Images.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	var img edgecloudV2.Image
	for _, i := range imgs {
		if i.OSDistro == osDistroTest {
			img = i
			break
		}
	}
	if img.ID == "" {
		t.Fatalf("images with os_distro='%s' does not exist", osDistroTest)
	}

	optsV := edgecloudV2.VolumeCreateRequest{
		Name:     volumeTestName,
		Size:     volumeSizeTest * 5,
		Source:   edgecloudV2.VolumeSourceImage,
		TypeName: edgecloudV2.VolumeTypeStandard,
		ImageID:  img.ID,
	}

	volumeID, err := createTestVolumeV2(ctx, cfg.CloudClient, &optsV)
	if err != nil {
		t.Fatal(err)
	}
	bootIndex := 0
	opts := edgecloudV2.InstanceCreateRequest{
		Names:  []string{instanceV2TestName},
		Flavor: flavorTest,
		Volumes: []edgecloudV2.InstanceVolumeCreate{{
			Source:    edgecloudV2.VolumeSourceExistingVolume,
			BootIndex: &bootIndex,
			VolumeID:  volumeID,
		}},
		Interfaces: []edgecloudV2.InstanceInterface{
			{
				Type:           edgecloudV2.InterfaceTypeExternal,
				SecurityGroups: []edgecloudV2.ID{},
			},
		},
	}

	taskResultCreate, err := utilV2.ExecuteAndExtractTaskResult(ctx, cfg.CloudClient.Instances.Create, &opts, cfg.CloudClient)
	if err != nil {
		t.Fatal(err)
	}

	instanceID := taskResultCreate.Instances[0]

	resourceName := "data.edgecenter_instanceV2.acctest"
	tpl := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_instanceV2" "acctest" {
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
				Config: tpl(instanceV2TestName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", instanceV2TestName),
					resource.TestCheckResourceAttr(resourceName, "id", instanceID),
				),
			},
		},
	})
	optsInstDel := edgecloudV2.InstanceDeleteOptions{
		Volumes: []string{volumeID},
	}

	taskResultDelete, _, err := cfg.CloudClient.Instances.Delete(ctx, instanceID, &optsInstDel)
	if err != nil {
		t.Fatal(err)
	}
	_, err = utilV2.WaitAndGetTaskInfo(ctx, cfg.CloudClient, taskResultDelete.Tasks[0])
	if err != nil {
		t.Fatal(err)
	}

	if err := utilV2.ResourceIsDeleted(ctx, cfg.CloudClient.Instances.Get, instanceID); err != nil {
		t.Fatal(err)
	}

}
