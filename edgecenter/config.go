package edgecenter

import (
	"fmt"

	dnsSDK "github.com/Edge-Center/edgecenter-dns-sdk-go"
	storageSDK "github.com/Edge-Center/edgecenter-storage-sdk-go"
	cdn "github.com/Edge-Center/edgecentercdn-go"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	protection "github.com/Edge-Center/edgecenterprotection-go"
)

type Config struct {
	PermanentToken   string
	CloudBaseURL     string
	UserAgent        string
	Provider         *edgecloud.ProviderClient
	CDNClient        cdn.ClientService
	StorageClient    *storageSDK.SDK
	DNSClient        *dnsSDK.Client
	ProtectionClient *protection.Client
}

func NewConfig(
	provider *edgecloud.ProviderClient,
	cdnClient cdn.ClientService,
	storageClient *storageSDK.SDK,
	dnsClient *dnsSDK.Client,
	protectionClient *protection.Client,
	permanentToken,
	cloudBaseURL,
	userAgent string,
) Config {
	return Config{
		PermanentToken:   permanentToken,
		CloudBaseURL:     cloudBaseURL,
		UserAgent:        userAgent,
		Provider:         provider,
		CDNClient:        cdnClient,
		StorageClient:    storageClient,
		DNSClient:        dnsClient,
		ProtectionClient: protectionClient,
	}
}

func (c *Config) NewCloudClient() (*edgecloudV2.Client, error) {
	cloudClient, err := edgecloudV2.NewWithRetries(nil,
		edgecloudV2.SetUserAgent(c.UserAgent),
		edgecloudV2.SetAPIKey(c.PermanentToken),
		edgecloudV2.SetBaseURL(c.CloudBaseURL),
	)
	if err != nil {
		return nil, fmt.Errorf("error from creating cloud client: %w", err)
	}
	return cloudClient, nil
}
