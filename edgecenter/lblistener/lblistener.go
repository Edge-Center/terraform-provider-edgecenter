package lblistener

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

func lblistenerSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: `listener name`,
		},
		"loadbalancer_id": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "ID of the load balancer",
			ValidateFunc: validation.IsUUID,
		},
		"protocol": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			Description: fmt.Sprintf(
				"available values are '%s', '%s', '%s', '%s' and '%s'",
				edgecloud.ListenerProtocolHTTP, edgecloud.ListenerProtocolHTTPS,
				edgecloud.ListenerProtocolTCP, edgecloud.ListenerProtocolUDP, edgecloud.ListenerProtocolTerminatedHTTPS,
			),
			ValidateDiagFunc: func(val interface{}, key cty.Path) diag.Diagnostics {
				v := val.(string)
				switch edgecloud.LoadbalancerListenerProtocol(v) {
				case edgecloud.ListenerProtocolHTTP, edgecloud.ListenerProtocolHTTPS, edgecloud.ListenerProtocolTCP,
					edgecloud.ListenerProtocolUDP, edgecloud.ListenerProtocolTerminatedHTTPS:
					return diag.Diagnostics{}
				default:
					return diag.Errorf(
						"wrong protocol %s, available values are '%s', '%s', '%s', '%s', '%s'", v,
						edgecloud.ListenerProtocolHTTP, edgecloud.ListenerProtocolHTTPS,
						edgecloud.ListenerProtocolTCP, edgecloud.ListenerProtocolUDP,
						edgecloud.ListenerProtocolTerminatedHTTPS,
					)
				}
			},
		},
		"protocol_port": {
			Type:        schema.TypeInt,
			Required:    true,
			ForceNew:    true,
			Description: "port on which the protocol is bound",
		},
		"insert_x_forwarded": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "add headers X-Forwarded-For, X-Forwarded-Port, X-Forwarded-Proto to requests. only used with HTTP or TERMINATED_HTTPS protocols",
		},
		"secret_id": {
			Type:         schema.TypeString,
			Optional:     true,
			Description:  "ID of the secret where PKCS12 file is stored for the TERMINATED_HTTPS load balancer",
			ValidateFunc: validation.IsUUID,
		},
		"sni_secret_id": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "list of secret identifiers used for Server Name Indication (SNI).",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"allowed_cidrs": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "the allowed CIDRs for listener",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		// computed attributes
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "listener uuid",
		},
		"operating_status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "operating status of the listener",
		},
		"provisioning_status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "lifecycle status of the listener",
		},
		"pool_count": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "number of pools",
		},
		"insert_headers": {
			Type:        schema.TypeMap,
			Computed:    true,
			Description: "dictionary of additional header insertion into the HTTP headers. only used with the HTTP and TERMINATED_HTTPS protocols",
		},
	}
}
