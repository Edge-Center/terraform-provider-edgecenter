package edgecenter

import (
	"context"
	"fmt"

	dnsSDK "github.com/Edge-Center/edgecenter-dns-sdk-go"
	storageSDK "github.com/Edge-Center/edgecenter-storage-sdk-go"
	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/buckets"
	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/locations"
	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/storages"
	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/models"
	cdn "github.com/Edge-Center/edgecentercdn-go"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	rmon "github.com/Edge-Center/edgecenteredgemon-go"
	protection "github.com/Edge-Center/edgecenterprotection-go"
)

type DNSZoneService interface {
	CreateZone(ctx context.Context, name string) (uint64, error)
	Zone(ctx context.Context, name string) (dnsSDK.Zone, error)
	DeleteZone(ctx context.Context, name string) error
}

type DNSRecordService interface {
	RRSet(ctx context.Context, zone, name, recordType string) (dnsSDK.RRSet, error)
	CreateRRSet(ctx context.Context, zone, name, recordType string, record dnsSDK.RRSet) error
	UpdateRRSet(ctx context.Context, zone, name, recordType string, record dnsSDK.RRSet) error
	DeleteRRSet(ctx context.Context, zone, name, recordType string) error
}

type DNSSecondaryZoneService interface {
	CreateSecondaryZone(ctx context.Context, req dnsSDK.CreateSecondaryZoneRequest) (dnsSDK.SecondaryZone, error)
	GetSecondaryZone(ctx context.Context, name string) (dnsSDK.SecondaryZone, error)
	UpdateSecondaryZone(ctx context.Context, name string, req dnsSDK.UpdateSecondaryZoneRequest) (dnsSDK.SecondaryZone, error)
	DeleteSecondaryZone(ctx context.Context, name string) error
	SecondaryZones(ctx context.Context) ([]dnsSDK.SecondaryZone, error)
}

// DNSClientService is the seam over the DNS SDK client. The SDK exports a concrete
// *dnsSDK.Client and no interface, so it is declared here, at the consumer, to keep
// the DNS resources mockable.
type DNSClientService interface {
	DNSZoneService
	DNSRecordService
	DNSSecondaryZoneService
}

var _ DNSClientService = (*dnsSDK.Client)(nil)

type StorageLocationService interface {
	LocationsList(opts ...func(params *locations.LocationListHTTPParams)) ([]models.ClientLocationRes, error)
}

type StorageS3Service interface {
	StoragesList(opts ...func(params *storages.StorageListHTTPV2Params)) ([]models.Storage, error)
	CreateStorage(opts ...func(params *storages.StorageCreateHTTPParams)) (*models.Storage, error)
	DeleteStorage(opts ...func(params *storages.StorageDeleteHTTPParams)) error
}

type StorageBucketService interface {
	BucketsList(opts ...func(params *buckets.StorageListBucketsHTTPParams)) ([]models.BucketDto, error)
	CreateBucket(opts ...func(params *buckets.StorageBucketCreateHTTPParams)) error
	DeleteBucket(opts ...func(params *buckets.StorageBucketRemoveHTTPParams)) error
}

type StorageClientService interface {
	StorageLocationService
	StorageS3Service
	StorageBucketService
}

var _ StorageClientService = (*storageSDK.SDK)(nil)

type Config struct {
	PermanentToken     string
	CloudBaseURL       string
	UserAgent          string
	Provider           *edgecloud.ProviderClient
	CDNClient          cdn.ClientService
	StorageClient      StorageClientService
	DNSClient          DNSClientService
	ProtectionClient   *protection.Client
	RmonClient         rmon.ClientService
	CloudClientFactory func() (*edgecloudV2.Client, error)
}

func NewConfig(
	provider *edgecloud.ProviderClient,
	cdnClient cdn.ClientService,
	storageClient StorageClientService,
	dnsClient DNSClientService,
	protectionClient *protection.Client,
	rmonClient rmon.ClientService,
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
		RmonClient:       rmonClient,
	}
}

func (c *Config) NewCloudClient() (*edgecloudV2.Client, error) {
	if c.CloudClientFactory != nil {
		return c.CloudClientFactory()
	}

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
