package edgecenter

import (
	"context"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/image/v1/images"
)

const (
	ImagesPoint   = "images"
	bmImagesPoint = "bmimages"
)

func dataSourceImage() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceImageRead,
		Description: "Represent image data",
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
				Type:        schema.TypeString,
				Description: "use 'os-version', for example 'ubuntu-20.04'",
				Required:    true,
			},
			"is_baremetal": {
				Type:        schema.TypeBool,
				Description: "set to true if need to get baremetal image",
				Optional:    true,
			},
			"min_disk": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"min_ram": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"os_distro": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"os_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
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

func dataSourceImageRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Image reading")
	name := d.Get("name").(string)

	config := m.(*Config)
	provider := config.Provider

	point := ImagesPoint
	if isBm, _ := d.Get("is_baremetal").(bool); isBm {
		point = bmImagesPoint
	}
	client, err := CreateClient(provider, d, point, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	listOpts := &images.ListOpts{}

	if metadataK, ok := d.GetOk("metadata_k"); ok {
		listOpts.MetadataK = metadataK.(string)
	}

	if metadataRaw, ok := d.GetOk("metadata_kv"); ok {
		typedMetadataKV := make(map[string]string, len(metadataRaw.(map[string]interface{})))
		for k, v := range metadataRaw.(map[string]interface{}) {
			typedMetadataKV[k] = v.(string)
		}
		listOpts.MetadataKV = typedMetadataKV
	}

	allImages, err := images.ListAll(client, *listOpts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var image images.Image
	for _, img := range allImages {
		if strings.HasPrefix(strings.ToLower(img.Name), strings.ToLower(name)) {
			image = img
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("image with name %s not found", name)
	}

	d.SetId(image.ID)
	d.Set("project_id", d.Get("project_id").(int))
	d.Set("region_id", d.Get("region_id").(int))
	d.Set("min_disk", image.MinDisk)
	d.Set("min_ram", image.MinRAM)
	d.Set("os_distro", image.OsDistro)
	d.Set("os_version", image.OsVersion)
	d.Set("description", image.Description)

	metadataReadOnly := PrepareMetadataReadonly(image.Metadata)
	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Image reading")

	return nil
}
