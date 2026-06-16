package testacc

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

func NewProvider() *schema.Provider {
	return provider.Provider()
}

func Factory(p *schema.Provider) map[string]func() (*schema.Provider, error) {
	return map[string]func() (*schema.Provider, error){
		"edgecenter": func() (*schema.Provider, error) { //nolint:unparam
			return p, nil
		},
	}
}

func PreCheck(t *testing.T, vars ...string) {
	t.Helper()
	for _, name := range append([]string{"EC_PERMANENT_TOKEN"}, vars...) {
		if os.Getenv(name) == "" {
			t.Fatalf("%s must be set for acceptance tests", name)
		}
	}
}

func UniqueName(prefix string) string {
	return acctest.RandomWithPrefix(prefix)
}
