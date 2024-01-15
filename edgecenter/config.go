package edgecenter

import (
	dnsSDK "github.com/Edge-Center/edgecenter-dns-sdk-go"
	storageSDK "github.com/Edge-Center/edgecenter-storage-sdk-go"
	cdn "github.com/Edge-Center/edgecentercdn-go"
	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

type Config struct {
	Provider      *edgecloud.ProviderClient
	CloudClient   *edgecloudV2.Client
	CDNClient     cdn.ClientService
	StorageClient *storageSDK.SDK
	DNSClient     *dnsSDK.Client
}
