// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/htchan/WebHistory/internal/repository (interfaces: Repostory)
//
// Generated by this command:
//
//	mockgen -destination=../mock/repository/repository.go -package=mockrepo . Repostory
//

// Package mockrepo is a generated GoMock package.
package mockrepo

import (
	context "context"
	sql "database/sql"
	reflect "reflect"

	model "github.com/htchan/WebHistory/internal/model"
	gomock "go.uber.org/mock/gomock"
)

// MockRepostory is a mock of Repostory interface.
type MockRepostory struct {
	ctrl     *gomock.Controller
	recorder *MockRepostoryMockRecorder
	isgomock struct{}
}

// MockRepostoryMockRecorder is the mock recorder for MockRepostory.
type MockRepostoryMockRecorder struct {
	mock *MockRepostory
}

// NewMockRepostory creates a new mock instance.
func NewMockRepostory(ctrl *gomock.Controller) *MockRepostory {
	mock := &MockRepostory{ctrl: ctrl}
	mock.recorder = &MockRepostoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRepostory) EXPECT() *MockRepostoryMockRecorder {
	return m.recorder
}

// CreateUserWebsite mocks base method.
func (m *MockRepostory) CreateUserWebsite(arg0 context.Context, arg1 *model.UserWebsite) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUserWebsite", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateUserWebsite indicates an expected call of CreateUserWebsite.
func (mr *MockRepostoryMockRecorder) CreateUserWebsite(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUserWebsite", reflect.TypeOf((*MockRepostory)(nil).CreateUserWebsite), arg0, arg1)
}

// CreateWebsite mocks base method.
func (m *MockRepostory) CreateWebsite(arg0 context.Context, arg1 *model.Website) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateWebsite", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateWebsite indicates an expected call of CreateWebsite.
func (mr *MockRepostoryMockRecorder) CreateWebsite(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateWebsite", reflect.TypeOf((*MockRepostory)(nil).CreateWebsite), arg0, arg1)
}

// DeleteUserWebsite mocks base method.
func (m *MockRepostory) DeleteUserWebsite(arg0 context.Context, arg1 *model.UserWebsite) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUserWebsite", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteUserWebsite indicates an expected call of DeleteUserWebsite.
func (mr *MockRepostoryMockRecorder) DeleteUserWebsite(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUserWebsite", reflect.TypeOf((*MockRepostory)(nil).DeleteUserWebsite), arg0, arg1)
}

// DeleteWebsite mocks base method.
func (m *MockRepostory) DeleteWebsite(arg0 context.Context, arg1 *model.Website) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteWebsite", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteWebsite indicates an expected call of DeleteWebsite.
func (mr *MockRepostoryMockRecorder) DeleteWebsite(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteWebsite", reflect.TypeOf((*MockRepostory)(nil).DeleteWebsite), arg0, arg1)
}

// FindUserWebsite mocks base method.
func (m *MockRepostory) FindUserWebsite(ctx context.Context, userUUID, websiteUUID string) (*model.UserWebsite, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindUserWebsite", ctx, userUUID, websiteUUID)
	ret0, _ := ret[0].(*model.UserWebsite)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindUserWebsite indicates an expected call of FindUserWebsite.
func (mr *MockRepostoryMockRecorder) FindUserWebsite(ctx, userUUID, websiteUUID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindUserWebsite", reflect.TypeOf((*MockRepostory)(nil).FindUserWebsite), ctx, userUUID, websiteUUID)
}

// FindUserWebsites mocks base method.
func (m *MockRepostory) FindUserWebsites(ctx context.Context, userUUID string) (model.UserWebsites, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindUserWebsites", ctx, userUUID)
	ret0, _ := ret[0].(model.UserWebsites)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindUserWebsites indicates an expected call of FindUserWebsites.
func (mr *MockRepostoryMockRecorder) FindUserWebsites(ctx, userUUID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindUserWebsites", reflect.TypeOf((*MockRepostory)(nil).FindUserWebsites), ctx, userUUID)
}

// FindUserWebsitesByGroup mocks base method.
func (m *MockRepostory) FindUserWebsitesByGroup(ctx context.Context, userUUID, group string) (model.WebsiteGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindUserWebsitesByGroup", ctx, userUUID, group)
	ret0, _ := ret[0].(model.WebsiteGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindUserWebsitesByGroup indicates an expected call of FindUserWebsitesByGroup.
func (mr *MockRepostoryMockRecorder) FindUserWebsitesByGroup(ctx, userUUID, group any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindUserWebsitesByGroup", reflect.TypeOf((*MockRepostory)(nil).FindUserWebsitesByGroup), ctx, userUUID, group)
}

// FindWebsite mocks base method.
func (m *MockRepostory) FindWebsite(ctx context.Context, uuid string) (*model.Website, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindWebsite", ctx, uuid)
	ret0, _ := ret[0].(*model.Website)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindWebsite indicates an expected call of FindWebsite.
func (mr *MockRepostoryMockRecorder) FindWebsite(ctx, uuid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindWebsite", reflect.TypeOf((*MockRepostory)(nil).FindWebsite), ctx, uuid)
}

// FindWebsites mocks base method.
func (m *MockRepostory) FindWebsites(arg0 context.Context) ([]model.Website, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindWebsites", arg0)
	ret0, _ := ret[0].([]model.Website)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindWebsites indicates an expected call of FindWebsites.
func (mr *MockRepostoryMockRecorder) FindWebsites(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindWebsites", reflect.TypeOf((*MockRepostory)(nil).FindWebsites), arg0)
}

// Stats mocks base method.
func (m *MockRepostory) Stats() sql.DBStats {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stats")
	ret0, _ := ret[0].(sql.DBStats)
	return ret0
}

// Stats indicates an expected call of Stats.
func (mr *MockRepostoryMockRecorder) Stats() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stats", reflect.TypeOf((*MockRepostory)(nil).Stats))
}

// UpdateUserWebsite mocks base method.
func (m *MockRepostory) UpdateUserWebsite(arg0 context.Context, arg1 *model.UserWebsite) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUserWebsite", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateUserWebsite indicates an expected call of UpdateUserWebsite.
func (mr *MockRepostoryMockRecorder) UpdateUserWebsite(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUserWebsite", reflect.TypeOf((*MockRepostory)(nil).UpdateUserWebsite), arg0, arg1)
}

// UpdateWebsite mocks base method.
func (m *MockRepostory) UpdateWebsite(arg0 context.Context, arg1 *model.Website) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateWebsite", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateWebsite indicates an expected call of UpdateWebsite.
func (mr *MockRepostoryMockRecorder) UpdateWebsite(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateWebsite", reflect.TypeOf((*MockRepostory)(nil).UpdateWebsite), arg0, arg1)
}
