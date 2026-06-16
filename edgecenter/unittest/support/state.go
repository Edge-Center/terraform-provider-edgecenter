package support

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func NewState(
	t *testing.T,
	resource *schema.Resource,
	raw map[string]interface{},
	id string,
) *terraform.InstanceState {
	t.Helper()

	data := schema.TestResourceDataRaw(t, resource.Schema, normalizeRawConfig(raw))
	data.SetId(id)

	return data.State()
}

func NewResourceDataFromState(
	t *testing.T,
	resource *schema.Resource,
	state *terraform.InstanceState,
) *schema.ResourceData {
	t.Helper()

	return resource.Data(state)
}

func ApplyConfig(
	t *testing.T,
	ctx context.Context,
	resource *schema.Resource,
	currentState *terraform.InstanceState,
	raw map[string]interface{},
	meta interface{},
) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()

	diff, err := resource.Diff(ctx, currentState, terraform.NewResourceConfigRaw(normalizeRawConfig(raw)), meta)
	require.NoError(t, err)

	return resource.Apply(ctx, currentState, diff, meta)
}

func EmptyState() *terraform.InstanceState {
	return nil
}

func currentStateFromCase(
	t *testing.T,
	resource *schema.Resource,
	raw map[string]interface{},
	id string,
) *terraform.InstanceState {
	t.Helper()

	if raw == nil && id == "" {
		return nil
	}

	return NewState(t, resource, raw, id)
}

func normalizeRawConfig(raw map[string]interface{}) map[string]interface{} {
	if raw != nil {
		return raw
	}

	return map[string]interface{}{}
}
