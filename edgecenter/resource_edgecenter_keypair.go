package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
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
				Computed:     true,
				ForceNew:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
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
	clientV2 := config.CloudClient

	projectID, err := GetProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	// To work with KeyPairsV2 endpoints, you only need a project.
	// Therefore, a stub with a value of 1 is applied for the region.
	clientV2.Region = 1
	clientV2.Project = projectID

	opts := &edgecloudV2.KeyPairCreateRequestV2{
		SSHKeyName: d.Get("sshkey_name").(string),
		PublicKey:  d.Get("public_key").(string),
		ProjectID:  projectID,
	}

	kp, _, err := clientV2.KeyPairs.CreateV2(ctx, opts)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] KeyPair id (%s)", kp.SSHKeyID)
	d.SetId(kp.SSHKeyID)

	resourceKeypairRead(ctx, d, m)

	log.Printf("[DEBUG] Finish KeyPair creating (%s)", kp.SSHKeyID)

	return diags
}

func resourceKeypairRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start KeyPair reading")

	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	projectID, err := GetProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	// To work with KeyPairsV2 endpoints, you only need a project.
	// Therefore, a stub with a value of 1 is applied for the region.
	clientV2.Region = 1
	clientV2.Project = projectID

	kpID := d.Id()
	kp, _, err := clientV2.KeyPairs.GetV2(ctx, kpID)
	if err != nil {
		return diag.Errorf("cannot get keypairs with ID %s. Error: %s", kpID, err.Error())
	}

	d.Set("sshkey_name", kp.SSHKeyName)
	d.Set("public_key", kp.PublicKey)
	d.Set("sshkey_id", kp.SSHKeyID)
	d.Set("fingerprint", kp.Fingerprint)
	d.Set("project_id", projectID)

	log.Println("[DEBUG] Finish KeyPair reading")

	return diags
}

func resourceKeypairDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start KeyPair deleting")

	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	projectID, err := GetProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	// To work with KeyPairsV2 endpoints, you only need a project.
	// Therefore, a stub with a value of 1 is applied for the region.
	clientV2.Region = 1
	clientV2.Project = projectID

	kpID := d.Id()
	if _, err := clientV2.KeyPairs.DeleteV2(ctx, kpID); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish of KeyPair deleting")

	return diags
}
