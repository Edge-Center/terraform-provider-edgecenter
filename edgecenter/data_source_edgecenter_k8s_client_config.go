package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecentercloud-go/edgecenter/k8s/v1/clusters"
)

func dataSourceK8sClientConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceK8sReadClientConfig,
		Description: "Represent k8s cluster with one default pool.",
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
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The uuid of the Kubernetes cluster.",
			},
			"client_certificate_data": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The client_certificate_data field from k8s config.",
			},
			"client_key_data": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The client_key_data field from k8s config.",
			},
		},
	}
}

func dataSourceK8sReadClientConfig(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start K8s client config reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	provider := config.Provider

	client, err := CreateClient(provider, d, K8sPoint, VersionPointV1)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get("cluster_id").(string)

	d.SetId(clusterID)

	getConfigResult, err := clusters.GetConfig(client, clusterID).Extract()
	if err != nil {
		return diag.FromErr(err)
	}

	clusterConfig, err := parseK8sConfig(getConfigResult.Config)
	if err != nil {
		return diag.Errorf("failed to parse k8s config: %s", err)
	}

	clientCertificateData := clusterConfig.Users[0].User.ClientCertificateData
	if err := d.Set("client_certificate_data", clientCertificateData); err != nil {
		return diag.Errorf("couldn't get client_certificate_data: %s", err)
	}

	clientKeyData := clusterConfig.Users[0].User.ClientKeyData
	if err := d.Set("client_key_data", clientKeyData); err != nil {
		return diag.Errorf("couldn't get client_key_data: %s", err)
	}

	log.Println("[DEBUG] Finish K8s client config reading")

	return diags
}
