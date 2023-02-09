//go:build cloud

package edgecenter_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/network/v1/networks"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/subnet/v1/subnets"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	subnetTestName = "test-subnet"
	cidr           = "192.168.42.0/24"

	subnet1TestName = "test-subnet1"
	cidr1           = "192.168.41.0/24"

	subnet2TestName = "test-subnet2"
	cidr2           = "192.168.43.0/24"
)

func TestAccSubnetDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	clientNet, err := CreateTestClient(cfg.Provider, edgecenter.NetworksPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	clientSubnet, err := CreateTestClient(cfg.Provider, edgecenter.SubnetPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := networks.CreateOpts{
		Name: networkTestName,
	}

	networkID, err := createTestNetwork(clientNet, opts)
	if err != nil {
		t.Fatal(err)
	}

	defer deleteTestNetwork(clientNet, networkID)

	optsSubnet1 := subnets.CreateOpts{
		Name:      subnet1TestName,
		NetworkID: networkID,
		Metadata:  map[string]string{"key1": "val1", "key2": "val2"},
	}

	subnet1ID, err := CreateTestSubnet(clientSubnet, optsSubnet1, cidr1)
	if err != nil {
		t.Fatal(err)
	}

	optsSubnet2 := subnets.CreateOpts{
		Name:      subnet2TestName,
		NetworkID: networkID,
		Metadata:  map[string]string{"key1": "val1", "key3": "val3"},
	}

	subnet2ID, err := CreateTestSubnet(clientSubnet, optsSubnet2, cidr2)
	if err != nil {
		t.Fatal(err)
	}

	fullName := "data.edgecenter_subnet.acctest"
	tpl1 := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_subnet" "acctest" {
  			%s
			%s
			name = "%s"
			metadata_k="key1"
			}
		`, projectInfo(), regionInfo(), name)
	}

	tpl2 := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_subnet" "acctest" {
			%s
			%s
 			name = "%s"
				metadata_kv={
					key3 = "val3"
			  	}
			}
		`, projectInfo(), regionInfo(), name)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl1(optsSubnet1.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", optsSubnet1.Name),
					resource.TestCheckResourceAttr(fullName, "id", subnet1ID),
					resource.TestCheckResourceAttr(fullName, "network_id", networkID),
					edgecenter.TestAccCheckMetadata(fullName, true, map[string]string{
						"key1": "val1", "key2": "val2",
					}),
				),
			},
			{
				Config: tpl2(optsSubnet2.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", optsSubnet2.Name),
					resource.TestCheckResourceAttr(fullName, "id", subnet2ID),
					// resource.TestCheckResourceAttr(fullName, "network_id", networkID),
					edgecenter.TestAccCheckMetadata(fullName, true, map[string]string{
						"key3": "val3",
					}),
				),
			},
		},
	})
}

func CreateTestSubnet(client *edgecloud.ServiceClient, opts subnets.CreateOpts, extra ...string) (string, error) {
	subCidr := cidr
	if extra != nil {
		subCidr = extra[0]
	}

	var eccidr edgecloud.CIDR
	_, netIPNet, err := net.ParseCIDR(subCidr)
	if err != nil {
		return "", err
	}
	eccidr.IP = netIPNet.IP
	eccidr.Mask = netIPNet.Mask
	opts.CIDR = eccidr

	res, err := subnets.Create(client, opts).Extract()
	if err != nil {
		return "", err
	}

	taskID := res.Tasks[0]
	subnetID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, edgecenter.SubnetCreatingTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		Subnet, err := subnets.ExtractSubnetIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve Subnet ID from task info: %w", err)
		}
		return Subnet, nil
	},
	)

	return subnetID.(string), err
}

func deleteTestSubnet(client *edgecloud.ServiceClient, subnetID string) error {
	results, err := subnets.Delete(client, subnetID).Extract()
	if err != nil {
		return err
	}
	taskID := results.Tasks[0]
	_, err = tasks.WaitTaskAndReturnResult(client, taskID, true, edgecenter.SubnetDeleting, func(task tasks.TaskID) (interface{}, error) {
		_, err := subnets.Get(client, subnetID).Extract()
		if err == nil {
			return nil, fmt.Errorf("cannot delete subnet with ID: %s", subnetID)
		}
		switch err.(type) {
		case edgecloud.ErrDefault404:
			return nil, nil
		default:
			return nil, err
		}
	})
	return err
}
