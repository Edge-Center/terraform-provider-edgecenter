package dbaas

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

func nestedResource(t *testing.T, field *schema.Schema) *schema.Resource {
	t.Helper()

	require.NotNil(t, field)

	elem, ok := field.Elem.(*schema.Resource)
	require.Truef(t, ok, "elem is %T, want *schema.Resource", field.Elem)

	return elem
}

func nestedField(t *testing.T, field *schema.Schema, key string) *schema.Schema {
	t.Helper()

	nested := nestedResource(t, field).Schema[key]
	require.NotNilf(t, nested, "nested field %q is missing", key)

	return nested
}

func requireComputedOnly(t *testing.T, res *schema.Resource, keys ...string) {
	t.Helper()

	for _, key := range keys {
		field := res.Schema[key]
		require.NotNilf(t, field, "attribute %q is missing", key)
		require.Truef(t, field.Computed, "attribute %q must stay computed", key)
		require.Falsef(t, field.Required, "attribute %q must stay read only", key)
		require.Falsef(t, field.Optional, "attribute %q must stay read only", key)
	}
}

func requireProjectRegionPair(t *testing.T, res *schema.Resource, forceNew bool) {
	t.Helper()

	pairs := [][2]string{
		{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
		{edgecenter.RegionIDField, edgecenter.RegionNameField},
	}

	for _, pair := range pairs {
		for _, key := range pair {
			field := res.Schema[key]
			require.NotNilf(t, field, "attribute %q is missing", key)
			require.Truef(t, field.Optional, "attribute %q must be optional", key)
			require.Equalf(t, []string{pair[0], pair[1]}, field.ExactlyOneOf,
				"attribute %q must be exactly one of the identity pair", key)
			require.Equalf(t, forceNew, field.ForceNew, "attribute %q force new flag", key)
		}
	}
}

func TestServiceRegistersEveryDBaaSName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "dbaas", Service{}.Name())

	require.Equal(t, "edgecenter_dbaas_cluster", DBaaSClusterResource)
	require.Equal(t, "edgecenter_dbaas_database", DBaaSDatabaseResource)
	require.Equal(t, "edgecenter_dbaas_user", DBaaSUserResource)
	require.Equal(t, "edgecenter_dbaas_backup", DBaaSBackupResource)

	require.Equal(t, "edgecenter_dbaas_dbms", DBaaSDbmsDataSource)
	require.Equal(t, "edgecenter_dbaas_clusters", DBaaSClustersDataSource)
	require.Equal(t, "edgecenter_dbaas_databases", DBaaSDatabasesDataSource)
	require.Equal(t, "edgecenter_dbaas_users", DBaaSUsersDataSource)
	require.Equal(t, "edgecenter_dbaas_backup", DBaaSBackupDataSource)

	resources := Service{}.Resources()
	require.Len(t, resources, 4)
	for _, name := range []string{
		DBaaSClusterResource,
		DBaaSDatabaseResource,
		DBaaSUserResource,
		DBaaSBackupResource,
	} {
		require.Containsf(t, resources, name, "resource %q is not registered", name)
	}

	dataSources := Service{}.DataSources()
	require.Len(t, dataSources, 5)
	for _, name := range []string{
		DBaaSDbmsDataSource,
		DBaaSClustersDataSource,
		DBaaSDatabasesDataSource,
		DBaaSUsersDataSource,
		DBaaSBackupDataSource,
	} {
		require.Containsf(t, dataSources, name, "data source %q is not registered", name)
	}
}

func TestEveryDBaaSResourcePassesInternalValidate(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).Resources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, res.InternalValidate(nil, true))
		})
	}
}

func TestEveryDBaaSDataSourcePassesInternalValidate(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).DataSources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, res.InternalValidate(nil, false))
		})
	}
}

func TestEveryDBaaSResourceCarriesAnImporter(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).Resources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, res.Importer, "resource must be importable")
			require.NotNil(t, res.Importer.StateContext)
		})
	}
}

func TestNoDBaaSDataSourceCarriesAnImporter(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).DataSources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Nil(t, res.Importer, "a data source must not declare an importer")
			require.Nil(t, res.CreateContext)
			require.Nil(t, res.UpdateContext)
			require.Nil(t, res.DeleteContext)
			require.NotNil(t, res.ReadContext)
		})
	}
}

func TestClusterSchemaFlags(t *testing.T) {
	t.Parallel()

	res := resourceDBaaSCluster()

	requireProjectRegionPair(t, res, false)

	require.True(t, res.Schema[edgecenter.NameField].Required)
	require.False(t, res.Schema[edgecenter.NameField].ForceNew)
	require.True(t, res.Schema[edgecenter.FlavorField].Required)
	require.False(t, res.Schema[edgecenter.FlavorField].ForceNew)
	require.True(t, res.Schema[edgecenter.DescriptionField].Optional)

	ha := res.Schema[edgecenter.DBaaSClusterHighAvailabilityField]
	require.True(t, ha.Optional)
	require.True(t, ha.ForceNew)
	require.Equal(t, true, ha.Default)

	dbms := res.Schema["dbms"]
	require.True(t, dbms.Required)
	require.True(t, dbms.ForceNew)
	require.Equal(t, 1, dbms.MaxItems)
	require.True(t, nestedField(t, dbms, edgecenter.TypeField).Required)
	require.True(t, nestedField(t, dbms, edgecenter.DBaaSDbmsVersionField).Required)

	volume := res.Schema["volume"]
	require.True(t, volume.Required)
	require.False(t, volume.ForceNew)
	require.Equal(t, 1, volume.MaxItems)
	require.True(t, nestedField(t, volume, edgecenter.DBaaSVolumeSizeField).Required)
	require.True(t, nestedField(t, volume, edgecenter.DBaaSVolumeTypeField).Required)

	iface := res.Schema["interface"]
	require.True(t, iface.Required)
	require.True(t, iface.ForceNew)
	require.Equal(t, 1, iface.MaxItems)
	require.True(t, nestedField(t, iface, edgecenter.NetworkIDField).Required)
	require.True(t, nestedField(t, iface, edgecenter.SubnetIDField).Required)

	requireComputedOnly(t, res,
		edgecenter.StatusField,
		edgecenter.CreatedAtField,
		edgecenter.UpdatedAtField,
		edgecenter.DBaaSClusterTaskIDField,
		edgecenter.DBaaSClusterConnectionField,
	)

	conn := res.Schema[edgecenter.DBaaSClusterConnectionField]
	require.True(t, nestedField(t, conn, edgecenter.DBaaSClusterHostField).Computed)
	require.Equal(t, schema.TypeInt, nestedField(t, conn, edgecenter.DBaaSClusterPortField).Type)
}

func TestClusterTimeouts(t *testing.T) {
	t.Parallel()

	res := resourceDBaaSCluster()

	require.NotNil(t, res.Timeouts)
	require.Equal(t, 30*time.Minute, *res.Timeouts.Create)
	require.Equal(t, 10*time.Minute, *res.Timeouts.Read)
	require.Equal(t, 30*time.Minute, *res.Timeouts.Update)
	require.Equal(t, 20*time.Minute, *res.Timeouts.Delete)
}

func TestDatabaseSchemaFlags(t *testing.T) {
	t.Parallel()

	res := resourceDBaaSDatabase()

	requireProjectRegionPair(t, res, true)

	require.Nil(t, res.UpdateContext, "every database attribute is force new, so update is unreachable")

	for _, key := range []string{
		edgecenter.DBaaSClusterIDField,
		edgecenter.NameField,
		edgecenter.DBaaSDatabaseEncodingField,
		edgecenter.DBaaSDatabaseLocaleField,
	} {
		require.Truef(t, res.Schema[key].ForceNew, "attribute %q must be force new", key)
	}

	require.True(t, res.Schema[edgecenter.DBaaSClusterIDField].Required)
	require.True(t, res.Schema[edgecenter.NameField].Required)
	require.True(t, res.Schema[edgecenter.DBaaSDatabaseEncodingField].Optional)
	require.True(t, res.Schema[edgecenter.DBaaSDatabaseLocaleField].Optional)
}

func TestUserSchemaFlags(t *testing.T) {
	t.Parallel()

	res := resourceDBaaSUser()

	requireProjectRegionPair(t, res, true)

	require.True(t, res.Schema[edgecenter.DBaaSClusterIDField].Required)
	require.True(t, res.Schema[edgecenter.DBaaSClusterIDField].ForceNew)
	require.True(t, res.Schema[edgecenter.NameField].Required)
	require.True(t, res.Schema[edgecenter.NameField].ForceNew)

	password := res.Schema[edgecenter.PasswordField]
	require.True(t, password.Required)
	require.True(t, password.Sensitive)
	require.False(t, password.ForceNew)

	databases := res.Schema[edgecenter.DBaaSUserDatabasesField]
	require.True(t, databases.Optional)
	require.False(t, databases.ForceNew)
	require.Equal(t, schema.TypeList, databases.Type,
		"the api collection is unordered, so an ordered list makes a reorder a diff that never converges")
}

func TestOnlyTheClusterLeavesTheScopeMutable(t *testing.T) {
	t.Parallel()

	scope := []string{
		edgecenter.ProjectIDField,
		edgecenter.ProjectNameField,
		edgecenter.RegionIDField,
		edgecenter.RegionNameField,
	}

	for _, key := range scope {
		require.Falsef(t, resourceDBaaSCluster().Schema[key].ForceNew,
			"cluster attribute %q is mutable although a moved cluster cannot be patched in place", key)

		for name, res := range map[string]*schema.Resource{
			DBaaSDatabaseResource: resourceDBaaSDatabase(),
			DBaaSUserResource:     resourceDBaaSUser(),
			DBaaSBackupResource:   resourceDBaaSBackup(),
		} {
			require.Truef(t, res.Schema[key].ForceNew, "%s attribute %q must be force new", name, key)
		}
	}

	for _, res := range []*schema.Resource{
		resourceDBaaSCluster(),
		resourceDBaaSDatabase(),
		resourceDBaaSUser(),
		resourceDBaaSBackup(),
	} {
		for _, key := range scope {
			require.Falsef(t, res.Schema[key].Computed,
				"attribute %q is written back by read but not computed, so a name based config never converges", key)
		}
	}
}

func TestBackupSchemaFlags(t *testing.T) {
	t.Parallel()

	res := resourceDBaaSBackup()

	requireProjectRegionPair(t, res, true)

	require.True(t, res.Schema[edgecenter.NameField].Required)
	require.False(t, res.Schema[edgecenter.NameField].ForceNew)
	require.True(t, res.Schema[edgecenter.DescriptionField].Optional)
	require.True(t, res.Schema[edgecenter.DBaaSClusterIDField].Required)
	require.True(t, res.Schema[edgecenter.DBaaSClusterIDField].ForceNew)
	require.True(t, res.Schema[edgecenter.DBaaSBackupParentIDField].Optional)
	require.True(t, res.Schema[edgecenter.DBaaSBackupParentIDField].ForceNew)

	requireComputedOnly(t, res,
		edgecenter.DBaaSBackupTypeField,
		edgecenter.StatusField,
		edgecenter.DBaaSBackupSizeField,
		edgecenter.DBaaSBackupIsServiceField,
		edgecenter.DBaaSBackupHasChildField,
		edgecenter.CreatedAtField,
		edgecenter.UpdatedAtField,
		edgecenter.DBaaSBackupFinishedAtField,
		edgecenter.DBaaSClusterTaskIDField,
		edgecenter.DBaaSBackupCreatorTaskIDField,
		"dbms",
	)

	require.Equal(t, schema.TypeFloat, res.Schema[edgecenter.DBaaSBackupSizeField].Type)
	require.Equal(t, schema.TypeBool, res.Schema[edgecenter.DBaaSBackupIsServiceField].Type)
	require.Equal(t, schema.TypeBool, res.Schema[edgecenter.DBaaSBackupHasChildField].Type)
}

func TestBackupTimeouts(t *testing.T) {
	t.Parallel()

	res := resourceDBaaSBackup()

	require.NotNil(t, res.Timeouts)
	require.Equal(t, 30*time.Minute, *res.Timeouts.Create)
	require.Equal(t, 10*time.Minute, *res.Timeouts.Read)
	require.Equal(t, 10*time.Minute, *res.Timeouts.Update)
	require.Equal(t, 20*time.Minute, *res.Timeouts.Delete)
}

func TestBackupDataSourceLookupPair(t *testing.T) {
	t.Parallel()

	res := dataSourceDBaaSBackup()

	for _, key := range []string{edgecenter.IDField, edgecenter.NameField} {
		field := res.Schema[key]
		require.NotNilf(t, field, "attribute %q is missing", key)
		require.True(t, field.Optional)
		require.False(t, field.Computed, "a declared lookup key that is not computed plans as null on a deferred read")
		require.Equal(t, []string{edgecenter.IDField, edgecenter.NameField}, field.ExactlyOneOf)
		require.Equal(t, []string{edgecenter.IDField, edgecenter.NameField}, field.AtLeastOneOf)
	}

	for _, key := range []string{edgecenter.ProjectIDField, edgecenter.RegionIDField} {
		require.Falsef(t, res.Schema[key].Computed, "attribute %q is written by read but not computed", key)
	}
}

func TestClustersDataSourceLookupPair(t *testing.T) {
	t.Parallel()

	res := dataSourceDBaaSClusters()

	for _, key := range []string{edgecenter.IDField, edgecenter.NameField} {
		field := res.Schema[key]
		require.NotNilf(t, field, "attribute %q is missing", key)
		require.True(t, field.Optional)
		require.False(t, field.Computed, "a declared lookup key that is not computed plans as null on a deferred read")
		require.Equal(t, []string{edgecenter.IDField, edgecenter.NameField}, field.ExactlyOneOf)
		require.Empty(t, field.AtLeastOneOf)
	}
}

func TestListDataSourcesRequireACluster(t *testing.T) {
	t.Parallel()

	for name, res := range map[string]*schema.Resource{
		DBaaSDatabasesDataSource: dataSourceDBaaSDatabases(),
		DBaaSUsersDataSource:     dataSourceDBaaSUsers(),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.True(t, res.Schema[edgecenter.DBaaSClusterIDField].Required)
			require.True(t, res.Schema[edgecenter.NameField].Optional)

			items := res.Schema["items"]
			require.True(t, items.Computed)
			require.True(t, nestedField(t, items, edgecenter.NameField).Computed)
		})
	}
}

func TestDbmsDataSourceItems(t *testing.T) {
	t.Parallel()

	res := dataSourceDBaaSDBMS()

	items := res.Schema["items"]
	require.True(t, items.Computed)
	require.Equal(t, schema.TypeInt, nestedField(t, items, edgecenter.IDField).Type)
	require.Equal(t, schema.TypeString, nestedField(t, items, edgecenter.TypeField).Type)
	require.Equal(t, schema.TypeString, nestedField(t, items, edgecenter.DBaaSDbmsVersionField).Type)
}

func TestComputedStringSchemaIsAFreshValueEveryCall(t *testing.T) {
	t.Parallel()

	first := computedStringSchema()
	second := computedStringSchema()

	require.NotSame(t, first, second, "a shared pointer would let one resource mutate another")
	require.Equal(t, schema.TypeString, first.Type)
	require.True(t, first.Computed)
	require.False(t, first.Optional)
	require.False(t, first.Required)
}
