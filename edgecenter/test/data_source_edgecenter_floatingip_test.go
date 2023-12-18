//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/floatingip/v1/floatingips"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/task/v1/tasks"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccFloatingIPDataSource(t *testing.T) {
	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.FloatingIPsPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts := floatingips.CreateOpts{}

	res, err := floatingips.Create(client, opts).Extract()
	if err != nil {
		t.Fatal(err)
	}

	taskID := res.Tasks[0]
	floatingIPID, err := tasks.WaitTaskAndReturnResult(client, taskID, true, edgecenter.FloatingIPCreateTimeout, func(task tasks.TaskID) (interface{}, error) {
		taskInfo, err := tasks.Get(client, string(task)).Extract()
		if err != nil {
			return nil, fmt.Errorf("cannot get task with ID: %s. Error: %w", task, err)
		}
		floatingIPID, err := floatingips.ExtractFloatingIPIDFromTask(taskInfo)
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve FloatingIP ID from task info: %w", err)
		}
		return floatingIPID, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	defer floatingips.Delete(client, floatingIPID.(string))

	fip, err := floatingips.Get(client, floatingIPID.(string)).Extract()
	if err != nil {
		t.Fatal(err)
	}

	resourceName := "data.edgecenter_floatingip.acctest"
	tpl := func(ip string) string {
		return fmt.Sprintf(`
			data "edgecenter_floatingip" "acctest" {
			  %s
              %s
              floating_ip_address = "%s"
			}
		`, projectInfo(), regionInfo(), ip)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: tpl(fip.FloatingIPAddress.String()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", floatingIPID.(string)),
					resource.TestCheckResourceAttr(resourceName, "floating_ip_address", fip.FloatingIPAddress.String()),
				),
			},
		},
	})
}
