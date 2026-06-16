package support

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type Operation string

const (
	// OpApply covers both Create and Update paths depending on whether
	// CurrentState is nil (Create) or populated (Update).
	OpApply  Operation = "apply"
	OpRead   Operation = "read"
	OpDelete Operation = "delete"
)

type CheckFunc[T any] func(
	t *testing.T,
	state *terraform.InstanceState,
	diags diag.Diagnostics,
	fake T,
)

type ResourceCase[T any] struct {
	Name         string
	Op           Operation
	CurrentState map[string]interface{}
	CurrentID    string
	NewConfig    map[string]interface{}
	Prepare      func() T
	Check        CheckFunc[T]
	Skip         bool
}
