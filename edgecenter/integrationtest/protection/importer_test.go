//go:build integration

package protection_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	protectionsvc "github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/services/protection"
)

func importerNames() []string {
	return []string{
		protectionsvc.ProtectionResourceResource,
		protectionsvc.ProtectionCertificateResource,
		protectionsvc.ProtectionOriginResource,
		protectionsvc.ProtectionHeaderResource,
		protectionsvc.ProtectionBlacklistEntryResource,
		protectionsvc.ProtectionWhitelistEntryResource,
		protectionsvc.ProtectionAliasResource,
		protectionsvc.ProtectionAliasCertificateResource,
	}
}

func TestIntegrationImporterPassthroughProtection(t *testing.T) {
	t.Parallel()

	ids := []string{
		resourceIDStr,
		compositeID,
		"not-a-number",
		"42:7:9",
		"  42:7  ",
		"",
	}

	for _, name := range importerNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resource := protectionResource(t, name)
			require.NotNil(t, resource.Importer, "%s must be importable", name)
			require.NotNil(t, resource.Importer.StateContext, "%s must have an import state function", name)
			require.Equal(t,
				reflect.ValueOf(schema.ImportStatePassthroughContext).Pointer(),
				reflect.ValueOf(resource.Importer.StateContext).Pointer(),
				"%s must import through the sdk passthrough", name)

			for _, id := range ids {
				data := resource.Data(&terraform.InstanceState{ID: id})

				results, err := resource.Importer.StateContext(context.Background(), data, nil)

				require.NoError(t, err, "%s must accept any id at import time", name)
				require.Len(t, results, 1)
				require.Same(t, data, results[0])
				require.Equal(t, id, results[0].Id(), "%s must keep the id verbatim", name)
			}
		})
	}
}

func TestIntegrationImporterKeepsSeededAttributesProtection(t *testing.T) {
	t.Parallel()

	resource := protectionResource(t, protectionsvc.ProtectionOriginResource)

	data := resource.Data(&terraform.InstanceState{
		ID: compositeID,
		Attributes: map[string]string{
			protectionsvc.ProtectionOriginSchemaResource: "stale",
			protectionsvc.ProtectionOriginSchemaIP:       "10.9.9.9",
		},
	})

	results, err := resource.Importer.StateContext(context.Background(), data, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "stale", results[0].Get(protectionsvc.ProtectionOriginSchemaResource))
	require.Equal(t, "10.9.9.9", results[0].Get(protectionsvc.ProtectionOriginSchemaIP))
}
