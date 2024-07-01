//go:build cloud_resource

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/image/v1/images"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/instance/v1/instances"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/securitygroups"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccInstance(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	clientImage, err := createTestClient(cfg.Provider, ImagesPoint, edgecenter.VersionPointV1)
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

	clientSec, err := createTestClient(cfg.Provider, edgecenter.SecurityGroupPoint, edgecenter.VersionPointV1)
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

	opts := networks.CreateOpts{
		Name: networkTestName,
	}

	networkID, err := createTestNetwork(clientNet, opts)
	if err != nil {
		t.Fatal(err)
	}

	defer networks.Delete(clientNet, networkID)

	optsSubnet := subnets.CreateOpts{
		Name:      subnetTestName,
		NetworkID: networkID,
	}

	subnetID, err := createTestSubnet(clientSubnet, optsSubnet)
	if err != nil {
		t.Fatal(err)
	}

	volumes := []instances.CreateVolumeOpts{
		{
			Source:    "existing-volume",
			BootIndex: 0,
		},
		{
			Source:    "existing-volume",
			BootIndex: 1,
		},
	}
	interfaces := []instances.InterfaceInstanceCreateOpts{{
		InterfaceOpts: instances.InterfaceOpts{
			Type:      "subnet",
			NetworkID: networkID,
			SubnetID:  subnetID,
		},
	}}
	updateInterfaces := []instances.InterfaceInstanceCreateOpts{{
		InterfaceOpts: instances.InterfaceOpts{
			Type:     "subnet",
			SubnetID: subnetID,
		},
	}}

	sgs, err := securitygroups.ListAll(clientSec, nil)
	if err != nil {
		t.Fatal(err)
	}

	secgroups := []edgecloud.ItemID{{ID: sgs[0].ID}}
	updateSg := []edgecloud.ItemID{
		{
			ID: "someid",
		},
	}
	metadata := instances.MetadataSetOpts{}
	metadata.Metadata = []instances.MetadataOpts{
		{
			Key:   "somekey",
			Value: "somevalue",
		},
	}
	updateMetadata := instances.MetadataSetOpts{}
	updateMetadata.Metadata = []instances.MetadataOpts{
		{
			Key:   "newsomekey",
			Value: "newsomevalue",
		},
	}

	createFixt := instances.CreateOpts{
		Names:          []string{"create_instance"},
		NameTemplates:  []string{},
		Flavor:         "g1-standard-2-4",
		Password:       "password",
		Username:       "user",
		Keypair:        "acctest",
		Volumes:        volumes,
		Interfaces:     interfaces,
		SecurityGroups: secgroups,
		Metadata:       &metadata,
		Configuration:  &metadata,
	}

	updateInterfacefixt := createFixt
	updateInterfacefixt.Interfaces = updateInterfaces

	updateInterfacefixt.SecurityGroups = updateSg

	updateFixt := createFixt
	updateFixt.Flavor = "g1-standard-2-8"
	updateFixt.Metadata = &updateMetadata
	updateFixt.Configuration = &updateMetadata

	type Params struct {
		Name           []string
		Flavor         string
		Password       string
		Username       string
		Keypair        string
		Publickey      string
		Image          string
		Interfaces     []map[string]string
		SecurityGroups []map[string]string
		MetaData       []map[string]string
		Configuration  []map[string]string
	}

	create := Params{
		Name:      []string{"create_instance"},
		Flavor:    "g1-standard-2-4",
		Password:  "password",
		Username:  "user",
		Keypair:   "acctest",
		Publickey: pkTest,
		Image:     img.ID,
		Interfaces: []map[string]string{
			{"type": "subnet", "network_id": networkID, "subnet_id": subnetID},
		},
		SecurityGroups: []map[string]string{{"id": sgs[0].ID, "name": sgs[0].Name}},
		MetaData:       []map[string]string{{"key": "somekey", "value": "somevalue"}},
		Configuration:  []map[string]string{{"key": "somekey", "value": "somevalue"}},
	}

	updateInterface := create
	updateInterface.Interfaces = []map[string]string{{"type": "subnet", "subnet_id": subnetID}}

	update := create
	update.Flavor = "g1-standard-2-8"
	update.MetaData = []map[string]string{{"key": "newsomekey", "value": "newsomevalue"}}
	update.Configuration = []map[string]string{{"key": "newsomekey", "value": "newsomevalue"}}

	instanceTemplate := func(params *Params) string {
		template := `
		locals {`

		template += fmt.Sprintf(`
			names = "%s"
            volumes_ids = [edgecenter_volume.first_volume.id, edgecenter_volume.second_volume.id]`, params.Name[0])

		template += fmt.Sprint(`
			interfaces = [`)
		for i := range params.Interfaces {
			template += fmt.Sprintf(`
			{
				type = "%s"
				network_id = "%s"
				subnet_id = "%s"
                fip_source = null
                existing_fip_id = null
                port_id = null
                ip_address = null
				
			},`, params.Interfaces[i]["type"], params.Interfaces[i]["network_id"], params.Interfaces[i]["subnet_id"])
		}
		template += fmt.Sprint(`]
			metadata = [`)
		for i := range params.MetaData {
			template += fmt.Sprintf(`
			{
				key = "%s"
				value = "%s"
			},`, params.MetaData[i]["key"], params.MetaData[i]["value"])
		}
		template += fmt.Sprint(`]
			configuration = [`)
		for i := range params.Configuration {
			template += fmt.Sprintf(`
			{
				key = "%s"
				value = "%s"
			},`, params.Configuration[i]["key"], params.Configuration[i]["value"])
		}
		template += fmt.Sprintf(`]
        }

        resource "edgecenter_volume" "first_volume" {
  			name = "boot volume"
  			type_name = "ssd_hiiops"
  			size = 5
  			image_id = "%[1]s"
  			%[7]s
			%[8]s
		}

		resource "edgecenter_volume" "second_volume" {
  			name = "second volume"
  			type_name = "ssd_hiiops"
  			size = 5
  			%[7]s
			%[8]s
		}

        resource "edgecenter_keypair" "kp" {
  			sshkey_name = "%[2]s"
            public_key = "%[3]s"
            %[8]s
		}

        resource "edgecenter_instance" "acctest" {
			flavor_id = "%[4]s"
           	name = local.names
           	keypair_name = edgecenter_keypair.kp.sshkey_name
           	password = "%[5]s"
           	username = "%[6]s"

			dynamic volume {
		  	iterator = vol
		  	for_each = local.volumes_ids
		  	content {
				boot_index = index(local.volumes_ids, vol.value)
				source = "existing-volume"
				volume_id = vol.value
				}
		  	}

			dynamic interface {
			iterator = ifaces
			for_each = local.interfaces
			content {
				type = ifaces.value.type
				network_id = ifaces.value.network_id
				subnet_id = ifaces.value.subnet_id
                fip_source = ifaces.value.fip_source
				existing_fip_id = ifaces.value.existing_fip_id
                port_id = ifaces.value.port_id
                ip_address = ifaces.value.ip_address
				}
			}

			dynamic metadata {
			iterator = md
			for_each = local.metadata
			content {	
				key = md.value.key
				value = md.value.value
				}
			}

			dynamic configuration {
			iterator = cfg
			for_each = local.configuration
			content {	
				key = cfg.value.key
				value = cfg.value.value
				}
			}

            %[7]s
			%[8]s

		`, params.Image, params.Keypair, params.Publickey, params.Flavor, params.Password, params.Username, regionInfo(), projectInfo())
		return template + "\n}"
	}

	resourceName := "edgecenter_instance.acctest"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: instanceTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					checkInstanceAttrs(resourceName, &createFixt),
				),
			},
			{
				Config: instanceTemplate(&updateInterface),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					checkInstanceAttrs(resourceName, &updateInterfacefixt),
				),
			},
		},
	})
}

func testAccInstanceDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*edgecenter.Config)
	client, err := createTestClient(config.Provider, edgecenter.InstancePoint, edgecenter.VersionPointV1)
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "edgecenter_instance" {
			continue
		}

		_, err := instances.Get(client, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("instance still exists")
		}
	}

	return nil
}

func checkInstanceAttrs(resourceName string, opts *instances.CreateOpts) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if s.Empty() == true {
			return fmt.Errorf("state not updated")
		}

		checksStore := []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "name", opts.Names[0]),
			resource.TestCheckResourceAttr(resourceName, "flavor_id", opts.Flavor),
			resource.TestCheckResourceAttr(resourceName, "keypair_name", opts.Keypair),
			resource.TestCheckResourceAttr(resourceName, "password", opts.Password),
			resource.TestCheckResourceAttr(resourceName, "username", opts.Username),
		}

		// todo add check for interfaces/volumes/secgroups
		for i, md := range opts.Metadata.Metadata {
			checksStore = append(checksStore,
				resource.TestCheckResourceAttr(resourceName, fmt.Sprintf(`metadata.%d.key`, i), md.Key),
				resource.TestCheckResourceAttr(resourceName, fmt.Sprintf(`metadata.%d.value`, i), md.Value),
			)
		}

		for i, cfg := range opts.Configuration.Metadata {
			checksStore = append(checksStore,
				resource.TestCheckResourceAttr(resourceName, fmt.Sprintf(`configuration.%d.key`, i), cfg.Key),
				resource.TestCheckResourceAttr(resourceName, fmt.Sprintf(`configuration.%d.value`, i), cfg.Value),
			)
		}

		return resource.ComposeTestCheckFunc(checksStore...)(s)
	}
}
