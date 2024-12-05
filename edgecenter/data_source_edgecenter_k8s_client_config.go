package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceK8sClientConfig() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "!> **WARNING:** This data source is deprecated and will be removed in the next major version. Data source \"edgecenter_k8s_client_config\" unavailable.",
		ReadContext:        dataSourceK8sReadClientConfig,
		Description:        "Represent k8s cluster with one default pool. \n\n **WARNING:** Data source \"edgecenter_k8s_client_config\" is deprecated and unavailable.",
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

func dataSourceK8sReadClientConfig(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.FromErr(fmt.Errorf("data source \"edgecenter_k8s_client_config\" is deprecated and unavailable"))
}
