package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/keypair/v2/keypairs"
)

const KeypairsPoint = "keypairs"

func resourceKeypair() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKeypairCreate,
		ReadContext:   resourceKeypairRead,
		DeleteContext: resourceKeypairDelete,
		Description:   "Represent a ssh key, do not depends on region",
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
			"public_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The public portion of the SSH key pair.",
			},
			"sshkey_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name assigned to the SSH key pair, used for identification purposes.",
			},
			"sshkey_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier assigned by the provider to the SSH key pair.",
			},
			"fingerprint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A fingerprint of the SSH public key, used to verify the integrity of the key.",
			},
		},
	}
}

func resourceKeypairCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start KeyPair creating")

	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, KeypairsPoint, VersionPointV2)
	if err != nil {
		return diag.FromErr(err)
	}

	opts := keypairs.CreateOpts{
		Name:      d.Get("sshkey_name").(string),
		PublicKey: d.Get("public_key").(string),
		ProjectID: d.Get("project_id").(int),
	}

	kp, err := keypairs.Create(client, opts).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] KeyPair id (%s)", kp.ID)
	d.SetId(kp.ID)

	resourceKeypairRead(ctx, d, m)

	log.Printf("[DEBUG] Finish KeyPair creating (%s)", kp.ID)

	return diags
}

func resourceKeypairRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start KeyPair reading")

	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, KeypairsPoint, VersionPointV2)
	if err != nil {
		return diag.FromErr(err)
	}

	kpID := d.Id()
	kp, err := keypairs.Get(client, kpID).Extract()
	if err != nil {
		return diag.Errorf("cannot get keypairs with ID %s. Error: %s", kpID, err.Error())
	}

	d.Set("sshkey_name", kp.Name)
	d.Set("public_key", kp.PublicKey)
	d.Set("sshkey_id", kp.ID)
	d.Set("fingerprint", kp.Fingerprint)
	d.Set("project_id", kp.ProjectID)

	log.Println("[DEBUG] Finish KeyPair reading")

	return diags
}

func resourceKeypairDelete(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start KeyPair deleting")

	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, KeypairsPoint, VersionPointV2)
	if err != nil {
		return diag.FromErr(err)
	}

	kpID := d.Id()
	if err := keypairs.Delete(client, kpID).ExtractErr(); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish of KeyPair deleting")

	return diags
}
