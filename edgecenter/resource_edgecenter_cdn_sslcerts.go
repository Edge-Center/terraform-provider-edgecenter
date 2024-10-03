package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercdn-go/sslcerts"
)

func resourceCDNCert() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Enter the SSL certificate name. It must be unique.",
			},
			"cert": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				ForceNew:    true,
				Description: "Enter the public part of the SSL certificate. Add all the certificate chains. Each certificate chain should be separated by '\\n'.",
			},
			"private_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				ForceNew:    true,
				Description: "Enter the private key of the SSL certificate. Add all the certificate chains. The private key should be separated by '\\n', as shown in the example. ",
			},
			"has_related_resources": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "If set to \"true\", the SSL certificate is used by a CDN resource.",
			},
			"automated": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Shows how the SSL certificate was added to the account. If set to \"true\", this is an automatically issued Let's Encrypt certificate. If set to \"false\", the SSL certificate was added by a user.",
			},
		},
		CreateContext: resourceCDNCertCreate,
		ReadContext:   resourceCDNCertRead,
		DeleteContext: resourceCDNCertDelete,
	}
}

func resourceCDNCertCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start CDN Cert creating")
	config := m.(*Config)
	client := config.CDNClient

	var req sslcerts.CreateRequest
	req.Name = d.Get("name").(string)
	req.Cert = d.Get("cert").(string)
	req.PrivateKey = d.Get("private_key").(string)

	result, err := client.SSLCerts().Create(ctx, &req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", result.ID))
	resourceCDNCertRead(ctx, d, m)

	log.Printf("[DEBUG] Finish CDN Cert creating (id=%d)\n", result.ID)

	return nil
}

func resourceCDNCertRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	certID := d.Id()
	log.Printf("[DEBUG] Start CDN Cert reading (id=%s)\n", certID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(certID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	result, err := client.SSLCerts().Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("has_related_resources", result.HasRelatedResources)
	d.Set("automated", result.Automated)

	log.Println("[DEBUG] Finish CDN Cert reading")

	return nil
}

func resourceCDNCertDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	certID := d.Id()
	log.Printf("[DEBUG] Start CDN Cert deleting (id=%s)\n", certID)
	config := m.(*Config)
	client := config.CDNClient

	id, err := strconv.ParseInt(certID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.SSLCerts().Delete(ctx, id); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish CDN Cert deleting")

	return nil
}
