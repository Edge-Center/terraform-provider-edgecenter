package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSecret() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSecretRead,
		Description: "Represent secret",
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
				Optional:     true,
				Computed:     true,
				Description:  "The name of the secret. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The ID of the secret. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
			},
			"algorithm": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The encryption algorithm used for the secret.",
			},
			"bit_length": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The bit length of the encryption algorithm.",
			},
			"mode": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The mode of the encryption algorithm.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the secret.",
			},
			"content_types": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "The content types associated with the secret's payload.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"expiration": {
				Type:        schema.TypeString,
				Description: "Datetime when the secret will expire. The format is 2025-12-28T19:14:44.180394",
				Computed:    true,
			},
			"created": {
				Type:        schema.TypeString,
				Description: "Datetime when the secret was created. The format is 2025-12-28T19:14:44.180394",
				Computed:    true,
			},
		},
	}
}

func dataSourceSecretRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start secret reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	secret, err := getSecret(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(secret.ID)
	_ = d.Set("name", secret.Name)
	_ = d.Set("id", secret.ID)
	_ = d.Set("algorithm", secret.Algorithm)
	_ = d.Set("bit_length", secret.BitLength)
	_ = d.Set("mode", secret.Mode)
	_ = d.Set("status", secret.Status)
	_ = d.Set("expiration", secret.Expiration)
	_ = d.Set("created", secret.Created)

	if err := d.Set("content_types", secret.ContentTypes); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish secret reading")

	return nil
}
