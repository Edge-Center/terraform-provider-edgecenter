package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/volume/v1/volumes"
)

func dataSourceVolume() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceVolumeRead,
		Description: "Represent volume. A volume is a file storage which is similar to SSD and HDD hard disks",
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
			},
			"region_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
			},
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"project_id",
					"project_name",
				},
			},
			"region_name": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"region_id",
					"region_name",
				},
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"type_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Available value is 'standard', 'ssd_hiiops', 'cold', 'ultra'. Defaults to standard",
			},
			"metadata_k": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"metadata_kv": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata_read_only": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"read_only": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceVolumeRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Volume reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, VolumesPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	volumeOpts := &volumes.ListOpts{}
	if metadataK, ok := d.GetOk("metadata_k"); ok {
		volumeOpts.MetadataK = metadataK.(string)
	}

	if metadataRaw, ok := d.GetOk("metadata_kv"); ok {
		typedMetadataKV := make(map[string]string, len(metadataRaw.(map[string]interface{})))
		for k, v := range metadataRaw.(map[string]interface{}) {
			typedMetadataKV[k] = v.(string)
		}
		volumeOpts.MetadataKV = typedMetadataKV
	}

	vols, err := volumes.ListAll(client, volumeOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var volume volumes.Volume
	for _, v := range vols {
		if v.Name == name {
			volume = v
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("volume with name %s not found", name)
	}

	d.SetId(volume.ID)
	d.Set("name", volume.Name)
	d.Set("size", volume.Size)
	d.Set("type_name", volume.VolumeType)
	d.Set("region_id", volume.RegionID)
	d.Set("project_id", volume.ProjectID)

	metadataReadOnly := PrepareMetadataReadonly(volume.Metadata)
	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Volume reading")

	return diags
}
