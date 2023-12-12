package loadbalancer

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/converter"
)

func ResourceEdgeCenterLoadbalancer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEdgeCenterLoadbalancerCreate,
		ReadContext:   resourceEdgeCenterLoadbalancerRead,
		UpdateContext: resourceEdgeCenterLoadbalancerUpdate,
		DeleteContext: resourceEdgeCenterLoadbalancerDelete,
		Description: `A loadbalancer is a software service that distributes incoming network traffic 
(e.g., web traffic, application requests) across multiple servers or resources.`,
		Schema: loadbalancerSchema(),
	}
}

func resourceEdgeCenterLoadbalancerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	opts := &edgecloud.LoadbalancerCreateRequest{
		Name:         d.Get("name").(string),
		Flavor:       d.Get("flavor_name").(string),
		VipPortID:    d.Get("vip_port_id").(string),
		VipNetworkID: d.Get("vip_network_id").(string),
		VipSubnetID:  d.Get("vip_subnet_id").(string),
	}

	switch d.Get("floating_ip_source").(string) {
	case "new":
		opts.FloatingIP = &edgecloud.InterfaceFloatingIP{
			Source: edgecloud.NewFloatingIP,
		}
	case "existing":
		opts.FloatingIP = &edgecloud.InterfaceFloatingIP{
			Source:             edgecloud.ExistingFloatingIP,
			ExistingFloatingID: d.Get("floating_ip").(string),
		}
	default:
		opts.FloatingIP = nil
	}

	if v, ok := d.GetOk("metadata"); ok {
		metadata := converter.MapInterfaceToMapString(v.(map[string]interface{}))
		opts.Metadata = metadata
	}

	log.Printf("[DEBUG] Loadbalancer create configuration: %#v", opts)

	taskResult, err := util.ExecuteAndExtractTaskResult(ctx, client.Loadbalancers.Create, opts, client, 2*time.Minute) //nolint: gomnd
	if err != nil {
		return diag.Errorf("error creating loadbalancer: %s", err)
	}

	d.SetId(taskResult.Loadbalancers[0])

	log.Printf("[INFO] Loadbalancer: %s", d.Id())

	return resourceEdgeCenterLoadbalancerRead(ctx, d, meta)
}

func resourceEdgeCenterLoadbalancerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	// Retrieve the loadbalancer properties for updating the state
	loadbalancer, resp, err := client.Loadbalancers.Get(ctx, d.Id())
	if err != nil {
		// check if the loadbalancer no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] EdgeCenter Loadbalancer (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return diag.Errorf("Error retrieving loadbalancer: %s", err)
	}

	d.Set("name", loadbalancer.Name)
	d.Set("region", loadbalancer.Region)
	d.Set("vip_address", loadbalancer.VipAddress.String())
	d.Set("provisioning_status", loadbalancer.ProvisioningStatus)
	d.Set("operating_status", loadbalancer.OperatingStatus)
	d.Set("vip_network_id", loadbalancer.VipNetworkID)
	d.Set("vip_port_id", loadbalancer.VipPortID)

	if len(loadbalancer.FloatingIPs) > 0 {
		d.Set("floating_ip", loadbalancer.FloatingIPs[0].ID)
	}

	if err := setVRRPIPs(ctx, d, loadbalancer); err != nil {
		return diag.FromErr(err)
	}

	if err := setFlavor(ctx, d, loadbalancer); err != nil {
		return diag.FromErr(err)
	}

	// TODO need to add metadataDetailed to Loadbalancers.Get resp

	return nil
}

func resourceEdgeCenterLoadbalancerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		if _, _, err := client.Loadbalancers.Rename(ctx, d.Id(), &edgecloud.Name{Name: newName}); err != nil {
			return diag.Errorf("Error when renaming the loadbalancer: %s", err)
		}
	}

	if d.HasChange("metadata") {
		metadata := edgecloud.Metadata(converter.MapInterfaceToMapString(d.Get("metadata").(map[string]interface{})))

		if _, err := client.Loadbalancers.MetadataUpdate(ctx, d.Id(), &metadata); err != nil {
			return diag.Errorf("cannot update metadata. error: %s", err)
		}
	}

	if d.HasChange("floating_ip") {
		if err := changeFloatingIP(ctx, d, client); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceEdgeCenterLoadbalancerRead(ctx, d, meta)
}

func resourceEdgeCenterLoadbalancerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	log.Printf("[INFO] Deleting loadbalancer: %s", d.Id())
	if err := util.DeleteResourceIfExist(ctx, client, client.Loadbalancers, d.Id()); err != nil {
		return diag.Errorf("Error deleting loadbalancer: %s", err)
	}
	d.SetId("")

	return nil
}
