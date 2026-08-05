package resellermock

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

type errorTransport struct{}

func (t *errorTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected HTTP call: %s %s - mock this service with resellermock", r.Method, r.URL.Path)
}

type MockedReseller struct {
	Client *edgecloud.Client
	Config *edgecenter.Config

	mocks []*mock.Mock

	Images   *ResellerImageV2Service
	Networks *ResellerNetworksService
}

func (mc *MockedReseller) TestMeta() interface{} {
	return mc.Config
}

func (mc *MockedReseller) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

func NewMockedReseller() *MockedReseller {
	client := edgecloud.NewClient(nil)
	client.HTTPClient = &http.Client{Transport: &errorTransport{}}

	mc := &MockedReseller{
		Images:   &ResellerImageV2Service{},
		Networks: &ResellerNetworksService{},
	}

	mc.mocks = []*mock.Mock{
		&mc.Images.Mock,
		&mc.Networks.Mock,
	}

	client.ResellerImageV2 = mc.Images
	client.ResellerNetworks = mc.Networks

	mc.Client = client
	mc.Config = &edgecenter.Config{
		CloudClientFactory: func() (*edgecloud.Client, error) {
			return client, nil
		},
	}

	return mc
}
