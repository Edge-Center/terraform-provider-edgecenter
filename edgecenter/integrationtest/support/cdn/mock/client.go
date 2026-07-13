package cdnmock

import (
	"testing"

	"github.com/stretchr/testify/mock"

	cdn "github.com/Edge-Center/edgecentercdn-go"
	"github.com/Edge-Center/edgecentercdn-go/lecerts"
	"github.com/Edge-Center/edgecentercdn-go/origingroups"
	"github.com/Edge-Center/edgecentercdn-go/resources"
	"github.com/Edge-Center/edgecentercdn-go/rules"
	"github.com/Edge-Center/edgecentercdn-go/shielding"
	"github.com/Edge-Center/edgecentercdn-go/sslcerts"
	"github.com/Edge-Center/edgecentercdn-go/statistics"
	"github.com/Edge-Center/edgecentercdn-go/tools"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

type MockedCDN struct {
	Config *edgecenter.Config

	mocks []*mock.Mock

	Resources    *ResourceService
	Rules        *RulesService
	LECerts      *LECertService
	OriginGroups *OriginGroupService
	Shielding    *ShieldingService
	SSLCerts     *SSLCertService
	Statistics   *ResourceStatisticsService
	Tools        *ResourceToolsService
}

func (mc *MockedCDN) TestMeta() interface{} {
	return mc.Config
}

func (mc *MockedCDN) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

func NewMockedCDN() *MockedCDN {
	mc := &MockedCDN{
		Resources:    &ResourceService{},
		Rules:        &RulesService{},
		LECerts:      &LECertService{},
		OriginGroups: &OriginGroupService{},
		Shielding:    &ShieldingService{},
		SSLCerts:     &SSLCertService{},
		Statistics:   &ResourceStatisticsService{},
		Tools:        &ResourceToolsService{},
	}

	mc.mocks = []*mock.Mock{
		&mc.Resources.Mock,
		&mc.Rules.Mock,
		&mc.LECerts.Mock,
		&mc.OriginGroups.Mock,
		&mc.Shielding.Mock,
		&mc.SSLCerts.Mock,
		&mc.Statistics.Mock,
		&mc.Tools.Mock,
	}

	mc.Config = &edgecenter.Config{CDNClient: &clientShim{mc: mc}}

	return mc
}

type clientShim struct{ mc *MockedCDN }

var _ cdn.ClientService = (*clientShim)(nil)

func (c *clientShim) Resources() resources.ResourceService          { return c.mc.Resources }
func (c *clientShim) Rules() rules.RulesService                     { return c.mc.Rules }
func (c *clientShim) LECerts() lecerts.LECertService                { return c.mc.LECerts }
func (c *clientShim) OriginGroups() origingroups.OriginGroupService { return c.mc.OriginGroups }
func (c *clientShim) Shielding() shielding.ShieldingService         { return c.mc.Shielding }
func (c *clientShim) SSLCerts() sslcerts.SSLCertService             { return c.mc.SSLCerts }
func (c *clientShim) Statistics() statistics.ResourceStatisticsService {
	return c.mc.Statistics
}
func (c *clientShim) Tools() tools.ResourceToolsService { return c.mc.Tools }
