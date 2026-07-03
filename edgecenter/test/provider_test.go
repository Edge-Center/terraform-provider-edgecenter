package edgecenter_test

import (
	"testing"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

func TestProvider(t *testing.T) {
	t.Parallel()
	if err := provider.Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
