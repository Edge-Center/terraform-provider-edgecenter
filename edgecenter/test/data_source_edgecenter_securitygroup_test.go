//go:build cloud

package edgecenter_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/securitygroup/v1/securitygroups"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	securityGroup1TestName = "test-sg1"
	securityGroup2TestName = "test-sg2"
)

func TestAccSecurityGroupDataSource(t *testing.T) {
	cfg, err := createTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	client, err := CreateTestClient(cfg.Provider, edgecenter.SecurityGroupPoint, edgecenter.VersionPointV1)
	if err != nil {
		t.Fatal(err)
	}

	opts1 := securitygroups.CreateOpts{
		SecurityGroup: securitygroups.CreateSecurityGroupOpts{
			Name:               securityGroup1TestName,
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
			Name:               securityGroup2TestName,
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

	fullName := "data.edgecenter_securitygroup.acctest"

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
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", sg1.Name),
					resource.TestCheckResourceAttr(fullName, "id", sg1.ID),
					edgecenter.TestAccCheckMetadata(fullName, true, map[string]interface{}{
						"key1": "val1", "key2": "val2",
					}),
				),
			},
			{
				Config: tpl2(sg2.Name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceExists(fullName),
					resource.TestCheckResourceAttr(fullName, "name", sg2.Name),
					resource.TestCheckResourceAttr(fullName, "id", sg2.ID),
					edgecenter.TestAccCheckMetadata(fullName, true, map[string]interface{}{
						"key3": "val3",
					}),
				),
			},
		},
	})
}