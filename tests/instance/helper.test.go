package test

import (
	"github.com/gruntwork-io/terratest/modules/terraform"
	"os"
)

var (
	apiToken          = os.Getenv("EC_API_TOKEN")
	edgecenterProject = os.Getenv("PROJECT_ID")
	edgecenterRegion  = os.Getenv("TEST_REGION_ID")
)

const (
	imageName       = "ubuntu-23.04-x64"
	networkName     = "terratest-network"
	subnetName      = "terratest-subnet"
	instanceName    = "terratest-instance"
	instanceFlavor  = "g1-standard-1-2"
	instanceVmState = "active"
	keypairName     = "terratest-keypair"
	serverGroupName = "terratest-servergroup"
	userData        = "Ym9vdGNtZDoKICAtIGVjaG8gMTkyLjE2OC4xLjEzMCB1cy5hcmNoaXZlLnVidW50dS5jb20gPj4gL2V0Yy9ob3N0cwogIC0gWyBjbG91ZC1pbml0LXBlciwgb25jZSwgbXlta2ZzLCBta2ZzLCAvZGV2L3ZkYiBd"
)

// Функции инициализации для модулей
func initializeNetworkModule(options *terraform.Options) {
	options.TerraformDir = "../modules/network"
	options.Vars = map[string]interface{}{
		"network_name":        networkName,
		"subnet_name":         subnetName,
		"region_id":           edgecenterRegion,
		"project_id":          edgecenterProject,
		"permanent_api_token": apiToken,
	}
}

func initializeVolumeModule(options *terraform.Options) {
	options.TerraformDir = "../modules/volumes"
	options.Vars = map[string]interface{}{
		"region_id":           edgecenterRegion,
		"project_id":          edgecenterProject,
		"permanent_api_token": apiToken,
	}
}

func initializeInstanceModule(options *terraform.Options, networkID, subnetID, firstVolumeID, secondVolumeID, thirdVolumeID, serverGroupID string) {
	options.TerraformDir = "../modules/instance"
	options.Vars = map[string]interface{}{
		"instance_name":       instanceName,
		"flavor_id":           instanceFlavor,
		"region_id":           edgecenterRegion,
		"project_id":          edgecenterProject,
		"keypair_name":        keypairName,
		"server_group":        serverGroupID,
		"user_data":           userData,
		"image_name":          imageName,
		"permanent_api_token": apiToken,
		"instance_volumes": []map[string]interface{}{
			{
				"volume_id":             firstVolumeID,
				"boot_index":            0,
				"delete_on_termination": true,
			},
			{
				"volume_id":             secondVolumeID,
				"boot_index":            1,
				"delete_on_termination": false,
			},
			{
				"volume_id":             thirdVolumeID,
				"boot_index":            2,
				"delete_on_termination": true,
			},
		},
		"instance_interfaces": []map[string]interface{}{
			{
				"type":                   "subnet",
				"network_id":             networkID,
				"subnet_id":              subnetID,
				"port_security_disabled": false,
			},
		},
		"metadata_map": map[string]string{
			"type":            "magic_carpet",
			"unicorn_access":  "true",
			"dragon_firewall": "very-hot",
			"enchanted_speed": "lightning-fast",
			"fairy_lights":    "5",
		},
	}
}

func initializeKeypairModule(options *terraform.Options) {
	options.TerraformDir = "../modules/keypair"
	options.Vars = map[string]interface{}{
		"project_id":          edgecenterProject,
		"keypair_name":        keypairName,
		"permanent_api_token": apiToken,
	}
}

func initializeServerGroupModule(options *terraform.Options) {
	options.TerraformDir = "../modules/server_group"
	options.Vars = map[string]interface{}{
		"region_id":           edgecenterRegion,
		"project_id":          edgecenterProject,
		"servergroup_name":    serverGroupName,
		"permanent_api_token": apiToken,
	}
}

func initializeFloatingIPModule(options *terraform.Options) {
	options.TerraformDir = "../modules/floating_ip"
	options.Vars = map[string]interface{}{
		"region_id":           edgecenterRegion,
		"project_id":          edgecenterProject,
		"permanent_api_token": apiToken,
	}
}

func initializeSecGroupModule(options *terraform.Options) {
	options.TerraformDir = "../modules/security_group"
	options.Vars = map[string]interface{}{
		"region_id":           edgecenterRegion,
		"project_id":          edgecenterProject,
		"permanent_api_token": apiToken,
		"security_groups": map[string]interface{}{
			"secgroup1": map[string]interface{}{
				"name": "terratest-security_group",
				"security_group_rules": []map[string]interface{}{
					{
						"direction":      "ingress",
						"ethertype":      "IPv4",
						"protocol":       "tcp",
						"port_range_min": 19990, // Значение без кавычек как число
						"port_range_max": 19990, // Значение без кавычек как число
					},
					{
						"direction":      "egress",
						"ethertype":      "IPv4",
						"protocol":       "tcp",
						"port_range_min": 19990, // Значение без кавычек как число
						"port_range_max": 19990, // Значение без кавычек как число
					},
				},
			},
		},
	}
}

func initializeReservedFIPModule(options *terraform.Options, networkID, subnetID string) {
	options.TerraformDir = "../modules/reserved_fixedip"
	options.Vars = map[string]interface{}{
		"region_id":           edgecenterRegion,
		"project_id":          edgecenterProject,
		"permanent_api_token": apiToken,
		"reserved_fixed_ips": map[string]map[string]interface{}{
			"ip1": {
				"type":      "subnet",
				"subnet_id": subnetID,
			},
			"ip2": {
				"type": "external",
			},
			"ip3": {
				"type":       "any_subnet",
				"network_id": networkID,
			},
			"ip4": {
				"type":             "ip_address",
				"fixed_ip_address": "192.168.10.50",
				"network_id":       networkID,
			},
		},
	}
}
