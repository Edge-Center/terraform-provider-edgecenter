package reseller

import (
	"context"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

const (
	schemaEntityID    = 4242
	schemaEntityIDStr = "4242"
	schemaEntityType  = "reseller"
	schemaRegionID    = 8
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

func requireSameSchemaMap(t *testing.T, want, got map[string]*schema.Schema) {
	t.Helper()

	require.Len(t, got, len(want))
	for key, field := range want {
		require.Samef(t, field, got[key], "field %q is not the exported one", key)
	}
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

func TestServiceRegistersEveryResellerName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "reseller", Service{}.Name())

	require.Equal(t, "edgecenter_reseller_images", ResellerImagesResource)
	require.Equal(t, "edgecenter_reseller_images", ResellerImagesDataSource)
	require.Equal(t, "edgecenter_reseller_imagesV2", ResellerImagesV2Resource)
	require.Equal(t, "edgecenter_reseller_imagesV2", ResellerImagesV2DataSource)
	require.Equal(t, "edgecenter_reseller_networks", ResellerNetworksDataSource)

	resources := Service{}.Resources()
	require.Len(t, resources, 2)
	for _, name := range []string{ResellerImagesResource, ResellerImagesV2Resource} {
		require.Containsf(t, resources, name, "resource %q is not registered", name)
	}

	dataSources := Service{}.DataSources()
	require.Len(t, dataSources, 3)
	for _, name := range []string{
		ResellerImagesDataSource,
		ResellerNetworksDataSource,
		ResellerImagesV2DataSource,
	} {
		require.Containsf(t, dataSources, name, "data source %q is not registered", name)
	}

	require.Contains(t, resources[ResellerImagesResource].Schema, edgecenter.ResellerIDField)
	require.Contains(t, resources[ResellerImagesV2Resource].Schema, edgecenter.EntityIDField)
	require.Contains(t, dataSources[ResellerImagesDataSource].Schema, edgecenter.ResellerIDField)
	require.Contains(t, dataSources[ResellerImagesV2DataSource].Schema, edgecenter.EntityIDField)
	require.Contains(t, dataSources[ResellerNetworksDataSource].Schema, edgecenter.NetworksField)
}

func TestEveryResellerResourcePassesInternalValidate(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).Resources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, res.InternalValidate(nil, true))
		})
	}
}

func TestEveryResellerDataSourcePassesInternalValidate(t *testing.T) {
	t.Parallel()

	for name, res := range (Service{}).DataSources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, res.InternalValidate(nil, false))
		})
	}
}

func TestResellerImagesV2ResourceSchema(t *testing.T) {
	t.Parallel()

	res := resourceResellerImagesV2()

	require.NotNil(t, res.Importer)
	require.NotNil(t, res.CreateContext)
	require.NotNil(t, res.ReadContext)
	require.NotNil(t, res.UpdateContext)
	require.NotNil(t, res.DeleteContext)
	require.NotNil(t, res.CustomizeDiff)
	require.Empty(t, res.DeprecationMessage)

	entityID := res.Schema[edgecenter.EntityIDField]
	require.Equal(t, schema.TypeInt, entityID.Type)
	require.True(t, entityID.Required)
	require.True(t, entityID.ForceNew)

	entityType := res.Schema[edgecenter.EntityTypeField]
	require.Equal(t, schema.TypeString, entityType.Type)
	require.True(t, entityType.Required)
	require.True(t, entityType.ForceNew)
	require.NotNil(t, entityType.ValidateFunc)

	options := res.Schema[edgecenter.ResellerImagesOptionsField]
	require.Equal(t, schema.TypeSet, options.Type)
	require.True(t, options.Required)
	require.False(t, options.ForceNew)

	require.Len(t, res.Schema, 3)
}

func TestResellerImagesV2OptionsSchema(t *testing.T) {
	t.Parallel()

	options := resourceResellerImagesV2().Schema[edgecenter.ResellerImagesOptionsField]

	requireSameSchemaMap(t, ResellerImageV2, nestedResource(t, options).Schema)

	regionID := nestedField(t, options, edgecenter.RegionIDField)
	require.Equal(t, schema.TypeInt, regionID.Type)
	require.True(t, regionID.Required)
	require.False(t, regionID.ForceNew)

	imageIDs := nestedField(t, options, edgecenter.ImageIDsField)
	require.Equal(t, schema.TypeSet, imageIDs.Type)
	require.True(t, imageIDs.Optional)
	require.False(t, imageIDs.Computed)

	imageIDsElem, ok := imageIDs.Elem.(*schema.Schema)
	require.True(t, ok)
	require.Equal(t, schema.TypeString, imageIDsElem.Type)

	allPublic := nestedField(t, options, edgecenter.AllPublicImagesAreAvailableField)
	require.Equal(t, schema.TypeBool, allPublic.Type)
	require.True(t, allPublic.Optional)
	require.False(t, allPublic.Computed)

	for _, key := range []string{edgecenter.CreatedAtField, edgecenter.UpdatedAtField} {
		field := nestedField(t, options, key)
		require.Equalf(t, schema.TypeString, field.Type, "attribute %q must be a string", key)
		require.Truef(t, field.Computed, "attribute %q must stay computed", key)
		require.Falsef(t, field.Optional, "attribute %q must stay read only", key)
	}
}

func TestResellerImagesV2EntityTypeValidation(t *testing.T) {
	t.Parallel()

	validators := map[string]func(interface{}, string) ([]string, []error){
		"resource":    resourceResellerImagesV2().Schema[edgecenter.EntityTypeField].ValidateFunc,
		"data source": dataSourceResellerImagesV2().Schema[edgecenter.EntityTypeField].ValidateFunc,
	}

	for where, validate := range validators {
		t.Run(where, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, validate)

			for _, entityType := range []string{
				edgecloudV2.ResellerType,
				edgecloudV2.ClientType,
				edgecloudV2.ProjectType,
			} {
				_, errs := validate(entityType, edgecenter.EntityTypeField)
				require.Emptyf(t, errs, "%q must be accepted", entityType)
			}

			for _, entityType := range []string{"", "Reseller", "resellers", " project", "admin"} {
				_, errs := validate(entityType, edgecenter.EntityTypeField)
				require.NotEmptyf(t, errs, "%q must be rejected", entityType)
			}
		})
	}
}

func TestResellerImagesV2DataSourceSchema(t *testing.T) {
	t.Parallel()

	ds := dataSourceResellerImagesV2()

	require.NotNil(t, ds.ReadContext)
	require.Nil(t, ds.CreateContext)
	require.Nil(t, ds.UpdateContext)
	require.Nil(t, ds.DeleteContext)
	require.Empty(t, ds.DeprecationMessage)

	require.True(t, ds.Schema[edgecenter.EntityIDField].Required)
	require.False(t, ds.Schema[edgecenter.EntityIDField].ForceNew)
	require.True(t, ds.Schema[edgecenter.EntityTypeField].Required)
	require.False(t, ds.Schema[edgecenter.EntityTypeField].ForceNew)

	options := ds.Schema[edgecenter.ResellerImagesOptionsField]
	require.Equal(t, schema.TypeSet, options.Type)
	require.True(t, options.Computed)
	require.False(t, options.Required)

	requireComputedOnly(t, nestedResource(t, options),
		edgecenter.RegionIDField,
		edgecenter.ImageIDsField,
		edgecenter.AllPublicImagesAreAvailableField,
		edgecenter.CreatedAtField,
		edgecenter.UpdatedAtField,
	)

	require.Len(t, nestedResource(t, options).Schema, 5)
}

func TestResellerImagesV1IsDeprecated(t *testing.T) {
	t.Parallel()

	res := resourceResellerImages()
	require.NotEmpty(t, res.DeprecationMessage)
	require.Contains(t, res.DeprecationMessage, ResellerImagesV2Resource)

	ds := dataSourceResellerImages()
	require.NotEmpty(t, ds.DeprecationMessage)
	require.Contains(t, ds.DeprecationMessage, ResellerImagesV2DataSource)
}

func TestResellerImagesV1OptionsHaveNoAllPublicImagesFlag(t *testing.T) {
	t.Parallel()

	require.NotContains(t, ResellerImage, edgecenter.AllPublicImagesAreAvailableField)
	require.Contains(t, ResellerImageV2, edgecenter.AllPublicImagesAreAvailableField)

	options := resourceResellerImages().Schema[edgecenter.ResellerImagesOptionsField]
	require.Equal(t, schema.TypeSet, options.Type)
	require.True(t, options.Required)
	requireSameSchemaMap(t, ResellerImage, nestedResource(t, options).Schema)

	dsOptions := dataSourceResellerImages().Schema[edgecenter.ResellerImagesOptionsField]
	require.True(t, dsOptions.Computed)
	require.NotContains(t, nestedResource(t, dsOptions).Schema, edgecenter.AllPublicImagesAreAvailableField)

	resellerID := resourceResellerImages().Schema[edgecenter.ResellerIDField]
	require.Equal(t, schema.TypeInt, resellerID.Type)
	require.True(t, resellerID.Required)
	require.NotContains(t, resourceResellerImages().Schema, edgecenter.EntityIDField)
	require.NotContains(t, resourceResellerImages().Schema, edgecenter.EntityTypeField)

	dsResellerID := dataSourceResellerImages().Schema[edgecenter.ResellerIDField]
	require.Equal(t, schema.TypeInt, dsResellerID.Type)
	require.True(t, dsResellerID.Required)
	require.NotContains(t, dataSourceResellerImages().Schema, edgecenter.EntityIDField)
	require.NotContains(t, dataSourceResellerImages().Schema, edgecenter.EntityTypeField)
}

func TestResellerNetworksDataSourceSchema(t *testing.T) {
	t.Parallel()

	ds := dataSourceResellerNetworksList()

	require.NotNil(t, ds.ReadContext)
	require.Nil(t, ds.CreateContext)
	require.Nil(t, ds.UpdateContext)
	require.Nil(t, ds.DeleteContext)

	for _, key := range []string{
		edgecenter.NetworkTypeField,
		edgecenter.OrderByField,
		edgecenter.SharedField,
		edgecenter.MetadataKVField,
		edgecenter.MetadataKField,
	} {
		field := ds.Schema[key]
		require.NotNilf(t, field, "attribute %q is missing", key)
		require.Truef(t, field.Optional, "attribute %q must stay optional", key)
		require.Falsef(t, field.Computed, "attribute %q must stay a filter", key)
		require.Falsef(t, field.Required, "attribute %q must stay optional", key)
	}

	require.NotNil(t, ds.Schema[edgecenter.NetworkTypeField].ValidateFunc)
	require.NotNil(t, ds.Schema[edgecenter.OrderByField].ValidateFunc)
	require.Nil(t, ds.Schema[edgecenter.SharedField].ValidateFunc)

	networks := ds.Schema[edgecenter.NetworksField]
	require.Equal(t, schema.TypeList, networks.Type)
	require.True(t, networks.Computed)
	require.False(t, networks.Optional)

	requireComputedOnly(t, nestedResource(t, networks),
		edgecenter.CreatedAtField,
		edgecenter.DefaultField,
		edgecenter.ExternalField,
		edgecenter.SharedField,
		edgecenter.IDField,
		edgecenter.MTUField,
		edgecenter.NameField,
		edgecenter.RegionIDField,
		edgecenter.RegionNameField,
		edgecenter.TypeField,
		edgecenter.CreatorTaskIDField,
		edgecenter.TaskIDField,
		edgecenter.SegmentationIDField,
		edgecenter.UpdatedAtField,
		edgecenter.MetadataField,
		edgecenter.ClientIDField,
		edgecenter.ProjectIDField,
	)

	require.Len(t, nestedResource(t, networks).Schema, 18)
	require.Len(t, ds.Schema, 6)
}

func TestResellerNetworksSubnetsSchema(t *testing.T) {
	t.Parallel()

	networks := dataSourceResellerNetworksList().Schema[edgecenter.NetworksField]
	subnets := nestedField(t, networks, edgecenter.SubnetsField)

	require.Equal(t, schema.TypeList, subnets.Type)
	require.True(t, subnets.Computed)
	require.True(t, subnets.Optional)

	requireComputedOnly(t, nestedResource(t, subnets),
		edgecenter.IDField,
		edgecenter.NameField,
		edgecenter.AvailableIPsField,
		edgecenter.TotalIPsField,
		edgecenter.EnableDHCPField,
		edgecenter.HasRouterField,
		edgecenter.CIDRField,
		edgecenter.DNSNameserversField,
		edgecenter.HostRoutesField,
		edgecenter.GatewayIPField,
	)

	require.Len(t, nestedResource(t, subnets).Schema, 10)
	require.Equal(t, schema.TypeSet, nestedField(t, subnets, edgecenter.HostRoutesField).Type)
	require.Equal(t, schema.TypeList, nestedField(t, subnets, edgecenter.DNSNameserversField).Type)
}

func TestResellerNetworksOrderByValidation(t *testing.T) {
	t.Parallel()

	validate := dataSourceResellerNetworksList().Schema[edgecenter.OrderByField].ValidateFunc
	require.NotNil(t, validate)

	for _, orderBy := range []string{"name.asc", "name.desc", "created_at.asc"} {
		_, errs := validate(orderBy, edgecenter.OrderByField)
		require.Emptyf(t, errs, "%q must be accepted", orderBy)
	}

	for _, orderBy := range []string{"", "name", "name.up", "asc"} {
		_, errs := validate(orderBy, edgecenter.OrderByField)
		require.NotEmptyf(t, errs, "%q must be rejected", orderBy)
	}
}

func TestResellerNetworksOrderByAcceptsAnyStringContainingADottedDirection(t *testing.T) {
	t.Parallel()

	validate := dataSourceResellerNetworksList().Schema[edgecenter.OrderByField].ValidateFunc
	require.NotNil(t, validate)

	for _, orderBy := range []string{".asc", "name.ascending", "name.desc.and.more", " .desc "} {
		_, errs := validate(orderBy, edgecenter.OrderByField)
		require.Emptyf(t, errs, "%q is accepted because the pattern is not anchored", orderBy)
	}
}

func TestResellerNetworksNetworkTypeValidation(t *testing.T) {
	t.Parallel()

	validate := dataSourceResellerNetworksList().Schema[edgecenter.NetworkTypeField].ValidateFunc
	require.NotNil(t, validate)

	for _, networkType := range []string{string(edgecloudV2.VLAN), string(edgecloudV2.VXLAN)} {
		_, errs := validate(networkType, edgecenter.NetworkTypeField)
		require.Emptyf(t, errs, "%q must be accepted", networkType)
	}

	for _, networkType := range []string{"", "VLAN", "vlan ", "gre"} {
		_, errs := validate(networkType, edgecenter.NetworkTypeField)
		require.NotEmptyf(t, errs, "%q must be rejected", networkType)
	}
}

func resellerImagesV2Raw(imageIDs []interface{}, allPublic bool) map[string]interface{} {
	option := map[string]interface{}{
		edgecenter.RegionIDField:                    schemaRegionID,
		edgecenter.AllPublicImagesAreAvailableField: allPublic,
	}
	if imageIDs != nil {
		option[edgecenter.ImageIDsField] = imageIDs
	}

	return map[string]interface{}{
		edgecenter.EntityIDField:              schemaEntityID,
		edgecenter.EntityTypeField:            schemaEntityType,
		edgecenter.ResellerImagesOptionsField: []interface{}{option},
	}
}

func resellerImagesV2RawConfig(imageIDs cty.Value, allPublic bool) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		edgecenter.EntityIDField:   cty.NumberIntVal(schemaEntityID),
		edgecenter.EntityTypeField: cty.StringVal(schemaEntityType),
		edgecenter.ResellerImagesOptionsField: cty.SetVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				edgecenter.RegionIDField:                    cty.NumberIntVal(schemaRegionID),
				edgecenter.ImageIDsField:                    imageIDs,
				edgecenter.AllPublicImagesAreAvailableField: cty.BoolVal(allPublic),
				edgecenter.CreatedAtField:                   cty.NullVal(cty.String),
				edgecenter.UpdatedAtField:                   cty.NullVal(cty.String),
			}),
		}),
	})
}

func resellerImagesV2Diff(t *testing.T, raw map[string]interface{}, rawConfig cty.Value) error {
	t.Helper()

	res := resourceResellerImagesV2()

	data := schema.TestResourceDataRaw(t, res.Schema, raw)
	data.SetId(schemaEntityIDStr)

	state := data.State()
	require.NotNil(t, state)
	state.RawConfig = rawConfig

	_, err := res.Diff(context.Background(), state, terraform.NewResourceConfigRaw(raw), nil)

	return err
}

func TestResellerImagesV2OptionsRejectImageIDsTogetherWithAllPublicImages(t *testing.T) {
	t.Parallel()

	err := resellerImagesV2Diff(
		t,
		resellerImagesV2Raw([]interface{}{"11111111-1111-1111-1111-111111111111"}, true),
		resellerImagesV2RawConfig(cty.SetVal([]cty.Value{cty.StringVal("11111111-1111-1111-1111-111111111111")}), true),
	)

	require.EqualError(t, err, "image_ids must not be set when all_public_images_are_available is true")
}

func TestResellerImagesV2OptionsAcceptOneOfImageIDsAndAllPublicImages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		imageIDs  []interface{}
		rawIDs    cty.Value
		allPublic bool
	}{
		{
			name:      "only image_ids",
			imageIDs:  []interface{}{"11111111-1111-1111-1111-111111111111"},
			rawIDs:    cty.SetVal([]cty.Value{cty.StringVal("11111111-1111-1111-1111-111111111111")}),
			allPublic: false,
		},
		{
			name:      "only all_public_images_are_available",
			rawIDs:    cty.NullVal(cty.Set(cty.String)),
			allPublic: true,
		},
		{
			name:   "neither",
			rawIDs: cty.NullVal(cty.Set(cty.String)),
		},
		{
			name:      "an empty image_ids set on its own",
			imageIDs:  []interface{}{},
			rawIDs:    cty.SetValEmpty(cty.String),
			allPublic: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, resellerImagesV2Diff(
				t,
				resellerImagesV2Raw(tc.imageIDs, tc.allPublic),
				resellerImagesV2RawConfig(tc.rawIDs, tc.allPublic),
			))
		})
	}
}

func TestResellerImagesV2OptionsRejectAnEmptyImageIDsSetTogetherWithAllPublicImages(t *testing.T) {
	t.Parallel()

	err := resellerImagesV2Diff(
		t,
		resellerImagesV2Raw([]interface{}{}, true),
		resellerImagesV2RawConfig(cty.SetValEmpty(cty.String), true),
	)

	require.EqualError(t, err, "image_ids must not be set when all_public_images_are_available is true")
}

func TestResellerNetworkReadFillsProjectIDWithTheRegionID(t *testing.T) {
	t.Parallel()

	network := prepareResellerNetwork(edgecloudV2.ResellerNetwork{
		RegionID:  schemaRegionID,
		ProjectID: 777,
		ClientID:  555,
	})

	require.Equal(t, schemaRegionID, network[edgecenter.RegionIDField])
	require.Equal(t, 555, network[edgecenter.ClientIDField])
	require.Equal(t, schemaRegionID, network[edgecenter.ProjectIDField])
	require.NotEqual(t, 777, network[edgecenter.ProjectIDField])
}

func TestResellerImagesV2RollbackReplacesTheIDWithTheEntityType(t *testing.T) {
	t.Parallel()

	res := resourceResellerImagesV2()

	raw := schema.TestResourceDataRaw(t, res.Schema, resellerImagesV2Raw(
		[]interface{}{"11111111-1111-1111-1111-111111111111"},
		false,
	))
	raw.SetId(schemaEntityIDStr)

	state := raw.State()
	require.NotNil(t, state)

	data := res.Data(state)
	require.Equal(t, schemaEntityIDStr, data.Id())

	rollbackResellerImagesV2Data(context.Background(), data)

	require.Equal(t, schemaEntityType, data.Id())
	require.NotEqual(t, schemaEntityIDStr, data.Id())
}
