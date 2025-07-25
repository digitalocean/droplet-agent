// Code generated by MockGen. DO NOT EDIT.
// Source: internal/sysutil/usermanager.go
//
// Generated by this command:
//
//	mockgen -source=internal/sysutil/usermanager.go -package=sysutil -destination=internal/sysutil/usermanager_mocks.go
//

// Package sysutil is a generated GoMock package.
package sysutil

import (
	user "os/user"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockuserOperator is a mock of userOperator interface.
type MockuserOperator struct {
	ctrl     *gomock.Controller
	recorder *MockuserOperatorMockRecorder
	isgomock struct{}
}

// MockuserOperatorMockRecorder is the mock recorder for MockuserOperator.
type MockuserOperatorMockRecorder struct {
	mock *MockuserOperator
}

// NewMockuserOperator creates a new mock instance.
func NewMockuserOperator(ctrl *gomock.Controller) *MockuserOperator {
	mock := &MockuserOperator{ctrl: ctrl}
	mock.recorder = &MockuserOperatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockuserOperator) EXPECT() *MockuserOperatorMockRecorder {
	return m.recorder
}

// Current mocks base method.
func (m *MockuserOperator) Current() (*user.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Current")
	ret0, _ := ret[0].(*user.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Current indicates an expected call of Current.
func (mr *MockuserOperatorMockRecorder) Current() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Current", reflect.TypeOf((*MockuserOperator)(nil).Current))
}
