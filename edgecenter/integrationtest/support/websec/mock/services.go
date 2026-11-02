package websecmock

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Edge-Center/edgecentercdn-go/websec/lecerts"
	"github.com/Edge-Center/edgecentercdn-go/websec/locations"
	"github.com/Edge-Center/edgecentercdn-go/websec/resources"
	"github.com/Edge-Center/edgecentercdn-go/websec/sourcegroups"
	"github.com/Edge-Center/edgecentercdn-go/websec/sslcerts"
)

func arg[T any](args mock.Arguments, idx int) T {
	if value := args.Get(idx); value != nil {
		return value.(T)
	}

	var zero T

	return zero
}

type ResourcesService struct{ mock.Mock }

func (m *ResourcesService) List(ctx context.Context) ([]resources.Resource, error) {
	args := m.Called(ctx)
	return arg[[]resources.Resource](args, 0), arg[error](args, 1)
}

func (m *ResourcesService) Get(ctx context.Context, id int64) (*resources.Resource, error) {
	args := m.Called(ctx, id)
	return arg[*resources.Resource](args, 0), arg[error](args, 1)
}

func (m *ResourcesService) Create(ctx context.Context, req *resources.CreateRequest) (*resources.Resource, error) {
	args := m.Called(ctx, req)
	return arg[*resources.Resource](args, 0), arg[error](args, 1)
}

func (m *ResourcesService) Update(ctx context.Context, id int64, req *resources.UpdateRequest) (*resources.Resource, error) {
	args := m.Called(ctx, id, req)
	return arg[*resources.Resource](args, 0), arg[error](args, 1)
}

func (m *ResourcesService) Delete(ctx context.Context, id int64) error {
	return arg[error](m.Called(ctx, id), 0)
}

type LocationsService struct{ mock.Mock }

func (m *LocationsService) List(ctx context.Context, resourceID int64) ([]locations.Location, error) {
	args := m.Called(ctx, resourceID)
	return arg[[]locations.Location](args, 0), arg[error](args, 1)
}

func (m *LocationsService) Get(ctx context.Context, resourceID, id int64) (*locations.Location, error) {
	args := m.Called(ctx, resourceID, id)
	return arg[*locations.Location](args, 0), arg[error](args, 1)
}

func (m *LocationsService) Create(ctx context.Context, resourceID int64, req *locations.CreateRequest) (*locations.Location, error) {
	args := m.Called(ctx, resourceID, req)
	return arg[*locations.Location](args, 0), arg[error](args, 1)
}

func (m *LocationsService) Update(ctx context.Context, resourceID, id int64, req *locations.UpdateRequest) (*locations.Location, error) {
	args := m.Called(ctx, resourceID, id, req)
	return arg[*locations.Location](args, 0), arg[error](args, 1)
}

func (m *LocationsService) Delete(ctx context.Context, resourceID, id int64) error {
	return arg[error](m.Called(ctx, resourceID, id), 0)
}

type SourceGroupsService struct{ mock.Mock }

func (m *SourceGroupsService) List(ctx context.Context) ([]sourcegroups.SourceGroup, error) {
	args := m.Called(ctx)
	return arg[[]sourcegroups.SourceGroup](args, 0), arg[error](args, 1)
}

func (m *SourceGroupsService) Get(ctx context.Context, id int64) (*sourcegroups.SourceGroup, error) {
	args := m.Called(ctx, id)
	return arg[*sourcegroups.SourceGroup](args, 0), arg[error](args, 1)
}

func (m *SourceGroupsService) Create(ctx context.Context, req *sourcegroups.GroupRequest) (*sourcegroups.SourceGroup, error) {
	args := m.Called(ctx, req)
	return arg[*sourcegroups.SourceGroup](args, 0), arg[error](args, 1)
}

func (m *SourceGroupsService) Update(ctx context.Context, id int64, req *sourcegroups.GroupRequest) (*sourcegroups.SourceGroup, error) {
	args := m.Called(ctx, id, req)
	return arg[*sourcegroups.SourceGroup](args, 0), arg[error](args, 1)
}

func (m *SourceGroupsService) Delete(ctx context.Context, id int64) error {
	return arg[error](m.Called(ctx, id), 0)
}

type SSLCertsService struct{ mock.Mock }

func (m *SSLCertsService) List(ctx context.Context) ([]sslcerts.Cert, error) {
	args := m.Called(ctx)
	return arg[[]sslcerts.Cert](args, 0), arg[error](args, 1)
}

func (m *SSLCertsService) Get(ctx context.Context, id int64) (*sslcerts.Cert, error) {
	args := m.Called(ctx, id)
	return arg[*sslcerts.Cert](args, 0), arg[error](args, 1)
}

func (m *SSLCertsService) Create(ctx context.Context, req *sslcerts.CreateRequest) (*sslcerts.Cert, error) {
	args := m.Called(ctx, req)
	return arg[*sslcerts.Cert](args, 0), arg[error](args, 1)
}

func (m *SSLCertsService) Update(ctx context.Context, id int64, req *sslcerts.UpdateRequest) (*sslcerts.Cert, error) {
	args := m.Called(ctx, id, req)
	return arg[*sslcerts.Cert](args, 0), arg[error](args, 1)
}

func (m *SSLCertsService) Delete(ctx context.Context, id int64) error {
	return arg[error](m.Called(ctx, id), 0)
}

type LECertsService struct{ mock.Mock }

func (m *LECertsService) Issue(ctx context.Context, resourceID int64, req *lecerts.IssueRequest) error {
	return arg[error](m.Called(ctx, resourceID, req), 0)
}

func (m *LECertsService) Renew(ctx context.Context, resourceID int64) error {
	return arg[error](m.Called(ctx, resourceID), 0)
}

func (m *LECertsService) Revoke(ctx context.Context, resourceID int64, force bool) error {
	return arg[error](m.Called(ctx, resourceID, force), 0)
}

func (m *LECertsService) Get(ctx context.Context, resourceID int64) (*lecerts.Status, error) {
	args := m.Called(ctx, resourceID)
	return arg[*lecerts.Status](args, 0), arg[error](args, 1)
}

func (m *LECertsService) Cancel(ctx context.Context, resourceID int64) error {
	return arg[error](m.Called(ctx, resourceID), 0)
}
