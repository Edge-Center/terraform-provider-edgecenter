package storage

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestServiceRegistersEveryStorageName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "storage", Service{}.Name())

	resources := Service{}.Resources()
	require.Len(t, resources, 2)
	require.Contains(t, resources, StorageS3Resource)
	require.Contains(t, resources, StorageS3BucketResource)

	dataSources := Service{}.DataSources()
	require.Len(t, dataSources, 2)
	require.Contains(t, dataSources, StorageS3DataSource)
	require.Contains(t, dataSources, StorageS3BucketDataSource)
}

func TestStorageS3ResourceSchema(t *testing.T) {
	t.Parallel()

	res := resourceStorageS3()

	require.NotNil(t, res.Importer)
	require.NotNil(t, res.CreateContext)
	require.NotNil(t, res.ReadContext)
	require.NotNil(t, res.DeleteContext)
	require.Nil(t, res.UpdateContext, "every attribute is ForceNew or computed")

	name := res.Schema[StorageSchemaName]
	require.True(t, name.Required)
	require.True(t, name.ForceNew)
	require.NotNil(t, name.ValidateDiagFunc)

	location := res.Schema[StorageSchemaLocation]
	require.True(t, location.Required)
	require.True(t, location.ForceNew)
	require.Nil(t, location.ValidateDiagFunc)

	for _, key := range []string{
		StorageSchemaID,
		StorageSchemaClientID,
		StorageS3SchemaGenerateAccessKey,
		StorageS3SchemaGenerateSecretKey,
		StorageSchemaGenerateHTTPEndpoint,
		StorageSchemaGenerateS3Endpoint,
		StorageSchemaGenerateEndpoint,
	} {
		attr := res.Schema[key]
		require.NotNilf(t, attr, "attribute %q is missing", key)
		require.Truef(t, attr.Optional, "attribute %q must stay optional", key)
		require.Truef(t, attr.Computed, "attribute %q must stay computed", key)
		require.Falsef(t, attr.ForceNew, "attribute %q must not force replacement", key)
	}

	require.True(t, res.Schema[StorageS3SchemaGenerateSecretKey].Optional)
	require.False(t, res.Schema[StorageS3SchemaGenerateSecretKey].Sensitive)
}

func TestStorageS3NameValidation(t *testing.T) {
	t.Parallel()

	validate := resourceStorageS3().Schema[StorageSchemaName].ValidateDiagFunc
	require.NotNil(t, validate)

	accepted := []string{"storage", "my_storage", "my-storage", "st123", strings.Repeat("a", 255)}
	for _, value := range accepted {
		require.Emptyf(t, validate(value, cty.Path{}), "value %q must be accepted", value)
	}

	rejected := []string{"", "  ", "my storage", "my.storage", "имя", strings.Repeat("a", 256)}
	for _, value := range rejected {
		require.NotEmptyf(t, validate(value, cty.Path{}), "value %q must be rejected", value)
	}
}

func TestStorageS3NameValidationRejectsPadding(t *testing.T) {
	t.Parallel()

	validate := resourceStorageS3().Schema[StorageSchemaName].ValidateDiagFunc

	require.NotEmpty(t, validate("  storage  ", cty.Path{}))
	require.Empty(t, validate("storage", cty.Path{}))
}

func TestStorageS3BucketResourceSchema(t *testing.T) {
	t.Parallel()

	res := resourceStorageS3Bucket()

	require.NotNil(t, res.Importer)
	require.Nil(t, res.UpdateContext)

	storageID := res.Schema[StorageS3BucketSchemaStorageID]
	require.True(t, storageID.Required)
	require.True(t, storageID.ForceNew)
	require.Equal(t, schema.TypeInt, storageID.Type)

	name := res.Schema[StorageS3BucketSchemaName]
	require.True(t, name.Required)
	require.True(t, name.ForceNew)
	require.NotNil(t, name.ValidateDiagFunc)

	require.Len(t, res.Schema, 2)
}

func TestStorageS3BucketNameValidation(t *testing.T) {
	t.Parallel()

	validate := resourceStorageS3Bucket().Schema[StorageS3BucketSchemaName].ValidateDiagFunc
	require.NotNil(t, validate)

	accepted := []string{"abc", "my-bucket", "my_bucket", strings.Repeat("a", 63)}
	for _, value := range accepted {
		require.Emptyf(t, validate(value, cty.Path{}), "value %q must be accepted", value)
	}

	rejected := []string{"", "ab", "my bucket", "my.bucket", strings.Repeat("a", 64)}
	for _, value := range rejected {
		require.NotEmptyf(t, validate(value, cty.Path{}), "value %q must be rejected", value)
	}
}

func TestStorageS3DataSourceSchema(t *testing.T) {
	t.Parallel()

	ds := dataSourceStorageS3()

	require.NotNil(t, ds.ReadContext)
	require.Nil(t, ds.CreateContext)
	require.Nil(t, ds.DeleteContext)

	atLeastOne := []string{StorageSchemaID, StorageSchemaName}
	require.Equal(t, atLeastOne, ds.Schema[StorageSchemaID].AtLeastOneOf)
	require.Equal(t, atLeastOne, ds.Schema[StorageSchemaName].AtLeastOneOf)

	require.True(t, ds.Schema[StorageSchemaID].Optional)
	require.True(t, ds.Schema[StorageSchemaName].Optional)

	for _, key := range []string{
		StorageSchemaClientID,
		StorageSchemaLocation,
		StorageSchemaGenerateHTTPEndpoint,
		StorageSchemaGenerateS3Endpoint,
		StorageSchemaGenerateEndpoint,
	} {
		require.Truef(t, ds.Schema[key].Computed, "attribute %q must stay computed", key)
	}

	require.NotContains(t, ds.Schema, StorageS3SchemaGenerateAccessKey)
	require.NotContains(t, ds.Schema, StorageS3SchemaGenerateSecretKey)
}

func TestStorageS3BucketDataSourceSchema(t *testing.T) {
	t.Parallel()

	ds := dataSourceStorageS3Bucket()

	require.NotNil(t, ds.ReadContext)
	require.True(t, ds.Schema[StorageS3BucketSchemaStorageID].Required)
	require.True(t, ds.Schema[StorageS3BucketSchemaName].Required)
	require.False(t, ds.Schema[StorageS3BucketSchemaStorageID].ForceNew)
	require.False(t, ds.Schema[StorageS3BucketSchemaName].ForceNew)
}

func TestStorageResourceID(t *testing.T) {
	t.Parallel()

	res := resourceStorageS3()

	cases := []struct {
		name string
		raw  map[string]interface{}
		id   string
		want string
	}{
		{name: "id wins over the schema", raw: map[string]interface{}{StorageSchemaID: 9}, id: "7", want: "7"},
		{name: "falls back to storage_id", raw: map[string]interface{}{StorageSchemaID: 9}, want: "9"},
		{name: "zero storage_id is not an id", raw: map[string]interface{}{StorageSchemaID: 0}, want: ""},
		{name: "nothing to fall back to", raw: map[string]interface{}{}, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := schema.TestResourceDataRaw(t, res.Schema, tc.raw)
			data.SetId(tc.id)

			require.Equal(t, tc.want, storageResourceID(data))
		})
	}
}

func TestStorageBucketResourceID(t *testing.T) {
	t.Parallel()

	res := resourceStorageS3Bucket()

	cases := []struct {
		name      string
		raw       map[string]interface{}
		id        string
		wantID    int
		wantBuck  string
		wantEmpty bool
	}{
		{
			name:     "composite id is split",
			id:       "7:mybucket",
			wantID:   7,
			wantBuck: "mybucket",
		},
		{
			name: "falls back to the schema and trims the name",
			raw: map[string]interface{}{
				StorageS3BucketSchemaStorageID: 7,
				StorageS3BucketSchemaName:      "  mybucket  ",
			},
			wantID:   7,
			wantBuck: "mybucket",
		},
		{
			name:      "an id with three parts yields nothing",
			id:        "7:my:bucket",
			wantEmpty: true,
		},
		{
			name:      "an id without a separator yields nothing",
			id:        "7",
			wantEmpty: true,
		},
		{
			name:     "a non numeric storage id becomes zero",
			id:       "seven:mybucket",
			wantID:   0,
			wantBuck: "mybucket",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := schema.TestResourceDataRaw(t, res.Schema, tc.raw)
			data.SetId(tc.id)

			gotID, gotBucket := storageBucketResourceID(data)
			if tc.wantEmpty {
				require.Zero(t, gotID)
				require.Empty(t, gotBucket)
				return
			}

			require.Equal(t, tc.wantID, gotID)
			require.Equal(t, tc.wantBuck, gotBucket)
		})
	}
}
