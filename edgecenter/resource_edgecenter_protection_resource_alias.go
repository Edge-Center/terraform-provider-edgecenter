package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	protectionSDK "github.com/phkrl/edgecenterprotection-go"
)

func resourceProtectionResourceAlias() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProtectionResourceAliasCreate,
		ReadContext:   resourceProtectionResourceAliasRead,
		DeleteContext: resourceProtectionResourceAliasDelete,
		Description:   "Allows to manage aliases for DDoS protection resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of alias of DDoS protection resource. Must be a sub-domain of resource.",
			},
			"resource": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of DDoS protection resource to manage alias for.",
			},
		},
	}
}

func resourceProtectionResourceAliasCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	resourceID := d.Get("resource").(string)
	log.Printf("[DEBUG] Start creating alias for DDoS protection resource %s", resourceID)
	config := m.(*Config)
	client := config.ProtectionClient

	id, err := strconv.ParseInt(resourceID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	var req protectionSDK.AliasCreateRequest

	req.Name = d.Get("name").(string)

	result, _, err := client.Aliases.Create(ctx, id, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s:%d", resourceID, result.ID))
	resourceProtectionResourceAliasRead(ctx, d, m)

	log.Printf("[DEBUG] Finish creating alias for DDoS protection resource (id=%d:%d)\n", resourceID, result.ID)

	return nil
}

func resourceProtectionResourceAliasRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start reading alias for DDoS protection resource (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	aliasID, err := strconv.ParseInt(aID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	result, _, err := client.Aliases.Get(ctx, resourceID, aliasID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", result.Name)
	d.Set("resource", fmt.Sprintf("%d", resourceID))

	log.Println("[DEBUG] Finish reading alias for DDoS")

	return nil
}

func resourceProtectionResourceAliasDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	rID, aID, err := ImportStringParserSimple(d.Id())
	log.Printf("[DEBUG] Start updating alias for DDoS protection resource (id=%s)\n", d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resourceID, err := strconv.ParseInt(rID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	aliasID, err := strconv.ParseInt(aID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	config := m.(*Config)
	client := config.ProtectionClient

	if _, err := client.Aliases.Delete(ctx, resourceID, aliasID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish Protection Resource Alias deleting")

	return nil
}
