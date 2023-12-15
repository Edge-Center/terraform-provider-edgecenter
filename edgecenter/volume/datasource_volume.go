package volume

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/util"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/config"
)

func DataSourceEdgeCenterVolume() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceEdgeCenterVolumeRead,
		Description: `A volume is a detachable block storage device akin to a USB hard drive or SSD, but located remotely in the cloud.
Volumes can be attached to a virtual machine and manipulated like a physical hard drive.`,

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the project",
			},
			"region_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "uuid of the region",
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "volume uuid",
				ValidateFunc: validation.IsUUID,
				ExactlyOneOf: []string{"id", "name"},
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Description: `volume name. this parameter is not unique, if there is more than one volume with the same name, 
then the first one will be used. it is recommended to use "id"`,
				ExactlyOneOf: []string{"id", "name"},
			},
			// computed attributes
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "current status of the volume resource",
			},
			"size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "size of the volume, specified in gigabytes (GB)",
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "name of the region",
			},
			"volume_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "volume type",
			},
			"metadata": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "metadata in detailed format",
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
		},
	}
}

func dataSourceEdgeCenterVolumeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).EdgeCloudClient()
	client.Region = d.Get("region_id").(int)
	client.Project = d.Get("project_id").(int)

	var foundVolume *edgecloud.Volume

	if id, ok := d.GetOk("id"); ok {
		volume, _, err := client.Volumes.Get(ctx, id.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundVolume = volume
	} else if volumeName, ok := d.GetOk("name"); ok {
		volumeList, err := util.VolumesListByName(ctx, client, volumeName.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		foundVolume = &volumeList[0]
	} else {
		return diag.Errorf("Error: specify either id or a name to lookup the volume")
	}

	d.SetId(foundVolume.ID)
	d.Set("name", foundVolume.Name)

	d.Set("status", foundVolume.Status)
	d.Set("size", foundVolume.Size)
	d.Set("region", foundVolume.Region)
	d.Set("volume_type", foundVolume.VolumeType)
	d.Set("bootable", foundVolume.Bootable)

	if len(foundVolume.MetadataDetailed) > 0 {
		metadata := make([]map[string]interface{}, 0, len(foundVolume.MetadataDetailed))
		for _, metadataItem := range foundVolume.MetadataDetailed {
			metadata = append(metadata, map[string]interface{}{
				"key":       metadataItem.Key,
				"value":     metadataItem.Value,
				"read_only": metadataItem.ReadOnly,
			})
		}
		d.Set("metadata", metadata)
	}

	if len(foundVolume.Attachments) > 0 {
		attachments := make([]map[string]interface{}, 0, len(foundVolume.Attachments))
		for _, attachment := range foundVolume.Attachments {
			attachments = append(attachments, map[string]interface{}{
				"volume_id":     attachment.VolumeID,
				"attachment_id": attachment.AttachmentID,
				"server_id":     attachment.ServerID,
			})
		}
		d.Set("attachments", attachments)
	}

	d.Set("limiter_stats",
		map[string]int{
			"iops_base_limit":  foundVolume.LimiterStats.IopsBaseLimit,
			"iops_burst_limit": foundVolume.LimiterStats.IopsBurstLimit,
			"MBps_base_limit":  foundVolume.LimiterStats.MBpsBaseLimit,
			"MBps_burst_limit": foundVolume.LimiterStats.MBpsBurstLimit,
		})

	return nil
}
