package edgecenter

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
)

const (
	SecretCreatingTimeout int = 1200
	SecretPoint               = "secrets"
	// RFC3339NoZ is the time format used in Heat (Orchestration).
	RFC3339NoZ          = "2006-01-02T15:04:05"
	RFC3339WithTimeZone = "2006-01-02T15:04:05+00:00"
)

func resourceSecret() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSecretCreate,
		ReadContext:   resourceSecretRead,
		DeleteContext: resourceSecretDelete,
		Description:   "Represent secret",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, secretID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("project_id", projectID)
				d.Set("region_id", regionID)
				d.SetId(secretID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the secret.",
			},
			"private_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "SSL private key in PEM format",
			},
			"certificate_chain": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "SSL certificate chain of intermediates and root certificates in PEM format",
			},
			"certificate": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "SSL certificate in PEM format",
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
				Description: "Datetime when the secret will expire. The format is 2025-12-28T19:14:44",
				Optional:    true,
				Computed:    true,
				StateFunc: func(val interface{}) string {
					expTime, _ := time.Parse(RFC3339NoZ, val.(string))
					return expTime.Format(RFC3339NoZ)
				},
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					rawTime := i.(string)
					_, err := time.Parse(RFC3339NoZ, rawTime)
					if err != nil {
						return diag.FromErr(err)
					}
					return nil
				},
			},
			"created": {
				Type:        schema.TypeString,
				Description: "Datetime when the secret was created. The format is 2025-12-28T19:14:44.180394",
				Computed:    true,
			},
		},
	}
}

func resourceSecretCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Secret creating")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	opts := &edgecloudV2.SecretCreateRequestV2{
		Name: d.Get("name").(string),
		Payload: edgecloudV2.Payload{
			CertificateChain: d.Get("certificate_chain").(string),
			Certificate:      d.Get("certificate").(string),
			PrivateKey:       d.Get("private_key").(string),
		},
	}
	if rawTime := d.Get("expiration").(string); rawTime != "" {
		opts.Expiration = &rawTime
	}

	taskResult, err := utilV2.ExecuteAndExtractTaskResult(ctx, clientV2.Secrets.CreateV2, opts, clientV2)
	if err != nil {
		return diag.FromErr(err)
	}

	secretID := taskResult.Secrets[0]

	log.Printf("[DEBUG] Secret id (%s)", secretID)

	d.SetId(secretID)

	resourceSecretRead(ctx, d, m)

	log.Printf("[DEBUG] Finish Secret creating (%s)", secretID)

	return diags
}

func resourceSecretRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start secret reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	secretID := d.Id()
	log.Printf("[DEBUG] Secret id = %s", secretID)

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	secret, _, err := clientV2.Secrets.Get(ctx, secretID)
	if err != nil {
		return diag.Errorf("cannot get secret with ID: %s. Error: %s", secretID, err.Error())
	}

	expTime, _ := time.Parse(RFC3339WithTimeZone, secret.Expiration)

	d.Set("name", secret.Name)
	d.Set("algorithm", secret.Algorithm)
	d.Set("bit_length", secret.BitLength)
	d.Set("mode", secret.Mode)
	d.Set("status", secret.Status)
	d.Set("expiration", expTime.Format(RFC3339NoZ))
	d.Set("created", secret.Created)
	if err := d.Set("content_types", secret.ContentTypes); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish secret reading")

	return diags
}

func resourceSecretDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start secret deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	secretID := d.Id()
	log.Printf("[DEBUG] Secret id = %s", secretID)

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	results, resp, err := clientV2.Secrets.Delete(ctx, secretID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			log.Printf("[DEBUG] Finish of Secret deleting")
			return diags
		}
		return diag.FromErr(err)
	}

	taskID := results.Tasks[0]

	err = utilV2.WaitForTaskComplete(ctx, clientV2, taskID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[DEBUG] Finish of secret deleting")

	return diags
}
