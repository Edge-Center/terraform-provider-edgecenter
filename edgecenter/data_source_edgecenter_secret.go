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
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the secret.",
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
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	secretID := d.Id()
	log.Printf("[DEBUG] Secret id = %s", secretID)

	allSecrets, _, err := clientV2.Secrets.List(ctx)
	if err != nil {
		return diag.Errorf("cannot get secrets. Error: %s", err.Error())
	}

	var found bool
	name := d.Get("name").(string)
	for _, secret := range allSecrets {
		if name == secret.Name {
			d.SetId(secret.ID)
			d.Set("name", name)
			d.Set("algorithm", secret.Algorithm)
			d.Set("bit_length", secret.BitLength)
			d.Set("mode", secret.Mode)
			d.Set("status", secret.Status)
			d.Set("expiration", secret.Expiration)
			d.Set("created", secret.Created)

			if err := d.Set("content_types", secret.ContentTypes); err != nil {
				return diag.FromErr(err)
			}
			found = true

			break
		}
	}

	if !found {
		return diag.Errorf("secret with name %s does not exit", name)
	}

	log.Println("[DEBUG] Finish secret reading")

	return diags
}
