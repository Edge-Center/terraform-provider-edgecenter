package mkaasmock

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
	return nil, fmt.Errorf("unexpected HTTP call: %s %s - mock this service with mkaasmock", r.Method, r.URL.Path)
}

type MockedMKaaS struct {
	Client *edgecloud.Client
	Config *edgecenter.Config

	mocks []*mock.Mock

	MKaaS    *MKaaSService
	Tasks    *TasksService
	Projects *ProjectsService
	Regions  *RegionsService
}

func (mc *MockedMKaaS) TestMeta() interface{} {
	return mc.Config
}

func (mc *MockedMKaaS) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

func NewMockedMKaaS() *MockedMKaaS {
	client := edgecloud.NewClient(nil)
	client.HTTPClient = &http.Client{Transport: &errorTransport{}}

	mc := &MockedMKaaS{
		MKaaS:    &MKaaSService{},
		Tasks:    &TasksService{},
		Projects: &ProjectsService{},
		Regions:  &RegionsService{},
	}

	mc.mocks = []*mock.Mock{
		&mc.MKaaS.Mock,
		&mc.Tasks.Mock,
		&mc.Projects.Mock,
		&mc.Regions.Mock,
	}

	client.MkaaS = mc.MKaaS
	client.Tasks = mc.Tasks
	client.Projects = mc.Projects
	client.Regions = mc.Regions

	mc.Client = client
	mc.Config = &edgecenter.Config{
		CloudClientFactory: func() (*edgecloud.Client, error) {
			return client, nil
		},
	}

	return mc
}

func AllowProjectResolution(mc *MockedMKaaS, projectID int) {
	mc.Projects.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Project{{ID: projectID, Name: "test-project"}}, nil, nil).Maybe()
}
