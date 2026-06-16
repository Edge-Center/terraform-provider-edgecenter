package cloudmock

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

// errorTransport fails fast on any HTTP request to unmocked services.
type errorTransport struct{}

func (t *errorTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected HTTP call: %s %s — mock this service with cloudmock", r.Method, r.URL.Path)
}

// MockedCloud holds a fully mocked cloud client and its Config.
// Tests set up expectations on exported service mocks, then pass
// MockedCloud.Config as Meta to CRUD functions.
type MockedCloud struct {
	Client *edgecloud.Client
	Config *edgecenter.Config

	mocks []*mock.Mock

	Tasks    *TasksService
	Networks *NetworksService
	Projects *ProjectsService
	Regions  *RegionsService
	Volumes  *VolumesService
	KeyPairs *KeyPairsService
}

// TestMeta returns the provider meta bound to this fixture.
func (mc *MockedCloud) TestMeta() interface{} {
	return mc.Config
}

// MockCleanup verifies all mock expectations. Designed to be used via
// the MockCleanuper interface — RunResourceCases registers it automatically
// via t.Cleanup, guaranteeing verification even when Check fails early.
func (mc *MockedCloud) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

// NewMockedCloud creates a strict MockedCloud with client/project/region set
// and selected services replaced by generated testify mocks.
//
// No default stubs are set — every expected SDK call must be explicitly
// mocked in the test. Unmocked services or unexpected calls fall through
// to errorTransport and fail fast.
func NewMockedCloud(projectID, regionID int) *MockedCloud {
	client := edgecloud.NewClient(nil)
	client.Project = projectID
	client.Region = regionID

	client.HTTPClient = &http.Client{Transport: &errorTransport{}}

	mc := &MockedCloud{
		mocks:    make([]*mock.Mock, 0, 6),
		Tasks:    &TasksService{},
		Networks: &NetworksService{},
		Projects: &ProjectsService{},
		Regions:  &RegionsService{},
		Volumes:  &VolumesService{},
		KeyPairs: &KeyPairsService{},
	}

	mc.mocks = append(mc.mocks,
		&mc.Tasks.Mock,
		&mc.Networks.Mock,
		&mc.Projects.Mock,
		&mc.Regions.Mock,
		&mc.Volumes.Mock,
		&mc.KeyPairs.Mock,
	)

	client.Tasks = mc.Tasks
	client.Networks = mc.Networks
	client.Projects = mc.Projects
	client.Regions = mc.Regions
	client.Volumes = mc.Volumes
	client.KeyPairs = mc.KeyPairs

	mc.Client = client
	mc.Config = &edgecenter.Config{
		CloudClientFactory: func() (*edgecloud.Client, error) {
			return client, nil
		},
	}

	return mc
}

// AllowProjectResolution stubs Projects.List with .Maybe().
// Use this when project resolution is not the behavior under test.
func AllowProjectResolution(mc *MockedCloud, projectID int) {
	mc.Projects.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Project{
			{ID: projectID, Name: "test-project"},
		}, nil, nil).Maybe()
}

// AllowRegionResolution stubs Regions.List with .Maybe().
// Use this when region resolution is not the behavior under test.
func AllowRegionResolution(mc *MockedCloud, regionID int) {
	mc.Regions.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Region{
			{ID: regionID, DisplayName: "test-region"},
		}, nil, nil).Maybe()
}

// ExpectProjectResolutionTimes stubs Projects.List with an exact expected call
// count. Use this when the test wants to verify how many times InitCloudClient
// resolves project identity for a single resource flow.
func ExpectProjectResolutionTimes(mc *MockedCloud, projectID, times int) {
	mc.Projects.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Project{
			{ID: projectID, Name: "test-project"},
		}, nil, nil).Times(times)
}

// ExpectRegionResolutionTimes stubs Regions.List with an exact expected call
// count. Use this when the test wants to verify how many times InitCloudClient
// resolves region identity for a single resource flow.
func ExpectRegionResolutionTimes(mc *MockedCloud, regionID, times int) {
	mc.Regions.On("List", mock.Anything, mock.Anything).
		Return([]edgecloud.Region{
			{ID: regionID, DisplayName: "test-region"},
		}, nil, nil).Times(times)
}
