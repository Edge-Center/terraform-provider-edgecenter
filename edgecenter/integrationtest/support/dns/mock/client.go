package dnsmock

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

type MockedDNS struct {
	Config *edgecenter.Config

	mocks []*mock.Mock

	Client *DNSClientService
}

func (mc *MockedDNS) TestMeta() interface{} {
	return mc.Config
}

func (mc *MockedDNS) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

func NewMockedDNS() *MockedDNS {
	mc := &MockedDNS{Client: &DNSClientService{}}

	mc.mocks = []*mock.Mock{&mc.Client.Mock}
	mc.Config = &edgecenter.Config{DNSClient: mc.Client}

	return mc
}

// NewUnconfiguredDNS models a provider without edgecenter_dns_api: Config.DNSClient
// stays nil, which is what checkDNSDependency rejects.
func NewUnconfiguredDNS() *MockedDNS {
	return &MockedDNS{Config: &edgecenter.Config{}}
}
