package config

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
)

type Config struct {
	TerraformVersion string
	APIKey           string
	CloudAPIURL      string
}

type CombinedConfig struct {
	cloud *edgecloud.Client
}

func (c *CombinedConfig) EdgeCloudClient() *edgecloud.Client { return c.cloud }

// Client returns a new client for accessing Edgecenter Cloud client.
func (c *Config) Client() (*CombinedConfig, diag.Diagnostics) {
	userAgent := fmt.Sprintf("Terraform/%s", c.TerraformVersion)

	client, err := edgecloud.NewWithRetries(nil,
		edgecloud.SetUserAgent(userAgent),
		edgecloud.SetAPIKey(c.APIKey),
		edgecloud.SetBaseURL(c.CloudAPIURL),
	)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("edgecloud client create error: %w", err))
	}

	clientTransport := logging.NewSubsystemLoggingHTTPTransport("EdgeCenter", client.HTTPClient.Transport)
	client.HTTPClient.Transport = clientTransport

	log.Printf("[INFO] EdgeCenter Client configured for URL: %s", client.BaseURL.String())

	return &CombinedConfig{cloud: client}, nil
}
