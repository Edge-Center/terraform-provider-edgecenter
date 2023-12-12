package lblistener

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
)

func DataSourceEdgeCenterLbListener() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceEdgeCenterLbListenerRead,
		Description: `A listener is a process that checks for connection requests using the protocol and port that you configure.`,

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the project",
			},
			"region_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the region",
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "listener uuid",
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{"id", "name"},
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Description: `listener name. this parameter is not unique, if there is more than one listener with the same name, 
then the first one will be used. it is recommended to use "id"`,
				ExactlyOneOf: []string{"id", "name"},
			},
			"loadbalancer_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the load balancer",
			},
			// computed attributes
			"protocol": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "protocol of the load balancer",
			},
			"protocol_port": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "protocol port number of the resource",
			},
			"secret_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the secret where PKCS12 file is stored for the TERMINATED_HTTPS load balancer",
			},
			"provisioning_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "lifecycle status of the listener",
			},
			"operating_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "operating status of the listener",
			},
			"pool_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "number of pools",
			},
			"insert_headers": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "dictionary of additional header insertion into the HTTP headers. only used with the HTTP and TERMINATED_HTTPS protocols",
			},
			"allowed_cidrs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "allowed CIDRs for listener.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceEdgeCenterLbListenerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	loadbalancerID := d.Get("loadbalancer_id").(string)

	var foundListener *edgecloud.Listener

	if id, ok := d.GetOk("id"); ok {
		listener, _, err := client.Loadbalancers.ListenerGet(ctx, id.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundListener = listener
	} else if listenerName, ok := d.GetOk("name"); ok {
		listener, err := util.LBListenerGetByName(ctx, client, listenerName.(string), loadbalancerID)
		if err != nil {
			return diag.FromErr(err)
		}

		foundListener = listener
	} else {
		return diag.Errorf("Error: specify either id or a name to lookup the listener")
	}

	d.SetId(foundListener.ID)
	d.Set("name", foundListener.Name)
	d.Set("loadbalancer_id", loadbalancerID)
	d.Set("provisioning_status", foundListener.ProvisioningStatus)
	d.Set("operating_status", foundListener.OperatingStatus)
	d.Set("protocol", foundListener.Protocol)
	d.Set("protocol_port", foundListener.ProtocolPort)
	d.Set("pool_count", foundListener.PoolCount)
	d.Set("secret_id", foundListener.SecretID)

	if err := setAllowedCIDRs(ctx, d, foundListener); err != nil {
		return diag.FromErr(err)
	}

	if err := setInsertHeaders(ctx, d, foundListener); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
