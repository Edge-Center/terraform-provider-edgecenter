package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceLBListener() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceLBListenerRead,
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
				Required:    true,
				Description: "The name of the load balancer listener.",
			},
			"loadbalancer_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The uuid for the load balancer.",
			},
			"protocol": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Available values is 'HTTP', 'HTTPS', 'TCP', 'UDP'",
			},
			"protocol_port": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The port on which the protocol is bound.",
			},
			"l7policies": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Set of l7policy uuids attached to this listener.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"pool_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of pools associated with the load balancer.",
			},
			"operating_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current operational status of the load balancer.",
			},
			"provisioning_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current provisioning status of the load balancer.",
			},
			"allowed_cidrs": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "The allowed CIDRs for listener.",
			},
			"timeout_client_data": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timeout for the frontend client inactivity (in milliseconds).",
			},
			"timeout_member_data": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timeout for the backend member inactivity (in milliseconds).",
			},
			"timeout_member_connect": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timeout for the backend member connection (in milliseconds).",
			},
		},
	}
}

func dataSourceLBListenerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBListener reading")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	var opts edgecloudV2.ListenerListOptions
	name := d.Get("name").(string)
	lbID := d.Get("loadbalancer_id").(string)
	if lbID != "" {
		opts.LoadbalancerID = lbID
	}

	ls, _, err := clientV2.Loadbalancers.ListenerList(ctx, &opts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var listener edgecloudV2.Listener
	for _, l := range ls {
		if l.Name == name {
			listener = l
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("lb listener with name %s not found", name)
	}

	d.SetId(listener.ID)
	d.Set("name", listener.Name)
	d.Set("protocol", listener.Protocol)
	d.Set("protocol_port", listener.ProtocolPort)
	d.Set("pool_count", listener.PoolCount)
	d.Set("operating_status", listener.OperatingStatus)
	d.Set("provisioning_status", listener.ProvisioningStatus)
	d.Set("loadbalancer_id", lbID)
	d.Set("project_id", d.Get("project_id").(int))
	d.Set("region_id", d.Get("region_id").(int))
	d.Set("allowed_cidrs", listener.AllowedCIDRs)
	d.Set("timeout_member_data", listener.TimeoutMemberData)
	d.Set("timeout_client_data", listener.TimeoutClientData)
	d.Set("timeout_member_connect", listener.TimeoutMemberConnect)

	l7Policies, err := GetListenerL7PolicyUUIDS(ctx, clientV2, listener.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("l7policies", l7Policies)

	log.Println("[DEBUG] Finish LBListener reading")

	return diags
}
