//go:build integration

package cdn_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestIntegrationRuleImporter(t *testing.T) {
	t.Parallel()

	resource := cdnResource(t, "edgecenter_cdn_rule")
	require.NotNil(t, resource.Importer, "edgecenter_cdn_rule must be importable")

	cases := []struct {
		name           string
		id             string
		wantErr        string
		wantID         string
		wantResourceID int
	}{
		{
			name:           "resource_id:rule_id is split into resource_id and id",
			id:             "1001:55",
			wantID:         "55",
			wantResourceID: 1001,
		},
		{
			name:    "missing separator is rejected",
			id:      "1001",
			wantErr: "unexpected format of ID",
		},
		{
			name:    "too many parts are rejected",
			id:      "1001:55:7",
			wantErr: "unexpected format of ID",
		},
		{
			name:    "non numeric resource_id is rejected",
			id:      "abc:55",
			wantErr: "invalid resource_id",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := resource.Data(&terraform.InstanceState{ID: tt.id})

			results, err := resource.Importer.StateContext(context.Background(), data, nil)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)

				return
			}

			require.NoError(t, err)
			require.Len(t, results, 1)
			require.Equal(t, tt.wantID, results[0].Id())
			require.Equal(t, tt.wantResourceID, results[0].Get("resource_id").(int))
		})
	}
}

func TestIntegrationPassthroughImporters(t *testing.T) {
	t.Parallel()

	names := []string{
		"edgecenter_cdn_resource",
		"edgecenter_cdn_origingroup",
		"edgecenter_cdn_lecert",
		"edgecenter_cdn_shielding",
		"edgecenter_cdn_sslcert",
	}

	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resource := cdnResource(t, name)
			require.NotNil(t, resource.Importer, "%s must be importable", name)

			data := resource.Data(&terraform.InstanceState{ID: "123"})

			results, err := resource.Importer.StateContext(context.Background(), data, nil)
			require.NoError(t, err)
			require.Len(t, results, 1)
			require.Equal(t, "123", results[0].Id())
		})
	}
}
