package storagemock

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter"
)

type MockedStorage struct {
	Config *edgecenter.Config

	mocks []*mock.Mock

	Locations *StorageLocationService
	Storages  *StorageS3Service
	Buckets   *StorageBucketService
}

func (mc *MockedStorage) TestMeta() interface{} {
	return mc.Config
}

func (mc *MockedStorage) MockCleanup(t *testing.T) {
	t.Helper()

	for _, m := range mc.mocks {
		m.AssertExpectations(t)
	}
}

func NewMockedStorage() *MockedStorage {
	mc := &MockedStorage{
		Locations: &StorageLocationService{},
		Storages:  &StorageS3Service{},
		Buckets:   &StorageBucketService{},
	}

	mc.mocks = []*mock.Mock{
		&mc.Locations.Mock,
		&mc.Storages.Mock,
		&mc.Buckets.Mock,
	}

	mc.Config = &edgecenter.Config{StorageClient: &clientShim{
		StorageLocationService: mc.Locations,
		StorageS3Service:       mc.Storages,
		StorageBucketService:   mc.Buckets,
	}}

	return mc
}

type clientShim struct {
	*StorageLocationService
	*StorageS3Service
	*StorageBucketService
}

var _ edgecenter.StorageClientService = (*clientShim)(nil)
