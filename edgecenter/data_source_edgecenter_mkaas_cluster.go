package edgecenter

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceMKaaSCluster() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMKaaSClusterRead,
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
				Description:  "The name of the MKaaS cluster. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
			},
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The ID of the MKaaS cluster. Either 'id' or 'name' must be specified.",
				ExactlyOneOf: []string{"id", "name"},
			},

			"ssh_keypair_name": {Type: schema.TypeString, Computed: true, Description: "SSH keypair name."},
			"network_id":       {Type: schema.TypeString, Computed: true, Description: "Network ID."},
			"subnet_id":        {Type: schema.TypeString, Computed: true, Description: "Subnet ID."},
			"internal_ip":      {Type: schema.TypeString, Computed: true, Description: "Internal cluster IP."},
			"external_ip":      {Type: schema.TypeString, Computed: true, Description: "External cluster IP."},
			"created":          {Type: schema.TypeString, Computed: true},
			"processing":       {Type: schema.TypeBool, Computed: true},
			"status":           {Type: schema.TypeString, Computed: true},
			"stage":            {Type: schema.TypeString, Computed: true},

			"control_plane": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"flavor":      {Type: schema.TypeString, Computed: true},
						"node_count":  {Type: schema.TypeInt, Computed: true},
						"volume_size": {Type: schema.TypeInt, Computed: true},
						"volume_type": {Type: schema.TypeString, Computed: true},
						"version":     {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceMKaaSClusterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "[DEBUG] Start MKaaS cluster reading")

	clientV2, err := InitCloudClient(ctx, d, m, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	var (
		cluster   *edgecloudV2.MKaaSCluster
		clusterID int
	)

	if v, ok := d.GetOk("id"); ok && v.(string) != "" {
		clusterID, err = strconv.Atoi(v.(string))
		if err != nil {
			return diag.Errorf("invalid id: %s", err)
		}

		var resp *edgecloudV2.Response
		cluster, resp, err = clientV2.MkaaS.ClusterGet(ctx, clusterID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return diag.Errorf("MKaaS cluster %d not found", clusterID)
			}
			return diag.FromErr(err)
		}
		if cluster == nil {
			return diag.Errorf("MKaaS cluster %d: empty response", clusterID)
		}

		clusterID = cluster.ID
	} else {
		name := d.Get("name").(string)
		if name == "" {
			return diag.Errorf("either 'id' or 'name' must be specified")
		}

		opts := &edgecloudV2.MKaaSClusterListOptions{
			Name:  name,
			Limit: 2,
		}

		clusters, _, err := clientV2.MkaaS.ClustersList(ctx, opts)
		if err != nil {
			return diag.FromErr(err)
		}
		if len(clusters) == 0 {
			return diag.Errorf("MKaaS cluster with name %s not found", name)
		}
		if len(clusters) > 1 {
			return diag.Errorf("multiple MKaaS clusters found with name %s; please specify 'id'", name)
		}

		cluster = &clusters[0]
		clusterID = cluster.ID
	}

	d.SetId(strconv.Itoa(clusterID))

	_ = d.Set("project_id", cluster.ProjectID)
	_ = d.Set("region_id", cluster.RegionID)

	_ = d.Set("name", cluster.Name)
	_ = d.Set("id", strconv.Itoa(cluster.ID))
	_ = d.Set("ssh_keypair_name", cluster.SSHKeypairName)
	_ = d.Set("network_id", cluster.NetworkID)
	_ = d.Set("subnet_id", cluster.SubnetID)

	cp := map[string]interface{}{
		"flavor":      cluster.ControlPlane.Flavor,
		"node_count":  cluster.ControlPlane.NodeCount,
		"volume_size": cluster.ControlPlane.VolumeSize,
		"volume_type": string(cluster.ControlPlane.VolumeType),
		"version":     cluster.ControlPlane.Version,
	}
	_ = d.Set("control_plane", []interface{}{cp})

	_ = d.Set("internal_ip", cluster.InternalIP)
	_ = d.Set("external_ip", cluster.ExternalIP)
	_ = d.Set("created", cluster.Created)
	_ = d.Set("processing", cluster.Processing)
	_ = d.Set("status", cluster.Status)
	_ = d.Set("stage", cluster.Stage)

	tflog.Debug(ctx, "[DEBUG] Finish MKaaS cluster reading")

	return diag.Diagnostics{}
}
