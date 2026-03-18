//go:build !sweeper

package edgecenter_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

var (
	testAccProvider  *schema.Provider
	testAccProviders map[string]func() (*schema.Provider, error)
)

func TestMain(m *testing.M) {
	testAccProvider = edgecenter.Provider()
	testAccProviders = map[string]func() (*schema.Provider, error){
		"edgecenter": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}
