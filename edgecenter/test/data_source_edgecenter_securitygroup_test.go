//go:build cloud_data_source

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/securitygroups"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func TestAccSecurityGroupDataSource(t *testing.T) {
	t.Parallel()
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createTestClient(cfg.Provider, edgecenter.SecurityGroupPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts1 := securitygroups.CreateOpts{
		SecurityGroup: securitygroups.CreateSecurityGroupOpts{
			Name:               "test-sg1",
			SecurityGroupRules: []securitygroups.CreateSecurityGroupRuleOpts{},
			Metadata:           map[string]interface{}{"key1": "val1", "key2": "val2"},
		},
	}

	sg1, err := securitygroups.Create(client, opts1).Extract()
	if err != nil {
		t.Fatal(err)
	}

	opts2 := securitygroups.CreateOpts{
		SecurityGroup: securitygroups.CreateSecurityGroupOpts{
			Name:               "test-sg2",
			SecurityGroupRules: []securitygroups.CreateSecurityGroupRuleOpts{},
			Metadata:           map[string]interface{}{"key1": "val1", "key3": "val3"},
		},
	}

	sg2, err := securitygroups.Create(client, opts2).Extract()
	if err != nil {
		t.Fatal(err)
	}
	defer securitygroups.Delete(client, sg1.ID)
	defer securitygroups.Delete(client, sg2.ID)

	resourceName := "data.edgecenter_securitygroup.acctest"

	tpl1 := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_securitygroup" "acctest" {
			  %s
              %s
	          name = "%s"
			  metadata_k="key1"
			}
		`, projectInfo(), regionInfo(), name)
	}

	tpl2 := func(name string) string {
		return fmt.Sprintf(`
			data "edgecenter_securitygroup" "acctest" {
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
				Config: tpl1(sg1.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", sg1.Name),
					resource.TestCheckResourceAttr(resourceName, "id", sg1.ID),
					testAccCheckMetadata(t, resourceName, true, map[string]interface{}{
						"key1": "val1", "key2": "val2",
					}),
				),
			},
			{
				Config: tpl2(sg2.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", sg2.Name),
					resource.TestCheckResourceAttr(resourceName, "id", sg2.ID),
					testAccCheckMetadata(t, resourceName, true, map[string]interface{}{
						"key3": "val3",
					}),
				),
			},
		},
	})
}
