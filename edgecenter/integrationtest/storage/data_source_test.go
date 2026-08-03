//go:build integration

package storage_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/buckets"
	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/storages"
	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/models"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	storagemock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/storage/mock"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
	storagesvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/storage"
)

func TestIntegrationStorageS3DataSourceLookupByName(t *testing.T) {
	t.Parallel()

	ds := storageDataSource(t, storagesvc.StorageS3DataSource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	var sent storages.StorageListHTTPV2Params

	mc.Storages.On("StoragesList", anyOpts(3)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[storages.StorageListHTTPV2Params](args) }).
		Return([]models.Storage{storageModel()}, nil)

	data := schema.TestResourceDataRaw(t, ds.Schema, map[string]interface{}{
		storagesvc.StorageSchemaName: storageName,
	})

	diags := ds.ReadContext(context.Background(), data, mc.Config)

	support.RequireNoDiags(t, diags)
	require.Nil(t, sent.ID)
	require.NotNil(t, sent.Name)
	require.Equal(t, storageName, *sent.Name)

	require.Equal(t, storageIDString, data.Id())
	require.Equal(t, storageName, data.Get(storagesvc.StorageSchemaName))
	require.Equal(t, 12345, data.Get(storagesvc.StorageSchemaClientID))
	require.Equal(t, storageLocation, data.Get(storagesvc.StorageSchemaLocation))
	require.Equal(t, storageAddress, data.Get(storagesvc.StorageSchemaGenerateEndpoint))
	require.Equal(t, "https://"+storageAddress, data.Get(storagesvc.StorageSchemaGenerateS3Endpoint))
	require.Equal(t,
		fmt.Sprintf("https://%s/{bucket_name}", storageAddress),
		data.Get(storagesvc.StorageSchemaGenerateHTTPEndpoint))
}

func TestIntegrationStorageS3DataSourceLookupByID(t *testing.T) {
	t.Parallel()

	ds := storageDataSource(t, storagesvc.StorageS3DataSource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	var sent storages.StorageListHTTPV2Params

	mc.Storages.On("StoragesList", anyOpts(3)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[storages.StorageListHTTPV2Params](args) }).
		Return([]models.Storage{storageModel()}, nil)

	data := schema.TestResourceDataRaw(t, ds.Schema, map[string]interface{}{
		storagesvc.StorageSchemaID: int(storageID),
	})

	diags := ds.ReadContext(context.Background(), data, mc.Config)

	support.RequireNoDiags(t, diags)
	require.NotNil(t, sent.ID)
	require.Equal(t, storageIDString, *sent.ID)
	require.Nil(t, sent.Name)
	require.Equal(t, storageIDString, data.Id())
}

func TestIntegrationStorageS3DataSourceMissingStorage(t *testing.T) {
	t.Parallel()

	ds := storageDataSource(t, storagesvc.StorageS3DataSource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	mc.Storages.On("StoragesList", anyOpts(3)...).Return([]models.Storage{}, nil)

	data := schema.TestResourceDataRaw(t, ds.Schema, map[string]interface{}{
		storagesvc.StorageSchemaName: storageName,
	})

	diags := ds.ReadContext(context.Background(), data, mc.Config)

	support.RequireOnlyErrorDiags(t, diags)
	support.RequireErrorDiagContains(t, diags, "wrong length of search result (0), want 1")
	require.Empty(t, data.Id())
}

func TestIntegrationStorageS3BucketDataSourceLookup(t *testing.T) {
	t.Parallel()

	ds := storageDataSource(t, storagesvc.StorageS3BucketDataSource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	var sent buckets.StorageListBucketsHTTPParams

	mc.Buckets.On("BucketsList", anyOpts(2)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[buckets.StorageListBucketsHTTPParams](args) }).
		Return(bucketList(bucketOtherName, bucketName), nil)

	data := schema.TestResourceDataRaw(t, ds.Schema, bucketConfig(bucketName))

	diags := ds.ReadContext(context.Background(), data, mc.Config)

	support.RequireNoDiags(t, diags)
	require.Equal(t, storageID, sent.ID)
	require.Equal(t, bucketResourceID, data.Id())
	require.Equal(t, bucketName, data.Get(storagesvc.StorageS3BucketSchemaName))
	require.Equal(t, int(storageID), data.Get(storagesvc.StorageS3BucketSchemaStorageID))
}

func TestIntegrationStorageS3BucketDataSourceMissingBucket(t *testing.T) {
	t.Parallel()

	ds := storageDataSource(t, storagesvc.StorageS3BucketDataSource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	mc.Buckets.On("BucketsList", anyOpts(2)...).Return(bucketList(bucketOtherName), nil)

	data := schema.TestResourceDataRaw(t, ds.Schema, bucketConfig(bucketName))

	diags := ds.ReadContext(context.Background(), data, mc.Config)

	support.RequireOnlyErrorDiags(t, diags)
	support.RequireErrorDiagContains(t, diags, "storage buckets list has not this bucket")
	require.Empty(t, data.Id())
}

func TestIntegrationStorageRegistration(t *testing.T) {
	t.Parallel()

	p := provider.Provider()

	require.Contains(t, p.ResourcesMap, storagesvc.StorageS3Resource)
	require.Contains(t, p.ResourcesMap, storagesvc.StorageS3BucketResource)
	require.Contains(t, p.DataSourcesMap, storagesvc.StorageS3DataSource)
	require.Contains(t, p.DataSourcesMap, storagesvc.StorageS3BucketDataSource)

	require.NoError(t, p.InternalValidate())
}
