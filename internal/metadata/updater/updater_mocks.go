// Code generated by MockGen. DO NOT EDIT.
// Source: internal/metadata/updater/updater.go
//
// Generated by this command:
//
//	mockgen -source=internal/metadata/updater/updater.go -package=updater -destination=internal/metadata/updater/updater_mocks.go
//

// Package updater is a generated GoMock package.
package updater

import (
	http "net/http"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockhttpClient is a mock of httpClient interface.
type MockhttpClient struct {
	ctrl     *gomock.Controller
	recorder *MockhttpClientMockRecorder
	isgomock struct{}
}

// MockhttpClientMockRecorder is the mock recorder for MockhttpClient.
type MockhttpClientMockRecorder struct {
	mock *MockhttpClient
}

// NewMockhttpClient creates a new mock instance.
func NewMockhttpClient(ctrl *gomock.Controller) *MockhttpClient {
	mock := &MockhttpClient{ctrl: ctrl}
	mock.recorder = &MockhttpClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockhttpClient) EXPECT() *MockhttpClientMockRecorder {
	return m.recorder
}

// Do mocks base method.
func (m *MockhttpClient) Do(req *http.Request) (*http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Do", req)
	ret0, _ := ret[0].(*http.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Do indicates an expected call of Do.
func (mr *MockhttpClientMockRecorder) Do(req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*MockhttpClient)(nil).Do), req)
}
