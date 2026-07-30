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

	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/client/storages"
	"github.com/Edge-Center/edgecenter-storage-sdk-go/swagger/models"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support"
	storagemock "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/integrationtest/support/storage/mock"
	storagesvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/storage"
)

const (
	storageName        = "mystorage"
	storagePaddedName  = "  mystorage  "
	storagePrefixed    = "12345-mystorage"
	storageLocation    = "s-dt2"
	storageDeniedPlace = "s-dt9"
	storageID          = int64(7)
	storageIDString    = "7"
	storageAddress     = "s-dt2.example.com"
	storageAccessKey   = "access-key"
	storageSecretKey   = "secret-key"
)

func allowedLocations() []models.ClientLocationRes {
	return []models.ClientLocationRes{
		{Name: storageLocation, AllowForNewStorage: "allow"},
		{Name: storageDeniedPlace, AllowForNewStorage: "deny"},
	}
}

func storageModel() models.Storage {
	return models.Storage{
		ID:       storageID,
		Name:     storagePrefixed,
		Location: storageLocation,
		Address:  storageAddress,
	}
}

func storageWithCredentials() models.Storage {
	st := storageModel()
	st.Credentials = &models.Credentials{S3: &models.S3Credentials{
		AccessKey: storageAccessKey,
		SecretKey: storageSecretKey,
	}}

	return st
}

func storageConfig(name, location string) map[string]interface{} {
	return map[string]interface{}{
		storagesvc.StorageSchemaName:     name,
		storagesvc.StorageSchemaLocation: location,
	}
}

func s3CreateCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	var sent storages.StorageCreateHTTPParams
	created := storageWithCredentials()

	mc.Locations.On("LocationsList", anyOpts(1)...).Return(allowedLocations(), nil)
	mc.Storages.On("CreateStorage", anyOpts(4)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[storages.StorageCreateHTTPParams](args) }).
		Return(&created, nil)
	mc.Storages.On("StoragesList", anyOpts(4)...).Return([]models.Storage{storageModel()}, nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "create sends the trimmed name and reads the storage back",
		Op:        support.OpApply,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		NewConfig: storageConfig(storagePaddedName, storageLocation),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireNoDiags(t, diags)

			require.Equal(t, "s3", sent.Body.Type)
			require.Equal(t, storageName, sent.Body.Name)
			require.Equal(t, storageLocation, sent.Body.Location)

			support.RequireStateID(t, state, storageIDString)
			support.RequireStateAttrs(t, state, map[string]string{
				storagesvc.StorageSchemaName:                storageName,
				storagesvc.StorageSchemaClientID:            "12345",
				storagesvc.StorageSchemaLocation:            storageLocation,
				storagesvc.StorageSchemaID:                  storageIDString,
				storagesvc.StorageS3SchemaGenerateAccessKey: storageAccessKey,
				storagesvc.StorageS3SchemaGenerateSecretKey: storageSecretKey,
				storagesvc.StorageSchemaGenerateEndpoint:    storageAddress,
				storagesvc.StorageSchemaGenerateS3Endpoint:  "https://" + storageAddress,
				storagesvc.StorageSchemaGenerateHTTPEndpoint: fmt.Sprintf(
					"https://%s/{bucket_name}", storageAddress),
			})
		},
	}
}

func s3CreateDeniedLocationCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Locations.On("LocationsList", anyOpts(1)...).Return(allowedLocations(), nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "create rejects a location that is not open for new storages",
		Op:        support.OpApply,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		NewConfig: storageConfig(storageName, storageDeniedPlace),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "Wrong name of location: "+storageDeniedPlace)
			support.RequireErrorDiagContains(t, diags, "available locations: "+storageLocation)
			fake.Storages.AssertNotCalled(t, "CreateStorage")
		},
	}
}

func s3CreateLocationsFailureCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Locations.On("LocationsList", anyOpts(1)...).
		Return(nil, fmt.Errorf("api error: locations unavailable"))

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "create fails when the locations list is unavailable",
		Op:        support.OpApply,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		NewConfig: storageConfig(storageName, storageLocation),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, fake *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "locations unavailable")
			fake.Storages.AssertNotCalled(t, "CreateStorage")
		},
	}
}

func s3CreateAPIFailureCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Locations.On("LocationsList", anyOpts(1)...).Return(allowedLocations(), nil)
	mc.Storages.On("CreateStorage", anyOpts(4)...).
		Return(nil, fmt.Errorf("api error: quota exceeded"))

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "create surfaces the api error",
		Op:        support.OpApply,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		NewConfig: storageConfig(storageName, storageLocation),
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "create storage")
			support.RequireErrorDiagContains(t, diags, "quota exceeded")
			require.Nil(t, state, "state must be nil when create fails")
		},
	}
}

func s3ReadSplitsClientPrefixCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	var sent storages.StorageListHTTPV2Params

	mc.Storages.On("StoragesList", anyOpts(3)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[storages.StorageListHTTPV2Params](args) }).
		Return([]models.Storage{storageModel()}, nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "read filters by id, hides deleted storages and splits the client prefix",
		Op:        support.OpRead,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: storageIDString,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireNoDiags(t, diags)

			require.NotNil(t, sent.ID)
			require.Equal(t, storageIDString, *sent.ID)
			require.NotNil(t, sent.ShowDeleted)
			require.False(t, *sent.ShowDeleted)
			require.Nil(t, sent.Name)

			support.RequireStateID(t, state, storageIDString)
			support.RequireStateAttrs(t, state, map[string]string{
				storagesvc.StorageSchemaName:     storageName,
				storagesvc.StorageSchemaClientID: "12345",
			})
		},
	}
}

func s3ReadDashedNameCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	dashed := storageModel()
	dashed.Name = "my-own-storage"

	mc.Storages.On("StoragesList", anyOpts(3)...).Return([]models.Storage{dashed}, nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "read treats everything before the first dash as a client id",
		Op:        support.OpRead,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: storageIDString,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				storagesvc.StorageSchemaName:     "own-storage",
				storagesvc.StorageSchemaClientID: "0",
			})
		},
	}
}

func s3ReadUndashedNameCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	plain := storageModel()
	plain.Name = storageName

	mc.Storages.On("StoragesList", anyOpts(3)...).Return([]models.Storage{plain}, nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "read keeps a name without a dash and leaves the client id unset",
		Op:        support.OpRead,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: storageIDString,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireNoDiags(t, diags)
			support.RequireStateAttrs(t, state, map[string]string{
				storagesvc.StorageSchemaName: storageName,
			})
			require.Empty(t, state.Attributes[storagesvc.StorageSchemaClientID])
		},
	}
}

func s3ReadEmptyResultCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Storages.On("StoragesList", anyOpts(3)...).Return([]models.Storage{}, nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "read reports an empty search result instead of clearing the id",
		Op:        support.OpRead,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: storageIDString,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "wrong length of search result (0), want 1")
			support.RequireStateID(t, state, storageIDString)
		},
	}
}

func s3ReadMultipleMatchesCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	first := storageModel()
	second := storageModel()
	second.ID = 8

	mc.Storages.On("StoragesList", anyOpts(4)...).Return([]models.Storage{first, second}, nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:         "read by name reports every matching storage id",
		Op:           support.OpRead,
		Prepare:      func() *storagemock.MockedStorage { return mc },
		CurrentID:    storageIDString,
		CurrentState: map[string]interface{}{storagesvc.StorageSchemaName: storageName},
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "multiple storages found")
			support.RequireErrorDiagContains(t, diags, "ID: 7")
			support.RequireErrorDiagContains(t, diags, "ID: 8")
		},
	}
}

func s3ReadAPIFailureCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Storages.On("StoragesList", anyOpts(3)...).
		Return(nil, fmt.Errorf("api error: server unavailable"))

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "read surfaces the api error",
		Op:        support.OpRead,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: storageIDString,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "storages list")
			support.RequireErrorDiagContains(t, diags, "server unavailable")
		},
	}
}

func s3DeleteCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	var sent storages.StorageDeleteHTTPParams

	mc.Storages.On("DeleteStorage", anyOpts(2)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[storages.StorageDeleteHTTPParams](args) }).
		Return(nil)

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "delete sends the numeric storage id and drops the resource",
		Op:        support.OpDelete,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: storageIDString,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireNoDiags(t, diags)
			require.Equal(t, storageID, sent.ID)
			require.Nil(t, state, "state must be dropped after a successful delete")
		},
	}
}

func s3DeleteAPIFailureCase() support.ResourceCase[*storagemock.MockedStorage] {
	mc := storagemock.NewMockedStorage()

	mc.Storages.On("DeleteStorage", anyOpts(2)...).
		Return(fmt.Errorf("api error: storage is busy"))

	return support.ResourceCase[*storagemock.MockedStorage]{
		Name:      "delete keeps the id when the api fails",
		Op:        support.OpDelete,
		Prepare:   func() *storagemock.MockedStorage { return mc },
		CurrentID: storageIDString,
		Check: func(t *testing.T, state *terraform.InstanceState, diags diag.Diagnostics, _ *storagemock.MockedStorage) {
			support.RequireOnlyErrorDiags(t, diags)
			support.RequireErrorDiagContains(t, diags, "storage is busy")
			support.RequireStateID(t, state, storageIDString)
		},
	}
}

func TestIntegrationStorageS3(t *testing.T) {
	t.Parallel()

	res := storageResource(t, storagesvc.StorageS3Resource)

	cases := []support.ResourceCase[*storagemock.MockedStorage]{
		s3CreateCase(),
		s3CreateDeniedLocationCase(),
		s3CreateLocationsFailureCase(),
		s3CreateAPIFailureCase(),
		s3ReadSplitsClientPrefixCase(),
		s3ReadDashedNameCase(),
		s3ReadUndashedNameCase(),
		s3ReadEmptyResultCase(),
		s3ReadMultipleMatchesCase(),
		s3ReadAPIFailureCase(),
		s3DeleteCase(),
		s3DeleteAPIFailureCase(),
	}

	support.RunResourceCases(t, res, cases, support.DispatchCase[*storagemock.MockedStorage])
}

func TestIntegrationStorageS3ReadWithoutIDAndName(t *testing.T) {
	t.Parallel()

	res := storageResource(t, storagesvc.StorageS3Resource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	data := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})

	diags := res.ReadContext(context.Background(), data, mc.Config)

	support.RequireOnlyErrorDiags(t, diags)
	support.RequireErrorDiagContains(t, diags, "empty storage id/name")
	mc.Storages.AssertNotCalled(t, "StoragesList")
}

func TestIntegrationStorageS3ReadFallsBackToStorageID(t *testing.T) {
	t.Parallel()

	res := storageResource(t, storagesvc.StorageS3Resource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	var sent storages.StorageListHTTPV2Params

	mc.Storages.On("StoragesList", anyOpts(3)...).
		Run(func(args mock.Arguments) { sent = appliedOpts[storages.StorageListHTTPV2Params](args) }).
		Return([]models.Storage{storageModel()}, nil)

	data := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
		storagesvc.StorageSchemaID: int(storageID),
	})
	require.Empty(t, data.Id())

	diags := res.ReadContext(context.Background(), data, mc.Config)

	support.RequireNoDiags(t, diags)
	require.NotNil(t, sent.ID)
	require.Equal(t, storageIDString, *sent.ID)
	require.Equal(t, storageIDString, data.Id())
}

func TestIntegrationStorageS3DeleteWithoutID(t *testing.T) {
	t.Parallel()

	res := storageResource(t, storagesvc.StorageS3Resource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	data := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})

	diags := res.DeleteContext(context.Background(), data, mc.Config)

	support.RequireOnlyErrorDiags(t, diags)
	support.RequireErrorDiagContains(t, diags, "empty storage id")
	mc.Storages.AssertNotCalled(t, "DeleteStorage")
}

func TestIntegrationStorageS3CreateWithoutCredentialsPanics(t *testing.T) {
	t.Parallel()

	res := storageResource(t, storagesvc.StorageS3Resource)

	mc := storagemock.NewMockedStorage()
	t.Cleanup(func() { mc.MockCleanup(t) })

	created := storageModel()

	mc.Locations.On("LocationsList", anyOpts(1)...).Return(allowedLocations(), nil)
	mc.Storages.On("CreateStorage", anyOpts(4)...).Return(&created, nil)

	data := schema.TestResourceDataRaw(t, res.Schema, storageConfig(storageName, storageLocation))

	require.Panics(t, func() {
		_ = res.CreateContext(context.Background(), data, mc.Config)
	})
}
