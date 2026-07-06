package edgemonmock

import (
	"testing"

	"github.com/stretchr/testify/mock"

	rmon "github.com/Edge-Center/edgecenteredgemon-go"
	"github.com/Edge-Center/edgecenteredgemon-go/channel"
	"github.com/Edge-Center/edgecenteredgemon-go/checkgroup"
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkdns"
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkhttp"
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkping"
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checkrabbitmq"
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checksmtp"
	"github.com/Edge-Center/edgecenteredgemon-go/checks/checktcp"
	"github.com/Edge-Center/edgecenteredgemon-go/statuspage"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

type MockedRMON struct {
	Config *edgecenter.Config

	mocks []*mock.Mock

	Channel       *ChannelService
	StatusPage    *StatusPageService
	CheckGroup    *CheckGroupService
	CheckDNS      *CheckService[checkdns.Request, checkdns.Response]
	CheckHTTP     *CheckService[checkhttp.Request, checkhttp.Response]
	CheckPing     *CheckService[checkping.Request, checkping.Response]
	CheckRabbitMQ *CheckService[checkrabbitmq.Request, checkrabbitmq.Response]
	CheckSMTP     *CheckService[checksmtp.Request, checksmtp.Response]
	CheckTCP      *CheckService[checktcp.Request, checktcp.Response]
}

func (mc *MockedRMON) TestMeta() interface{} {
	return mc.Config
}

func (mc *MockedRMON) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

func NewMockedRMON() *MockedRMON {
	mc := &MockedRMON{
		Channel:       &ChannelService{},
		StatusPage:    &StatusPageService{},
		CheckGroup:    &CheckGroupService{},
		CheckDNS:      &CheckService[checkdns.Request, checkdns.Response]{},
		CheckHTTP:     &CheckService[checkhttp.Request, checkhttp.Response]{},
		CheckPing:     &CheckService[checkping.Request, checkping.Response]{},
		CheckRabbitMQ: &CheckService[checkrabbitmq.Request, checkrabbitmq.Response]{},
		CheckSMTP:     &CheckService[checksmtp.Request, checksmtp.Response]{},
		CheckTCP:      &CheckService[checktcp.Request, checktcp.Response]{},
	}

	mc.mocks = []*mock.Mock{
		&mc.Channel.Mock,
		&mc.StatusPage.Mock,
		&mc.CheckGroup.Mock,
		&mc.CheckDNS.Mock,
		&mc.CheckHTTP.Mock,
		&mc.CheckPing.Mock,
		&mc.CheckRabbitMQ.Mock,
		&mc.CheckSMTP.Mock,
		&mc.CheckTCP.Mock,
	}

	mc.Config = &edgecenter.Config{RmonClient: &clientShim{mc: mc}}

	return mc
}

type clientShim struct{ mc *MockedRMON }

var _ rmon.ClientService = (*clientShim)(nil)

func (c *clientShim) Channel() channel.Service             { return c.mc.Channel }
func (c *clientShim) StatusPage() statuspage.Service       { return c.mc.StatusPage }
func (c *clientShim) CheckGroup() checkgroup.Service       { return c.mc.CheckGroup }
func (c *clientShim) CheckDNS() checkdns.Service           { return c.mc.CheckDNS }
func (c *clientShim) CheckHTTP() checkhttp.Service         { return c.mc.CheckHTTP }
func (c *clientShim) CheckPing() checkping.Service         { return c.mc.CheckPing }
func (c *clientShim) CheckRabbitMQ() checkrabbitmq.Service { return c.mc.CheckRabbitMQ }
func (c *clientShim) CheckSMTP() checksmtp.Service         { return c.mc.CheckSMTP }
func (c *clientShim) CheckTCP() checktcp.Service           { return c.mc.CheckTCP }
