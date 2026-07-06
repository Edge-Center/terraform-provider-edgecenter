package edgemonmock

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Edge-Center/edgecenteredgemon-go/channel"
	"github.com/Edge-Center/edgecenteredgemon-go/checkgroup"
	"github.com/Edge-Center/edgecenteredgemon-go/checks"
	"github.com/Edge-Center/edgecenteredgemon-go/statuspage"
)

func ptrArg[T any](args mock.Arguments, idx int) *T {
	if v := args.Get(idx); v != nil {
		return v.(*T)
	}

	return nil
}

type ChannelService struct{ mock.Mock }

func (m *ChannelService) Get(ctx context.Context, receiver string, channelID int) (*channel.Response, error) {
	args := m.Called(ctx, receiver, channelID)
	return ptrArg[channel.Response](args, 0), args.Error(1)
}

func (m *ChannelService) Create(ctx context.Context, receiver string, req *channel.Request) (*channel.Response, error) {
	args := m.Called(ctx, receiver, req)
	return ptrArg[channel.Response](args, 0), args.Error(1)
}

func (m *ChannelService) Update(ctx context.Context, receiver string, channelID int, req *channel.Request) error {
	return m.Called(ctx, receiver, channelID, req).Error(0)
}

func (m *ChannelService) Delete(ctx context.Context, receiver string, channelID int) error {
	return m.Called(ctx, receiver, channelID).Error(0)
}

type StatusPageService struct{ mock.Mock }

func (m *StatusPageService) Create(ctx context.Context, req *statuspage.Request) (*statuspage.CreateResponse, error) {
	args := m.Called(ctx, req)
	return ptrArg[statuspage.CreateResponse](args, 0), args.Error(1)
}

func (m *StatusPageService) Get(ctx context.Context, pageID int) (*statuspage.Response, error) {
	args := m.Called(ctx, pageID)
	return ptrArg[statuspage.Response](args, 0), args.Error(1)
}

func (m *StatusPageService) Update(ctx context.Context, pageID int, req *statuspage.Request) error {
	return m.Called(ctx, pageID, req).Error(0)
}

func (m *StatusPageService) Delete(ctx context.Context, pageID int) error {
	return m.Called(ctx, pageID).Error(0)
}

type CheckGroupService struct{ mock.Mock }

func (m *CheckGroupService) Create(ctx context.Context, req *checkgroup.Request) (*checkgroup.Response, error) {
	args := m.Called(ctx, req)
	return ptrArg[checkgroup.Response](args, 0), args.Error(1)
}

func (m *CheckGroupService) Get(ctx context.Context, id int) (*checkgroup.Response, error) {
	args := m.Called(ctx, id)
	return ptrArg[checkgroup.Response](args, 0), args.Error(1)
}

func (m *CheckGroupService) Update(ctx context.Context, id int, req *checkgroup.Request) (*checkgroup.Response, error) {
	args := m.Called(ctx, id, req)
	return ptrArg[checkgroup.Response](args, 0), args.Error(1)
}

func (m *CheckGroupService) Delete(ctx context.Context, id int) error {
	return m.Called(ctx, id).Error(0)
}

type CheckService[Req any, Resp any] struct{ mock.Mock }

func (m *CheckService[Req, Resp]) Create(ctx context.Context, req *Req) (*checks.CreateResponse, error) {
	args := m.MethodCalled("Create", ctx, req)
	return ptrArg[checks.CreateResponse](args, 0), args.Error(1)
}

func (m *CheckService[Req, Resp]) Get(ctx context.Context, checkID int) (*Resp, error) {
	args := m.MethodCalled("Get", ctx, checkID)
	return ptrArg[Resp](args, 0), args.Error(1)
}

func (m *CheckService[Req, Resp]) Update(ctx context.Context, checkID int, req *Req) error {
	return m.MethodCalled("Update", ctx, checkID, req).Error(0)
}

func (m *CheckService[Req, Resp]) Delete(ctx context.Context, checkID int) error {
	return m.MethodCalled("Delete", ctx, checkID).Error(0)
}
