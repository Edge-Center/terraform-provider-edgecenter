package floatingip

import (
	"context"
	"log"
	"net"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/converter"
)

func ResourceEdgeCenterFloatingIP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEdgeCenterFloatingIPCreate,
		ReadContext:   resourceEdgeCenterFloatingIPRead,
		UpdateContext: resourceEdgeCenterFloatingIPUpdate,
		DeleteContext: resourceEdgeCenterFloatingIPDelete,
		Description: `A floating IP is a static IP address that can be associated with one of your instances or loadbalancers, 
allowing it to have a static public IP address. The floating IP can be re-associated to any other instance in the same datacenter.`,
		Schema: floatingIPSchema(),
	}
}

func resourceEdgeCenterFloatingIPCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	opts := &edgecloud.FloatingIPCreateRequest{}

	if v, ok := d.GetOk("port_id"); ok {
		opts.PortID = v.(string)
	}

	if v, ok := d.GetOk("fixed_ip_address"); ok {
		opts.FixedIPAddress = net.ParseIP(v.(string))
	}

	if v, ok := d.GetOk("metadata"); ok {
		metadata := converter.MapInterfaceToMapString(v.(map[string]interface{}))
		opts.Metadata = metadata
	}

	log.Printf("[DEBUG] Floating IP create configuration: %#v", opts)

	taskResult, err := util.ExecuteAndExtractTaskResult(ctx, client.Floatingips.Create, opts, client)
	if err != nil {
		return diag.Errorf("error creating floating IP: %s", err)
	}

	d.SetId(taskResult.FloatingIPs[0])

	log.Printf("[INFO] Floating IP: %s", d.Id())

	return resourceEdgeCenterFloatingIPRead(ctx, d, meta)
}

func resourceEdgeCenterFloatingIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	// Retrieve the floating ip properties for updating the state
	floatingIP, resp, err := client.Floatingips.Get(ctx, d.Id())
	if err != nil {
		// check if the floating ip no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] EdgeCenter FloatingIP (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return diag.Errorf("Error retrieving floating ip: %s", err)
	}

	d.Set("floating_ip_address", floatingIP.FloatingIPAddress)
	d.Set("status", floatingIP.Status)
	d.Set("router_id", floatingIP.RouterID)
	d.Set("subnet_id", floatingIP.SubnetID)
	d.Set("region", floatingIP.Region)

	return nil
}

func resourceEdgeCenterFloatingIPUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	if d.HasChanges("fixed_ip_address", "port_id") {
		oldFixedIPAddress, newFixedIPAddress := d.GetChange("fixed_ip_address")
		oldPortID, newPortID := d.GetChange("port_id")
		if oldPortID.(string) != "" || oldFixedIPAddress.(string) != "" {
			_, _, err := client.Floatingips.UnAssign(ctx, d.Id())
			if err != nil {
				return diag.FromErr(err)
			}
		}

		if newPortID.(string) != "" || newFixedIPAddress.(string) != "" {
			assignFloatingIPRequest := &edgecloud.AssignFloatingIPRequest{
				PortID:         newPortID.(string),
				FixedIPAddress: net.ParseIP(newFixedIPAddress.(string)),
			}

			if _, _, err := client.Floatingips.Assign(ctx, d.Id(), assignFloatingIPRequest); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("metadata") {
		metadata := edgecloud.Metadata(converter.MapInterfaceToMapString(d.Get("metadata").(map[string]interface{})))

		if _, err := client.Floatingips.MetadataUpdate(ctx, d.Id(), &metadata); err != nil {
			return diag.Errorf("cannot update metadata. Error: %s", err)
		}
	}

	return resourceEdgeCenterFloatingIPRead(ctx, d, meta)
}

func resourceEdgeCenterFloatingIPDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	log.Printf("[INFO] Deleting floating ip: %s", d.Id())
	if err := util.DeleteResourceIfExist(ctx, client, client.Floatingips, d.Id()); err != nil {
		return diag.Errorf("Error deleting firewall: %s", err)
	}
	d.SetId("")

	return nil
}
