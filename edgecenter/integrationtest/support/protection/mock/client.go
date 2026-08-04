package protectionmock

import (
	"testing"

	"github.com/stretchr/testify/mock"

	protectionSDK "github.com/Edge-Center/edgecenterprotection-go"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

type MockedProtection struct {
	Config *edgecenter.Config

	mocks []*mock.Mock

	Resources  *ResourcesService
	Aliases    *AliasesService
	Origins    *OriginsService
	Headers    *HeadersService
	Blacklists *BlacklistsService
	Whitelists *WhitelistsService
	Services   *ServicesService
}

func (mc *MockedProtection) TestMeta() interface{} {
	return mc.Config
}

func (mc *MockedProtection) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

func NewMockedProtection() *MockedProtection {
	mc := &MockedProtection{
		Resources:  &ResourcesService{},
		Aliases:    &AliasesService{},
		Origins:    &OriginsService{},
		Headers:    &HeadersService{},
		Blacklists: &BlacklistsService{},
		Whitelists: &WhitelistsService{},
		Services:   &ServicesService{},
	}

	mc.mocks = []*mock.Mock{
		&mc.Resources.Mock,
		&mc.Aliases.Mock,
		&mc.Origins.Mock,
		&mc.Headers.Mock,
		&mc.Blacklists.Mock,
		&mc.Whitelists.Mock,
		&mc.Services.Mock,
	}

	mc.Config = &edgecenter.Config{ProtectionClient: &protectionSDK.Client{
		Resources:  mc.Resources,
		Aliases:    mc.Aliases,
		Origins:    mc.Origins,
		Headers:    mc.Headers,
		Blacklists: mc.Blacklists,
		Whitelists: mc.Whitelists,
		Services:   mc.Services,
	}}

	return mc
}
