//go:build integration

package storage_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/buckets"
	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/models"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	storagemock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/storage/mock"
	storagesvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/storage"
)

const (
	bucketName       = "mybucket"
	bucketPaddedName = "  mybucket  "
	bucketOtherName  = "otherbucket"
	bucketResourceID = "7:mybucket"
)

func bucketConfig(name string) map[string]interface{} {
	return map[string]interface{}{
		storagesvc.StorageS3BucketSchemaStorageID: int(storageID),
		storagesvc.StorageS3BucketSchemaName:      name,
	}
}

func bucketList(names ...string) []models.BucketDto {
	list := make([]models.BucketDto, 0, len(names))
	for _, name := range names {
		list = append(list, models.BucketDto{Name: name})
	}

	return list
}

func bucketCreateCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	var sent buckets.StorageBucketCreateHTTPParams

	mc.Buckets.On("CreateBucket", anyOpts(2)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[buckets.StorageBucketCreateHTTPParams](args) }).
		Return(nil)
	mc.Buckets.On("BucketsList", anyOpts(2)...).Return(bucketList(bucketOtherName, bucketName), nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "create sends the trimmed name and builds a composite id",
		Op:        support.OpApply,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		NewConfig: bucketConfig(bucketPaddedName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireNoDiags(t, diags)

			require.Equal(t, storageID, sent.ID)
			require.Equal(t, bucketName, sent.Name)

			support.RequireStateID(t, state, bucketResourceID)
			support.RequireStateAttrs(t, state, map[string]string{
				storagesvc.StorageS3BucketSchemaStorageID: storageIDString,
				storagesvc.StorageS3BucketSchemaName:      bucketName,
			})
		},
	}
}

func bucketCreateAPIFailureCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Buckets.On("CreateBucket", anyOpts(2)...).
		Return(fmt.Errorf("api error: bucket already exists"))

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "create surfaces the api error",
		Op:        support.OpApply,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		NewConfig: bucketConfig(bucketName),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create storage bucket")
			support.RequireErrorDiagContains(t, diags, "bucket already exists")
			require.Nil(t, state, "state must be nil when create fails")
			fake.Buckets.AssertNotCalled(t, "BucketsList")
		},
	}
}

func bucketReadCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	var sent buckets.StorageListBucketsHTTPParams

	mc.Buckets.On("BucketsList", anyOpts(2)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[buckets.StorageListBucketsHTTPParams](args) }).
		Return(bucketList(bucketOtherName, bucketName), nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "read finds the bucket among the storage buckets",
		Op:        support.OpRead,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: bucketResourceID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireNoDiags(t, diags)
			require.Equal(t, storageID, sent.ID)
			support.RequireStateID(t, state, bucketResourceID)
			support.RequireStateAttrs(t, state, map[string]string{
				storagesvc.StorageS3BucketSchemaStorageID: storageIDString,
				storagesvc.StorageS3BucketSchemaName:      bucketName,
			})
		},
	}
}

func bucketReadEmptyListCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Buckets.On("BucketsList", anyOpts(2)...).Return(bucketList(), nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "read reports an empty bucket list instead of clearing the id",
		Op:        support.OpRead,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: bucketResourceID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "wrong length of search result (0), want more")
			support.RequireStateID(t, state, bucketResourceID)
		},
	}
}

func bucketReadMissingCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Buckets.On("BucketsList", anyOpts(2)...).Return(bucketList(bucketOtherName), nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "read errors when the bucket is gone instead of clearing the id",
		Op:        support.OpRead,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: bucketResourceID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "storage buckets list has not this bucket")
			support.RequireStateID(t, state, bucketResourceID)
		},
	}
}

func bucketReadAPIFailureCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Buckets.On("BucketsList", anyOpts(2)...).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "read surfaces the api error",
		Op:        support.OpRead,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: bucketResourceID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "storage buckets list")
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func bucketDeleteCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	var sent buckets.StorageBucketRemoveHTTPParams

	mc.Buckets.On("DeleteBucket", anyOpts(3)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[buckets.StorageBucketRemoveHTTPParams](args) }).
		Return(nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "delete sends the storage id and the bucket name",
		Op:        support.OpDelete,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: bucketResourceID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireNoDiags(t, diags)
			require.Equal(t, storageID, sent.ID)
			require.Equal(t, bucketName, sent.Name)
			require.Nil(t, state, "state must be dropped after a successful delete")
		},
	}
}

func bucketDeleteAPIFailureCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Buckets.On("DeleteBucket", anyOpts(3)...).
		Return(fmt.Errorf("api error: bucket is not empty"))

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "delete keeps the id when the api fails",
		Op:        support.OpDelete,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: bucketResourceID,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "bucket is not empty")
			support.RequireStateID(t, state, bucketResourceID)
		},
	}
}

func TestIntegrationStorageS3Bucket(t *testing.T) {
	t.Parallel()

	res := storageResource(t, storagesvc.StorageS3BucketResource)

	cases := []support.ResourceCase[*storagemock.MockedStorage]{
		bucketCreateCase(),
		bucketCreateAPIFailureCase(),
		bucketReadCase(),
		bucketReadEmptyListCase(),
		bucketReadMissingCase(),
		bucketReadAPIFailureCase(),
		bucketDeleteCase(),
		bucketDeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, res, cases, support.DispatchCase[*storagemock.MockedStorage])
}

func TestIntegrationStorageS3BucketDeleteWithoutName(t *testing.T) {
	t.Parallel()

	res := storageResource(t, storagesvc.StorageS3BucketResource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	data := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})

	diags := res.DeleteContext(context.Background(), data, mc.Config)

	support.RequireOnlyErrorDiags(t, diags)
	support.RequireErrorDiagContains(t, diags, "empty bucket")
	mc.Buckets.AssertNotCalled(t, "DeleteBucket")
}

func TestIntegrationStorageS3BucketMalformedIDLosesBothParts(t *testing.T) {
	t.Parallel()

	res := storageResource(t, storagesvc.StorageS3BucketResource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	var sent buckets.StorageListBucketsHTTPParams

	mc.Buckets.On("BucketsList", anyOpts(2)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[buckets.StorageListBucketsHTTPParams](args) }).
		Return(bucketList(bucketName), nil)

	data := schema.TestResourceDataRaw(t, res.Schema, bucketConfig(bucketName))
	data.SetId("7:my:bucket")

	diags := res.ReadContext(context.Background(), data, mc.Config)

	require.Zero(t, sent.ID)
	support.RequireOnlyErrorDiags(t, diags)
	support.RequireErrorDiagContains(t, diags, "storage buckets list has not this bucket")
}

func TestIntegrationStorageS3BucketReadFallsBackToSchema(t *testing.T) {
	t.Parallel()

	res := storageResource(t, storagesvc.StorageS3BucketResource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	var sent buckets.StorageListBucketsHTTPParams

	mc.Buckets.On("BucketsList", anyOpts(2)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[buckets.StorageListBucketsHTTPParams](args) }).
		Return(bucketList(bucketName), nil)

	data := schema.TestResourceDataRaw(t, res.Schema, bucketConfig(bucketPaddedName))
	require.Empty(t, data.Id())

	diags := res.ReadContext(context.Background(), data, mc.Config)

	support.RequireNoDiags(t, diags)
	require.Equal(t, storageID, sent.ID)
	require.Equal(t, bucketResourceID, data.Id())
}
