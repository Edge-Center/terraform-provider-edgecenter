package volume

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func volumeSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"project_id": {
			Type:        schema.TypeInt,
			Required:    true,
			ForceNew:    true,
			Description: "uuid of the project",
		},
		"region_id": {
			Type:        schema.TypeInt,
			Required:    true,
			ForceNew:    true,
			Description: "uuid of the region",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "name of the volume",
		},
		"size": {
			Type:         schema.TypeInt,
			Required:     true,
			Description:  "size of the volume, specified in gigabytes (GB)",
			ValidateFunc: validation.IntAtLeast(1),
		},
		"source": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "volume source. valid values are 'new-volume', 'snapshot' or 'image'",
			ValidateFunc: validation.StringInSlice([]string{"new-volume", "snapshot", "image"}, false),
		},
		"metadata": {
			Type:        schema.TypeMap,
			Optional:    true,
			Description: "map containing metadata, for example tags.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"volume_type": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "volume type with valid values. defaults to 'standard'",
			ValidateFunc: validation.StringInSlice([]string{
				string(edgecloud.VolumeTypeSsdHiIops),
				string(edgecloud.VolumeTypeSsdLocal),
				string(edgecloud.VolumeTypeUltra),
				string(edgecloud.VolumeTypeCold),
				string(edgecloud.VolumeTypeStandard),
			}, false),
			Default: string(edgecloud.VolumeTypeStandard),
		},
		"instance_id_to_attach_to": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "VM’s instance_id to attach a newly created volume to",
		},
		"attachment_tag": {
			Type:         schema.TypeString,
			Optional:     true,
			Description:  "the block device attachment tag (exposed in the metadata)",
			RequiredWith: []string{"instance_id_to_attach_to"},
		},
		"image_id": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "ID of the image. this field is mandatory if creating a volume from an image",
		},
		"snapshot_id": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "ID of the snapshot. this field is mandatory if creating a volume from a snapshot",
		},
		// computed attributes
		"region": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "name of the region",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "status of the volume",
		},
		"bootable": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "the bootable boolean flag",
		},
		"limiter_stats": {
			Type:        schema.TypeMap,
			Computed:    true,
			Description: "the QoS parameters of this volume",
			Elem: &schema.Schema{
				Type: schema.TypeInt,
			},
		},
		"attachments": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "the attachment list",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"volume_id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "ID of the volume",
					},
					"attachment_id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "ID of the attachment object",
					},
					"server_id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "ID of the instance",
					},
				},
			},
		},
		"snapshot_ids": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "snapshots of the volume",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	}
}
