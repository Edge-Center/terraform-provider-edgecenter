package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/servergroup/v1/servergroups"
)

func dataSourceServerGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceServerGroupRead,
		Description: "Represent server group data",
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the server group.",
				Required:    true,
			},
			"policy": {
				Type:        schema.TypeString,
				Description: "Server group policy. Available value is 'affinity', 'anti-affinity'",
				Computed:    true,
			},
			"instances": {
				Type:        schema.TypeList,
				Description: "Instances in this server group",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"instance_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceServerGroupRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ServerGroup reading")
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, ServerGroupsPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	var serverGroup servergroups.ServerGroup
	serverGroups, err := servergroups.ListAll(client)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	name := d.Get("name").(string)
	for _, sg := range serverGroups {
		if sg.Name == name {
			serverGroup = sg
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("server group with name %s not found", name)
	}

	d.SetId(serverGroup.ServerGroupID)
	d.Set("name", name)
	d.Set("project_id", serverGroup.ProjectID)
	d.Set("region_id", serverGroup.RegionID)
	d.Set("policy", serverGroup.Policy.String())

	instances := make([]map[string]string, len(serverGroup.Instances))
	for i, instance := range serverGroup.Instances {
		rawInstance := make(map[string]string)
		rawInstance["instance_id"] = instance.InstanceID
		rawInstance["instance_name"] = instance.InstanceName
		instances[i] = rawInstance
	}
	if err := d.Set("instances", instances); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish ServerGroup reading")

	return nil
}
