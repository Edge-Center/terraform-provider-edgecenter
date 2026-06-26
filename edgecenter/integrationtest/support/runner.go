package support

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type CaseRunner[T any] func(
	t *testing.T,
	resource *schema.Resource,
	tc ResourceCase[T],
	fake T,
) (*terraform.InstanceState, diag.Diagnostics)

// MockCleanuper is an optional interface that fake values can implement.
// If implemented, RunResourceCases registers its Cleanup method via t.Cleanup
// so that mock expectations are verified even when Check fails early.
type MockCleanuper interface {
	MockCleanup(t *testing.T)
}

// MetaProvider is an optional interface that fake values can implement.
// If implemented, DispatchCase automatically uses TestMeta as the resource
// meta argument, keeping fake and meta tied to the same fixture object.
type MetaProvider interface {
	TestMeta() interface{}
}

func RunResourceCases[T any](
	t *testing.T,
	resource *schema.Resource,
	cases []ResourceCase[T],
	run CaseRunner[T],
) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Skip {
				t.Skip("case skipped")
			}

			if tc.Prepare == nil {
				t.Fatalf("case %q: ResourceCase.Prepare must not be nil", tc.Name)
			}

			var fake T
			if tc.Prepare != nil {
				fake = tc.Prepare()
			}

			if c, ok := any(fake).(MockCleanuper); ok {
				t.Cleanup(func() { c.MockCleanup(t) })
			}

			state, diags := run(t, resource, tc, fake)
			if tc.Check != nil {
				tc.Check(t, state, diags, fake)
			}
		})
	}
}

func DispatchCase[T any](
	t *testing.T,
	resource *schema.Resource,
	tc ResourceCase[T],
	fake T,
) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()

	meta := resolveMeta(fake)

	switch tc.Op {
	case OpApply:
		return RunCaseApply(t, resource, tc, meta)
	case OpRead:
		return RunCaseRead(t, resource, tc, meta)
	case OpDelete:
		return RunCaseDelete(t, resource, tc, meta)
	default:
		t.Fatalf("unsupported resource case operation: %q", tc.Op)
		return nil, nil
	}
}

func resolveMeta[T any](fake T) interface{} {
	if provider, ok := any(fake).(MetaProvider); ok {
		return provider.TestMeta()
	}

	return nil
}

func RunCaseApply[T any](
	t *testing.T,
	resource *schema.Resource,
	tc ResourceCase[T],
	meta interface{},
) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()

	return ApplyConfig(
		t,
		context.Background(),
		resource,
		currentStateFromCase(t, resource, tc.CurrentState, tc.CurrentID),
		tc.NewConfig,
		meta,
	)
}

func RunCaseRead[T any](
	t *testing.T,
	resource *schema.Resource,
	tc ResourceCase[T],
	meta interface{},
) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()

	current := currentStateFromCase(t, resource, tc.CurrentState, tc.CurrentID)
	data := NewResourceDataFromState(t, resource, current)
	diags := resource.ReadContext(context.Background(), data, meta)

	return data.State(), diags
}

func RunCaseDelete[T any](
	t *testing.T,
	resource *schema.Resource,
	tc ResourceCase[T],
	meta interface{},
) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()

	current := currentStateFromCase(t, resource, tc.CurrentState, tc.CurrentID)
	data := NewResourceDataFromState(t, resource, current)
	diags := resource.DeleteContext(context.Background(), data, meta)

	return data.State(), diags
}
