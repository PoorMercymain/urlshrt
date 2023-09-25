// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/PoorMercymain/urlshrt/internal/domain (interfaces: URLRepository)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	state "github.com/PoorMercymain/urlshrt/internal/state"
	gomock "github.com/golang/mock/gomock"
)

// MockURLRepository is a mock of URLRepository interface.
type MockURLRepository struct {
	ctrl     *gomock.Controller
	recorder *MockURLRepositoryMockRecorder
}

// MockURLRepositoryMockRecorder is the mock recorder for MockURLRepository.
type MockURLRepositoryMockRecorder struct {
	mock *MockURLRepository
}

// NewMockURLRepository creates a new mock instance.
func NewMockURLRepository(ctrl *gomock.Controller) *MockURLRepository {
	mock := &MockURLRepository{ctrl: ctrl}
	mock.recorder = &MockURLRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockURLRepository) EXPECT() *MockURLRepositoryMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockURLRepository) Create(arg0 context.Context, arg1 []state.URLStringJSON) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockURLRepositoryMockRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockURLRepository)(nil).Create), arg0, arg1)
}

// CreateBatch mocks base method.
func (m *MockURLRepository) CreateBatch(arg0 context.Context, arg1 []*state.URLStringJSON) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBatch", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateBatch indicates an expected call of CreateBatch.
func (mr *MockURLRepositoryMockRecorder) CreateBatch(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBatch", reflect.TypeOf((*MockURLRepository)(nil).CreateBatch), arg0, arg1)
}

// DeleteUserURLs mocks base method.
func (m *MockURLRepository) DeleteUserURLs(arg0 context.Context, arg1 []string, arg2 []int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUserURLs", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteUserURLs indicates an expected call of DeleteUserURLs.
func (mr *MockURLRepositoryMockRecorder) DeleteUserURLs(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUserURLs", reflect.TypeOf((*MockURLRepository)(nil).DeleteUserURLs), arg0, arg1, arg2)
}

// IsURLDeleted mocks base method.
func (m *MockURLRepository) IsURLDeleted(arg0 context.Context, arg1 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsURLDeleted", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsURLDeleted indicates an expected call of IsURLDeleted.
func (mr *MockURLRepositoryMockRecorder) IsURLDeleted(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsURLDeleted", reflect.TypeOf((*MockURLRepository)(nil).IsURLDeleted), arg0, arg1)
}

// PingPg mocks base method.
func (m *MockURLRepository) PingPg(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PingPg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PingPg indicates an expected call of PingPg.
func (mr *MockURLRepositoryMockRecorder) PingPg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PingPg", reflect.TypeOf((*MockURLRepository)(nil).PingPg), arg0)
}

// ReadAll mocks base method.
func (m *MockURLRepository) ReadAll(arg0 context.Context) ([]state.URLStringJSON, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadAll", arg0)
	ret0, _ := ret[0].([]state.URLStringJSON)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadAll indicates an expected call of ReadAll.
func (mr *MockURLRepositoryMockRecorder) ReadAll(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadAll", reflect.TypeOf((*MockURLRepository)(nil).ReadAll), arg0)
}

// ReadUserURLs mocks base method.
func (m *MockURLRepository) ReadUserURLs(arg0 context.Context) ([]state.URLStringJSON, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadUserURLs", arg0)
	ret0, _ := ret[0].([]state.URLStringJSON)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadUserURLs indicates an expected call of ReadUserURLs.
func (mr *MockURLRepositoryMockRecorder) ReadUserURLs(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadUserURLs", reflect.TypeOf((*MockURLRepository)(nil).ReadUserURLs), arg0)
}
