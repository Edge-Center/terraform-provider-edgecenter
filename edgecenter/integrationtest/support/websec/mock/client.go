package websecmock

import (
	"testing"

	"github.com/stretchr/testify/mock"

	websecsdk "github.com/Edge-Center/edgecentercdn-go/websec"
	"github.com/Edge-Center/edgecentercdn-go/websec/lecerts"
	"github.com/Edge-Center/edgecentercdn-go/websec/locations"
	"github.com/Edge-Center/edgecentercdn-go/websec/resources"
	"github.com/Edge-Center/edgecentercdn-go/websec/sourcegroups"
	"github.com/Edge-Center/edgecentercdn-go/websec/sslcerts"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

type MockedWebsec struct {
	Config *edgecenter.Config

	mocks []*mock.Mock

	Resources    *ResourcesService
	Locations    *LocationsService
	SourceGroups *SourceGroupsService
	SSLCerts     *SSLCertsService
	LECerts      *LECertsService
}

func (mc *MockedWebsec) TestMeta() interface{} {
	return mc.Config
}

func (mc *MockedWebsec) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

func NewMockedWebsec() *MockedWebsec {
	mc := &MockedWebsec{
		Resources:    &ResourcesService{},
		Locations:    &LocationsService{},
		SourceGroups: &SourceGroupsService{},
		SSLCerts:     &SSLCertsService{},
		LECerts:      &LECertsService{},
	}

	mc.mocks = []*mock.Mock{
		&mc.Resources.Mock,
		&mc.Locations.Mock,
		&mc.SourceGroups.Mock,
		&mc.SSLCerts.Mock,
		&mc.LECerts.Mock,
	}

	mc.Config = &edgecenter.Config{WebsecClient: &clientShim{mc: mc}}

	return mc
}

type clientShim struct{ mc *MockedWebsec }

var _ websecsdk.ClientService = (*clientShim)(nil)

func (c *clientShim) Resources() resources.Service       { return c.mc.Resources }
func (c *clientShim) Locations() locations.Service       { return c.mc.Locations }
func (c *clientShim) SourceGroups() sourcegroups.Service { return c.mc.SourceGroups }
func (c *clientShim) SSLCerts() sslcerts.Service         { return c.mc.SSLCerts }
func (c *clientShim) LECerts() lecerts.Service           { return c.mc.LECerts }
