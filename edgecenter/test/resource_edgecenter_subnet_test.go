//go:build cloud_resource

package edgecenter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type testSubnetParams struct {
	NetworkName            string
	Name                   string
	CIDR                   string
	DNSNameservers         []string
	HostRoutes             []map[string]string
	GatewayIP              string
	MetadataMap            map[string]string
	EnableDHCP             bool
	ConnectToNetworkRouter bool
}

func (t testSubnetParams) DNSNameserversString() string {
	buf := new(bytes.Buffer)

	json.NewEncoder(buf).Encode(t.DNSNameservers)

	return buf.String()
}

func (t testSubnetParams) HostRoutesString() string {
	result := ""

	template := `host_routes {
					destination = "%[1]s"
					nexthop     = "%[2]s"
				}`

	for _, route := range t.HostRoutes {
		result += fmt.Sprintf(template, route["destination"], route["nexthop"])
		result += "\n"
	}

	return result
}

func (t testSubnetParams) MetadataString() string {
	result := "metadata_map = {\n"

	for k, v := range t.MetadataMap {
		result += fmt.Sprintf("%s = \"%s\"\n", k, v)
	}

	result += "}\n"

	return result
}

func TestAccSubnet(t *testing.T) {
	t.Parallel()

	checkMetadataKey := "test_subnet"

	create := testSubnetParams{
		NetworkName:    "network_for_test_subnetwork",
		Name:           "create_subnet",
		CIDR:           "192.168.10.0/24",
		DNSNameservers: []string{"8.8.4.4", "1.1.1.1"},
		HostRoutes: []map[string]string{
			{"destination": "10.0.3.0/24", "nexthop": "192.168.10.1"},
			{"destination": "10.0.4.0/24", "nexthop": "192.168.10.1"},
		},
		GatewayIP:              "192.168.10.1",
		ConnectToNetworkRouter: true,
		EnableDHCP:             true,
		MetadataMap: map[string]string{
			checkMetadataKey: "val0",
			"key1":           "val1",
			"key2":           "val2",
		},
	}

	update := testSubnetParams{
		NetworkName:            "network_for_test_subnetwork",
		Name:                   "update_subnet",
		CIDR:                   "192.168.10.0/24",
		DNSNameservers:         []string{},
		HostRoutes:             []map[string]string{},
		GatewayIP:              "disable",
		ConnectToNetworkRouter: false,
		EnableDHCP:             false,
		MetadataMap: map[string]string{
			checkMetadataKey: "val0",
			"key3":           "val3",
		},
	}

	SubnetTemplate := func(params *testSubnetParams) string {
		templateNetwork := fmt.Sprintf(`
		resource "edgecenter_network" "network" {
			name       = "%[1]s"
			type       = "vxlan"
			
			// region
            %[2]s
			
			// project
			%[3]s
		}`, params.NetworkName, regionInfo(), projectInfo())

		templateSubnet := fmt.Sprintf(`
		resource "edgecenter_subnet" "acctest" {
			name            = "%[1]s"
			cidr            = "%[2]s"
			network_id      = edgecenter_network.network.id
			dns_nameservers = %[3]s
		
			enable_dhcp = %[4]t
			connect_to_network_router = %[5]t
		
			// host_routes
			%[6]s
		
			gateway_ip = "%[7]s"
			
			// metadata
			%[8]s

			// region
            %[9]s
			
			// project
			%[10]s
		}
		`,
			params.Name,
			params.CIDR,
			params.DNSNameserversString(),
			params.EnableDHCP,
			params.ConnectToNetworkRouter,
			params.HostRoutesString(),
			params.GatewayIP,
			params.MetadataString(),
			regionInfo(),
			projectInfo(),
		)

		template := templateNetwork + templateSubnet + "\n"

		return template
	}

	resourceName := "edgecenter_subnet.acctest"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccSubnetDestroy(checkMetadataKey),
		Steps: []resource.TestStep{
			{
				Config: SubnetTemplate(&create),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					checkSubnetAttrs(resourceName, create),
					testAccCheckMetadata(t, resourceName, true, create.MetadataMap),
				),
			},
			{
				Config: SubnetTemplate(&update),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					checkSubnetAttrs(resourceName, update),
					testAccCheckMetadata(t, resourceName, true, update.MetadataMap),
				),
			},
		},
	})
}

func checkSubnetAttrs(resourceName string, opts testSubnetParams) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if s.Empty() == true {
			return fmt.Errorf("State not updated")
		}

		checksStore := []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "name", opts.Name),
			resource.TestCheckResourceAttr(resourceName, "cidr", opts.CIDR),
			resource.TestCheckResourceAttr(resourceName, "enable_dhcp", strconv.FormatBool(opts.EnableDHCP)),
			resource.TestCheckResourceAttr(resourceName, "dns_nameservers.#", strconv.Itoa(len(opts.DNSNameservers))),
			resource.TestCheckResourceAttr(resourceName, "host_routes.#", strconv.Itoa(len(opts.HostRoutes))),
			resource.TestCheckResourceAttr(resourceName, "gateway_ip", opts.GatewayIP),
		}

		for i, hr := range opts.HostRoutes {
			checksStore = append(checksStore,
				resource.TestCheckResourceAttr(resourceName, fmt.Sprintf(`host_routes.%d.destination`, i), hr["destination"]),
				resource.TestCheckResourceAttr(resourceName, fmt.Sprintf(`host_routes.%d.nexthop`, i), hr["nexthop"]),
			)
		}

		return resource.ComposeTestCheckFunc(checksStore...)(s)
	}
}

func testAccSubnetDestroy(metadataK string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := testAccProvider.Meta().(*edgecenter.Config)
		clientV2, err := config.NewCloudClient()
		if err != nil {
			return err
		}

		clientV2.Region, clientV2.Project, err = getRegionIDAndProjectID()
		if err != nil {
			return err
		}

		subs, _, err := clientV2.Subnetworks.List(context.Background(), &edgecloudV2.SubnetworkListOptions{MetadataK: metadataK})
		if err != nil {
			return fmt.Errorf("subnetworks.List error: %w", err)
		}

		if len(subs) != 0 {
			return fmt.Errorf("subnetworks still exist")
		}

		return nil
	}
}
