//go:build cloud_data_source

package edgecenter_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const InstancePortSecurityDatasourceName = "instance_port_security_datasource"

var InstancePortSecurityDatasourceInstanceName = fmt.Sprintf("%s-%s-datasource", InstancePortSecurityDatasourceName, instanceTestName)

func TestAccInstancePortSecurityDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client := cfg.CloudClient

	ctx := context.Background()

	imgs, _, err := client.Images.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	clientNet, err := createTestClient(cfg.Provider, edgecenter.NetworksPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	clientSubnet, err := createTestClient(cfg.Provider, edgecenter.SubnetPoint, edgecenter.VersionPointV1)
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

	volumeOpts := edgecloudV2.VolumeCreateRequest{
		ImageID:  img.ID,
		Source:   "image",
		Name:     InstancePortSecurityDatasourceName + volumeTestName,
		Size:     5,
		TypeName: "standard",
	}

	volumeID, err := createTestVolumeV2(ctx, client, &volumeOpts)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Volumes.Delete(ctx, volumeID)

	opts := networks.CreateOpts{
		Name: InstancePortSecurityDatasourceName + networkTestName,
	}

	networkID, err := createTestNetwork(clientNet, opts)
	if err != nil {
		t.Fatal(err)
	}

	defer networks.Delete(clientNet, networkID)

	optsSubnet := subnets.CreateOpts{
		Name:      InstancePortSecurityDatasourceName + subnetTestName,
		NetworkID: networkID,
	}

	subnetID, err := createTestSubnet(clientSubnet, optsSubnet)
	if err != nil {
		t.Fatal(err)
	}
	bootIndex := 0

	volumes := []edgecloudV2.InstanceVolumeCreate{
		{
			Source:    "existing-volume",
			BootIndex: &bootIndex,
			VolumeID:  volumeID,
		},
	}

	allSGs, _, err := client.SecurityGroups.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	sgID := allSGs[0].ID
	sgs := []edgecloudV2.ID{{ID: sgID}}

	interfaces := []edgecloudV2.InstanceInterface{{
		Type:           "subnet",
		NetworkID:      networkID,
		SubnetID:       subnetID,
		SecurityGroups: sgs,
	},
	}

	instanceCreateOpts := edgecloudV2.InstanceCreateRequest{
		Names:         []string{InstancePortSecurityDatasourceInstanceName},
		NameTemplates: []string{},
		Flavor:        FlavorG1Standart24,
		Password:      "password",
		Username:      "user",
		Volumes:       volumes,
		Interfaces:    interfaces,
	}

	taskInstanceResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.Instances.Create, &instanceCreateOpts, client)
	if err != nil {
		t.Fatal(err)
	}
	instanceID := taskInstanceResult.Instances[0]
	defer client.Instances.Delete(ctx, instanceID, nil)

	instancePortInterfaces, _, err := client.Instances.InterfaceList(ctx, instanceID)
	if err != nil {
		t.Fatal(err)
	}
	portID := instancePortInterfaces[0].PortID

	resourceName := "data.edgecenter_instance_port_security.instance_port_security_acctest"

	instancePortSecurityTemplate := fmt.Sprintf(`
			data "edgecenter_instance_port_security" "instance_port_security_acctest" {
			  %s
			  %s
			  port_id = "%s"
			  instance_id = "%s"
			}
		`, projectInfo(), regionInfo(), portID, instanceID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: instancePortSecurityTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "all_security_group_ids.0", sgID),
					resource.TestCheckResourceAttr(resourceName, "port_security_disabled", fmt.Sprintf("%t", false)),
					resource.TestCheckResourceAttr(resourceName, "all_security_group_ids.#", strconv.Itoa(1)),
				),
			},
		},
	})
}
