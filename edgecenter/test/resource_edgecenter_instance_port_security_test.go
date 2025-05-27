//go:build cloud_resource

package edgecenter_test

import (
	"context"
	"fmt"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"strconv"
	"strings"
	"testing"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const InstancePortSecurityResourceName = "instance_port_security"

var InstancePortSecurityInstanceName = fmt.Sprintf("%s-%s", InstancePortSecurityResourceName, instanceTestName)

func TestAccInstancePortSecurity(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestCloudClient()
	if err != nil {
		t.Fatal(err)
	}

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
		Name:     InstancePortSecurityResourceName + volumeTestName,
		Size:     5,
		TypeName: "standard",
	}

	volumeID, err := createTestVolumeV2(ctx, client, &volumeOpts)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Volumes.Delete(ctx, volumeID)

	opts := networks.CreateOpts{
		Name: InstancePortSecurityResourceName + networkTestName,
	}

	networkID, err := createTestNetwork(clientNet, opts)
	if err != nil {
		t.Fatal(err)
	}

	defer networks.Delete(clientNet, networkID)

	optsSubnet := subnets.CreateOpts{
		Name:      InstancePortSecurityResourceName + subnetTestName,
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

	newSG1, _, err := client.SecurityGroups.Create(ctx, &edgecloudV2.SecurityGroupCreateRequest{
		SecurityGroup: edgecloudV2.SecurityGroupCreateRequestInner{
			Name: InstancePortSecurityResourceName,
		},
	})
	defer client.SecurityGroups.Delete(ctx, newSG1.ID)

	sgsUpdate := []string{fmt.Sprintf("\"%s\"", newSG1.ID)}

	if err != nil {
		t.Fatal(err)
	}

	interfaces := []edgecloudV2.InstanceInterface{{
		Type:           "subnet",
		NetworkID:      networkID,
		SubnetID:       subnetID,
		SecurityGroups: sgs,
	},
	}

	instanceCreateOpts := edgecloudV2.InstanceCreateRequest{
		Names:         []string{InstancePortSecurityInstanceName},
		NameTemplates: []string{},
		Flavor:        FlavorG1Standart24,
		Password:      "password",
		Username:      "user",
		Volumes:       volumes,
		Interfaces:    interfaces,
	}

	taskInstanceResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, client.Instances.Create, &instanceCreateOpts, client, edgecenter.InstanceCreateTimeout)
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

	type Params struct {
		PortSecurityDisabled bool
		OverwriteExisting    bool
		SecurityGroups       []string
	}

	create := Params{
		PortSecurityDisabled: false,
		OverwriteExisting:    false,
		SecurityGroups:       []string{},
	}

	update := Params{
		PortSecurityDisabled: false,
		OverwriteExisting:    true,
		SecurityGroups:       sgsUpdate,
	}
	resourceName := "edgecenter_instance_port_security.instance_port_security_acctest"

	instancePortSecurityTemplate := func(params *Params) string {
		return fmt.Sprintf(`
			resource "edgecenter_instance_port_security" "instance_port_security_acctest" {
			  %s
			  %s
			  port_id = "%s"
			  instance_id = "%s"
			  port_security_disabled = %t
			  security_groups {
				overwrite_existing = %t
				security_group_ids = [%s] 
              }
			}
		`, projectInfo(), regionInfo(), portID, instanceID, params.PortSecurityDisabled, params.OverwriteExisting, strings.Join(params.SecurityGroups, ", "))
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: instancePortSecurityTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "security_groups.0.all_security_group_ids.0", sgID),
					resource.TestCheckResourceAttr(resourceName, "security_groups.0.overwrite_existing", fmt.Sprintf("%t", create.OverwriteExisting)),
					resource.TestCheckResourceAttr(resourceName, "port_security_disabled", fmt.Sprintf("%t", create.PortSecurityDisabled)),
					resource.TestCheckResourceAttr(resourceName, "security_groups.0.security_group_ids.#", strconv.Itoa(0)),
					resource.TestCheckResourceAttr(resourceName, "security_groups.0.all_security_group_ids.#", strconv.Itoa(1)),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: instancePortSecurityTemplate(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "security_groups.0.all_security_group_ids.0", newSG1.ID),
					resource.TestCheckResourceAttr(resourceName, "security_groups.0.overwrite_existing", fmt.Sprintf("%t", update.OverwriteExisting)),
					resource.TestCheckResourceAttr(resourceName, "port_security_disabled", fmt.Sprintf("%t", update.PortSecurityDisabled)),
					resource.TestCheckResourceAttr(resourceName, "security_groups.0.security_group_ids.#", strconv.Itoa(1)),
					resource.TestCheckResourceAttr(resourceName, "security_groups.0.all_security_group_ids.#", strconv.Itoa(1)),
				),
			},
		},
	})
}
