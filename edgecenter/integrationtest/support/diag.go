package support

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func RequireNoErrorDiags(t *testing.T, diags diag.Diagnostics) {
	t.Helper()

	for _, d := range diags {
		require.NotEqualf(t, diag.Error, d.Severity, "unexpected error diag: %s", d.Summary)
	}
}

func RequireHasErrorDiags(t *testing.T, diags diag.Diagnostics) {
	t.Helper()

	for _, d := range diags {
		if d.Severity == diag.Error {
			return
		}
	}

	require.FailNow(t, "expected at least one error diagnostic")
}

func RequireOnlyErrorDiags(t *testing.T, diags diag.Diagnostics) {
	t.Helper()

	if len(diags) == 0 {
		require.FailNow(t, "expected error diagnostics, got none")
	}

	for _, d := range diags {
		if d.Severity != diag.Error {
			require.FailNowf(t, "unexpected non-error diagnostic", "severity=%v summary=%s", d.Severity, d.Summary)
		}
	}
}

func RequireErrorDiagContains(t *testing.T, diags diag.Diagnostics, want string) {
	t.Helper()

	for _, d := range diags {
		if d.Severity != diag.Error {
			continue
		}

		if strings.Contains(d.Summary, want) || strings.Contains(d.Detail, want) {
			return
		}
	}

	require.FailNowf(t, "expected error diagnostic to contain substring", "want substring: %q", want)
}

func RequireNoDiags(t *testing.T, diags diag.Diagnostics) {
	t.Helper()

	require.Empty(t, diags, "expected no diagnostics")
}

func RequireStateID(t *testing.T, state *terraform.InstanceState, want string) {
	t.Helper()

	require.NotNil(t, state, "expected non-nil state")
	require.Equal(t, want, state.ID)
}

func RequireStateAttrs(t *testing.T, state *terraform.InstanceState, want map[string]string) {
	t.Helper()

	require.NotNil(t, state, "expected non-nil state")
	for key, val := range want {
		require.Equal(t, val, state.Attributes[key], "attribute %q mismatch", key)
	}
}
