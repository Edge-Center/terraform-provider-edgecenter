package edgecenter

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceImage() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceImageRead,
		Description: "A cloud image is a pre-configured virtual machine template that you can use to create new instances.",
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
				Description: "The name of the image. Use 'os-version', for example 'ubuntu-20.04'.",
				Required:    true,
			},
			"is_baremetal": {
				Type:        schema.TypeBool,
				Description: "Set to true if need to get the baremetal image.",
				Optional:    true,
			},
			"min_disk": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Minimum disk space (in GB) required to launch an instance using this image.",
			},
			"min_ram": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Minimum VM RAM (in MB) required to launch an instance using this image.",
			},
			"os_distro": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The distribution of the OS present in the image, e.g. Debian, CentOS, Ubuntu etc.",
			},
			"os_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of the OS present in the image. e.g. 19.04 (for Ubuntu) or 9.4 for Debian.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A detailed description of the image.",
			},
			"metadata_k": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filtration query opts (only key).",
			},
			"metadata_kv": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: `Filtration query opts, for example, {offset = "10", limit = "10"}.`,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metadata_read_only": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `A list of read-only metadata items, e.g. tags.`,
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

func dataSourceImageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start Image reading")
	name := d.Get("name").(string)

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	listOpts := &edgecloudV2.ImageListOptions{}

	if metadataK, ok := d.GetOk("metadata_k"); ok {
		listOpts.MetadataK = metadataK.(string)
	}

	if metadataRaw, ok := d.GetOk("metadata_kv"); ok {
		typedMetadataKV := make(map[string]string, len(metadataRaw.(map[string]interface{})))
		for k, v := range metadataRaw.(map[string]interface{}) {
			typedMetadataKV[k] = v.(string)
		}
		typedMetadataKVJson, err := json.Marshal(typedMetadataKV)
		if err != nil {
			return diag.FromErr(err)
		}
		listOpts.MetadataKV = string(typedMetadataKVJson)
	}

	var allImages []edgecloudV2.Image

	if isBm, _ := d.Get("is_baremetal").(bool); isBm {
		allImages, _, err = clientV2.Images.ImagesBaremetalList(ctx, listOpts)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		allImages, _, err = clientV2.Images.List(ctx, nil)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	var found bool
	var image edgecloudV2.Image
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
	d.Set("project_id", clientV2.Project)
	d.Set("region_id", clientV2.Region)
	d.Set("min_disk", image.MinDisk)
	d.Set("min_ram", image.MinRAM)
	d.Set("os_distro", image.OSDistro)
	d.Set("os_version", image.OSVersion)
	d.Set("description", image.Description)

	metadataReadOnly := PrepareMetadataReadonly(image.MetadataDetailed)
	if err := d.Set("metadata_read_only", metadataReadOnly); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish Image reading")

	return nil
}
