package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
				Type:         schema.TypeString,
				Description:  "The name of the server group. Either 'id' or 'name' must be specified.",
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"id", "name"},
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The ID of the server group. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
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

func dataSourceServerGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ServerGroup reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	serverGroup, err := getServerGroup(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverGroup.ID)
	_ = d.Set("name", serverGroup.Name)
	_ = d.Set("id", serverGroup.ID)
	_ = d.Set("project_id", serverGroup.ProjectID)
	_ = d.Set("region_id", serverGroup.RegionID)
	_ = d.Set("policy", serverGroup.Policy)

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
