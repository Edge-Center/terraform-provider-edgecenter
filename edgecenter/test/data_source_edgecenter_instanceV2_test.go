//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"

	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const DefaultSecurityGroupID string = "594d2778-ac8d-4f1f-9ba7-4be760b48458"

func TestAccInstanceV2DataSource(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client, err := createTestCloudClient()
	if err != nil {
		t.Fatal(err)
	}

	imgs, _, err := client.Images.List(ctx, nil)
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

	volumeID, err := createTestVolumeV2(ctx, client, &optsV)
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
				SecurityGroups: []edgecloudV2.ID{{ID: DefaultSecurityGroupID}},
			},
		},
	}

	taskResultCreate, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.Instances.Create, &opts, client, edgecenter.InstanceCreateTimeout)
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

	taskResultDelete, _, err := client.Instances.Delete(ctx, instanceID, &optsInstDel)
	if err != nil {
		t.Fatal(err)
	}
	_, err = utilV2.WaitAndGetTaskInfo(ctx, client, taskResultDelete.Tasks[0])
	if err != nil {
		t.Fatal(err)
	}

	if err := utilV2.ResourceIsDeleted(ctx, client.Instances.Get, instanceID); err != nil {
		t.Fatal(err)
	}

}
