package dbaasmock

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
	return nil, fmt.Errorf("unexpected HTTP call: %s %s - mock this service with dbaasmock", r.Method, r.URL.Path)
}

type MockedDBaaS struct {
	Client *edgecloud.Client
	Config *edgecenter.Config

	mocks []*mock.Mock

	DBaaS    *DBaaSService
	Tasks    *TasksService
	Projects *ProjectsService
	Regions  *RegionsService
}

func (mc *MockedDBaaS) TestMeta() interface{} {
	return mc.Config
}

func (mc *MockedDBaaS) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

func NewMockedDBaaS() *MockedDBaaS {
	client := edgecloud.NewClient(nil)
	client.HTTPClient = &http.Client{Transport: &errorTransport{}}

	mc := &MockedDBaaS{
		DBaaS:    &DBaaSService{},
		Tasks:    &TasksService{},
		Projects: &ProjectsService{},
		Regions:  &RegionsService{},
	}

	mc.mocks = []*mock.Mock{
		&mc.DBaaS.Mock,
		&mc.Tasks.Mock,
		&mc.Projects.Mock,
		&mc.Regions.Mock,
	}

	client.DBaaS = mc.DBaaS
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

func AllowProjectResolution(mc *MockedDBaaS, projectID int) {
	mc.Projects.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Project{{ID: projectID, Name: "test-project"}}, nil, nil).Maybe()
}

func AllowRegionResolution(mc *MockedDBaaS, regionID int) {
	mc.Regions.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Region{{ID: regionID, DisplayName: "test-region"}}, nil, nil).Maybe()
}

func ExpectProjectResolutionTimes(mc *MockedDBaaS, projectID, times int) {
	mc.Projects.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Project{{ID: projectID, Name: "test-project"}}, nil, nil).Times(times)
}

func ExpectRegionResolutionTimes(mc *MockedDBaaS, regionID, times int) {
	mc.Regions.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Region{{ID: regionID, DisplayName: "test-region"}}, nil, nil).Times(times)
}
