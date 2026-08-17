package mkaas

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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

func TestServiceRegistersEveryMKaaSName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "mkaas", Service{}.Name())

	require.Equal(t, "edgecenter_mkaas_cluster", MKaaSClusterResource)
	require.Equal(t, "edgecenter_mkaas_pool", MKaaSPoolResource)

	require.Equal(t, "edgecenter_mkaas_cluster", MKaaSClusterDataSource)
	require.Equal(t, "edgecenter_mkaas_pool", MKaaSPoolDataSource)
	require.Equal(t, "edgecenter_k8s", K8sDataSource)
	require.Equal(t, "edgecenter_k8s_pool", K8sPoolDataSource)
	require.Equal(t, "edgecenter_k8s_client_config", K8sClientConfigDataSource)

	resources := Service{}.Resources()
	require.Len(t, resources, 2)
	for _, name := range []string{MKaaSClusterResource, MKaaSPoolResource} {
		require.Containsf(t, resources, name, "resource %q is not registered", name)
	}

	dataSources := Service{}.DataSources()
	require.Len(t, dataSources, 5)
	for _, name := range []string{
		MKaaSClusterDataSource,
		MKaaSPoolDataSource,
		K8sDataSource,
		K8sPoolDataSource,
		K8sClientConfigDataSource,
	} {
		require.Containsf(t, dataSources, name, "data source %q is not registered", name)
	}
}

func TestEveryMKaaSResourcePassesInternalValidate(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).Resources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, res.InternalValidate(nil, true))
		})
	}
}

func TestEveryMKaaSDataSourcePassesInternalValidate(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).DataSources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, res.InternalValidate(nil, false))
		})
	}
}

func TestEveryMKaaSResourceCarriesAnImporter(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).Resources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, res.Importer, "resource must be importable")
			require.NotNil(t, res.Importer.StateContext)
			require.NotNil(t, res.CreateContext)
			require.NotNil(t, res.ReadContext)
			require.NotNil(t, res.UpdateContext)
			require.NotNil(t, res.DeleteContext)
		})
	}
}

func TestNoMKaaSDataSourceCarriesAnImporter(t *testing.T) {
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

	res := resourceMKaaSCluster()

	requireProjectRegionPair(t, res, false)

	name := res.Schema[edgecenter.NameField]
	require.True(t, name.Required)
	require.False(t, name.ForceNew)
	require.NotNil(t, name.ValidateFunc)

	for _, key := range []string{
		edgecenter.MKaaSClusterKeypairNameField,
		edgecenter.NetworkIDField,
		edgecenter.SubnetIDField,
	} {
		require.Truef(t, res.Schema[key].Required, "attribute %q must be required", key)
		require.Falsef(t, res.Schema[key].ForceNew, "attribute %q force new flag", key)
	}

	require.True(t, res.Schema[edgecenter.MKaaSClusterPublishKubeAPIToInternet].Optional)
	require.Equal(t, schema.TypeBool, res.Schema[edgecenter.MKaaSClusterPublishKubeAPIToInternet].Type)

	for _, key := range []string{
		edgecenter.MKaaSClusterPodSubnetField,
		edgecenter.MKaaSClusterServiceSubnetField,
	} {
		field := res.Schema[key]
		require.Truef(t, field.Required, "attribute %q must be required", key)
		require.Truef(t, field.ForceNew, "attribute %q must be force new", key)
		require.NotNilf(t, field.ValidateDiagFunc, "attribute %q must validate its cidr", key)
	}

	cp := res.Schema[edgecenter.MKaaSClusterControlPlaneField]
	require.True(t, cp.Required)
	require.False(t, cp.ForceNew)
	require.Equal(t, schema.TypeList, cp.Type)
	require.Equal(t, 1, cp.MaxItems)

	for _, key := range []string{
		edgecenter.FlavorField,
		edgecenter.MKaaSNodeCountField,
		edgecenter.MKaaSVolumeSizeField,
		edgecenter.MKaaSVolumeTypeField,
		edgecenter.MKaaSClusterVersionField,
	} {
		require.Truef(t, nestedField(t, cp, key).Required, "control plane %q must be required", key)
	}

	require.NotNil(t, nestedField(t, cp, edgecenter.MKaaSNodeCountField).ValidateFunc)
	require.NotNil(t, nestedField(t, cp, edgecenter.MKaaSVolumeSizeField).ValidateFunc)
	require.NotNil(t, nestedField(t, cp, edgecenter.MKaaSVolumeTypeField).ValidateFunc)
	require.Nil(t, nestedField(t, cp, edgecenter.MKaaSClusterVersionField).ValidateFunc)

	requireComputedOnly(t, res,
		edgecenter.MKaaSClusterInternalIPField,
		edgecenter.MKaaSClusterExternalIPField,
		edgecenter.MKaaSClusterCreatedField,
		edgecenter.MKaaSClusterProcessingField,
		edgecenter.StatusField,
		edgecenter.MKaaSClusterStageField,
	)
}

func TestPoolSchemaFlags(t *testing.T) {
	t.Parallel()

	res := resourceMKaaSPool()

	requireProjectRegionPair(t, res, false)

	clusterID := res.Schema[edgecenter.MKaaSClusterIDField]
	require.Equal(t, schema.TypeInt, clusterID.Type)
	require.True(t, clusterID.Required)
	require.True(t, clusterID.ForceNew)

	for _, key := range []string{
		edgecenter.NameField,
		edgecenter.FlavorField,
		edgecenter.MKaaSVolumeSizeField,
		edgecenter.MKaaSVolumeTypeField,
	} {
		require.Truef(t, res.Schema[key].Required, "attribute %q must be required", key)
		require.Falsef(t, res.Schema[key].ForceNew, "attribute %q force new flag", key)
	}

	require.NotNil(t, res.Schema[edgecenter.MKaaSVolumeSizeField].ValidateFunc)
	require.NotNil(t, res.Schema[edgecenter.MKaaSVolumeTypeField].ValidateFunc)

	nodeCount := res.Schema[edgecenter.MKaaSNodeCountField]
	require.True(t, nodeCount.Optional)
	require.True(t, nodeCount.Computed)
	require.Equal(t,
		[]string{edgecenter.MKaaSNodeCountField, edgecenter.MKaaSPoolScalePolicyField},
		nodeCount.ExactlyOneOf)

	requireComputedOnly(t, res,
		edgecenter.MKaaSPoolCurrentNodeCountField,
		edgecenter.MKaaSPoolStateField,
		edgecenter.MKaaSPoolStatusField,
	)

	securityGroups := res.Schema[edgecenter.MKaaSPoolSecurityGroupIDsField]
	require.Equal(t, schema.TypeList, securityGroups.Type)
	require.True(t, securityGroups.Optional)

	labels := res.Schema[edgecenter.MKaaSPoolLabelsField]
	require.Equal(t, schema.TypeMap, labels.Type)
	require.True(t, labels.Optional)

	taints := res.Schema[edgecenter.MKaaSPoolTaintsField]
	require.Equal(t, schema.TypeSet, taints.Type)
	require.True(t, taints.Optional)
	for _, key := range []string{"key", "value", "effect"} {
		require.Truef(t, nestedField(t, taints, key).Required, "taint %q must be required", key)
	}
	require.NotNil(t, nestedField(t, taints, "effect").ValidateFunc)

	scalePolicy := res.Schema[edgecenter.MKaaSPoolScalePolicyField]
	require.Equal(t, schema.TypeList, scalePolicy.Type)
	require.True(t, scalePolicy.Optional)
	require.Equal(t, 1, scalePolicy.MaxItems)
	require.Equal(t,
		[]string{edgecenter.MKaaSNodeCountField, edgecenter.MKaaSPoolScalePolicyField},
		scalePolicy.ExactlyOneOf)

	autoScale := nestedField(t, scalePolicy, edgecenter.MKaaSPoolAutoScaleField)
	require.True(t, autoScale.Required)
	require.Equal(t, 1, autoScale.MaxItems)
	for _, key := range []string{
		edgecenter.MKaaSPoolMinNodeCountField,
		edgecenter.MKaaSPoolMaxNodeCountField,
	} {
		nested := nestedField(t, autoScale, key)
		require.Truef(t, nested.Required, "auto scale %q must be required", key)
		require.NotNilf(t, nested.ValidateFunc, "auto scale %q must be validated", key)
	}
}

func TestClusterDescriptions(t *testing.T) {
	t.Parallel()

	require.Equal(t, "Represent resourceMKaaSCluster cluster.", resourceMKaaSCluster().Description,
		"the go constructor name is published into the generated docs")
	require.Equal(t, "Represent MKaaS cluster's pool.", resourceMKaaSPool().Description)
	require.Empty(t, dataSourceMKaaSCluster().Description)
	require.Equal(t, "Represent MKaaS cluster's pool.", dataSourceMKaaSPool().Description)
}

func TestDeprecatedK8sStubsRefuseEveryRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		res         *schema.Resource
		deprecation string
		description string
		message     string
	}{
		{
			name:        K8sDataSource,
			res:         dataSourceK8s(),
			deprecation: `!> **WARNING:** This data source is deprecated and will be removed in the next major version. Data source "edgecenter_k8s" unavailable.`,
			description: "Represent k8s cluster with one default pool.\n\n **WARNING:** Data source \"edgecenter_k8s\" is deprecated and unavailable.",
			message:     `resource "edgecenter_k8s" is deprecated and unavailable`,
		},
		{
			name:        K8sPoolDataSource,
			res:         dataSourceK8sPool(),
			deprecation: `!> **WARNING:** This data source is deprecated and will be removed in the next major version. Data source "edgecenter_k8s_pool" unavailable.`,
			description: "Represent k8s cluster's pool.\n\n **WARNING:** Data source \"edgecenter_k8s_pool\" is deprecated and unavailable.",
			message:     `data source "edgecenter_k8s_pool" is deprecated and unavailable`,
		},
		{
			name:        K8sClientConfigDataSource,
			res:         dataSourceK8sClientConfig(),
			deprecation: `!> **WARNING:** This data source is deprecated and will be removed in the next major version. Data source "edgecenter_k8s_client_config" unavailable.`,
			description: "Represent k8s cluster with one default pool. \n\n **WARNING:** Data source \"edgecenter_k8s_client_config\" is deprecated and unavailable.",
			message:     `data source "edgecenter_k8s_client_config" is deprecated and unavailable`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.deprecation, tc.res.DeprecationMessage)
			require.Equal(t, tc.description, tc.res.Description)

			diags := tc.res.ReadContext(t.Context(), tc.res.Data(nil), nil)
			require.Len(t, diags, 1)
			require.Equal(t, diag.Error, diags[0].Severity)
			require.Equal(t, tc.message, diags[0].Summary)

			requireScopePair(t, tc.res)
			require.True(t, tc.res.Schema["cluster_id"].Required)
		})
	}
}

func requireScopePair(t *testing.T, res *schema.Resource) {
	t.Helper()

	for _, pair := range [][2]string{
		{edgecenter.ProjectIDField, edgecenter.ProjectNameField},
		{edgecenter.RegionIDField, edgecenter.RegionNameField},
	} {
		for _, key := range pair {
			field := res.Schema[key]
			require.NotNilf(t, field, "attribute %q is missing", key)
			require.Truef(t, field.Optional, "attribute %q must be optional", key)
			require.Equalf(t, []string{pair[0], pair[1]}, field.ExactlyOneOf,
				"attribute %q must be exactly one of the identity pair", key)
		}
	}
}

func TestClusterTimeouts(t *testing.T) {
	t.Parallel()

	res := resourceMKaaSCluster()

	require.NotNil(t, res.Timeouts)
	require.Equal(t, 30*time.Minute, *res.Timeouts.Create)
	require.Equal(t, 10*time.Minute, *res.Timeouts.Read)
	require.Equal(t, 30*time.Minute, *res.Timeouts.Update)
	require.Equal(t, 20*time.Minute, *res.Timeouts.Delete)
}

func TestPoolTimeouts(t *testing.T) {
	t.Parallel()

	res := resourceMKaaSPool()

	require.NotNil(t, res.Timeouts)
	require.Equal(t, 60*time.Minute, *res.Timeouts.Create)
	require.Equal(t, 10*time.Minute, *res.Timeouts.Read)
	require.Equal(t, 60*time.Minute, *res.Timeouts.Update)
	require.Equal(t, 20*time.Minute, *res.Timeouts.Delete)
}
