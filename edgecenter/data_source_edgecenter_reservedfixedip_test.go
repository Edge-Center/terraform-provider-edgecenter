//go:build cloud
// +build cloud

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/reservedfixedip/v1/reservedfixedips"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
)

func TestAccReservedFixedIPDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := CreateTestClient(cfg.Provider, reservedFixedIPsPoint, versionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := reservedfixedips.CreateOpts{
		Type: reservedfixedips.External,
	}

	res, err := reservedfixedips.Create(client, opts).Extract()
	if err != nil {
		t.Fatal(err)
	}

	taskID := res.Tasks[0]
	reservedFixedIPID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, ReservedFixedIPCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		reservedFixedIPID, err := reservedfixedips.ExtractReservedFixedIPIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve reservedFixedIP ID from task info: %w", err)
		}
		return reservedFixedIPID, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	defer reservedfixedips.Delete(client, reservedFixedIPID.(string))

	fip, err := reservedfixedips.Get(client, reservedFixedIPID.(string)).Extract()
	if err != nil {
		t.Fatal(err)
	}

	fullName := "data.edgecenter_reservedfixedip.acctest"
	tpl := func(ip string) string {
		return fmt.Sprintf(`
			data "edgecenter_reservedfixedip" "acctest" {
			  %s
              %s
              fixed_ip_address = "%s"
			}
		`, projectInfo(), regionInfo(), ip)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(fip.FixedIPAddress.String()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "id", reservedFixedIPID.(string)),
					resource.TestCheckResourceAttr(fullName, "fixed_ip_address", fip.FixedIPAddress.String()),
				),
			},
		},
	})
}
