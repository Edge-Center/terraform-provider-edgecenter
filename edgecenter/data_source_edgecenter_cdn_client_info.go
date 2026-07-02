package edgecenter

import (
	"context"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCDNClientInfo() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCDNClientInfoRead,
		Description: "CDN client info, including CNAME target for DNS configuration.",
		Schema: map[string]*schema.Schema{
			"cname": {
				Type:        schema.TypeString,
				Description: "CNAME target (e.g. cl-XXXXX.edgecdn.ru)",
				Computed:    true,
			},
			"client_id": {
				Type:        schema.TypeInt,
				Description: "CDN client ID",
				Computed:    true,
			},
		},
	}
}

func dataSourceCDNClientInfoRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start reading CDN client info.")

	config := m.(*Config)
	client := config.CDNClient

	info, err := client.Tools().ClientInfo(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	if info.Cname == "" {
		return diag.Errorf("CDN client CNAME target is empty; check your CDN account configuration")
	}

	d.SetId(strconv.FormatInt(info.ID, 10))
	err = d.Set("cname", info.Cname)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("client_id", info.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish reading CDN client info.")

	return nil
}
