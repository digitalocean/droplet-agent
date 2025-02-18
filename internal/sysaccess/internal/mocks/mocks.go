// Code generated by MockGen. DO NOT EDIT.
// Source: internal/sysaccess/common.go
//
// Generated by this command:
//
//	mockgen -source=internal/sysaccess/common.go -package=mocks -destination=internal/sysaccess/internal/mocks/mocks.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	os "os"
	reflect "reflect"
	time "time"

	sysutil "github.com/digitalocean/droplet-agent/internal/sysutil"
	gomock "go.uber.org/mock/gomock"
)

// MocksysManager is a mock of sysManager interface.
type MocksysManager struct {
	ctrl     *gomock.Controller
	recorder *MocksysManagerMockRecorder
}

// MocksysManagerMockRecorder is the mock recorder for MocksysManager.
type MocksysManagerMockRecorder struct {
	mock *MocksysManager
}

// NewMocksysManager creates a new mock instance.
func NewMocksysManager(ctrl *gomock.Controller) *MocksysManager {
	mock := &MocksysManager{ctrl: ctrl}
	mock.recorder = &MocksysManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MocksysManager) EXPECT() *MocksysManagerMockRecorder {
	return m.recorder
}

// CopyFileAttribute mocks base method.
func (m *MocksysManager) CopyFileAttribute(from, to string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CopyFileAttribute", from, to)
	ret0, _ := ret[0].(error)
	return ret0
}

// CopyFileAttribute indicates an expected call of CopyFileAttribute.
func (mr *MocksysManagerMockRecorder) CopyFileAttribute(from, to any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CopyFileAttribute", reflect.TypeOf((*MocksysManager)(nil).CopyFileAttribute), from, to)
}

// CreateTempFile mocks base method.
func (m *MocksysManager) CreateTempFile(dir, pattern string, user *sysutil.User) (sysutil.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTempFile", dir, pattern, user)
	ret0, _ := ret[0].(sysutil.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateTempFile indicates an expected call of CreateTempFile.
func (mr *MocksysManagerMockRecorder) CreateTempFile(dir, pattern, user any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTempFile", reflect.TypeOf((*MocksysManager)(nil).CreateTempFile), dir, pattern, user)
}

// FileExists mocks base method.
func (m *MocksysManager) FileExists(name string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FileExists", name)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FileExists indicates an expected call of FileExists.
func (mr *MocksysManagerMockRecorder) FileExists(name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FileExists", reflect.TypeOf((*MocksysManager)(nil).FileExists), name)
}

// GetUserByName mocks base method.
func (m *MocksysManager) GetUserByName(username string) (*sysutil.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserByName", username)
	ret0, _ := ret[0].(*sysutil.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserByName indicates an expected call of GetUserByName.
func (mr *MocksysManagerMockRecorder) GetUserByName(username any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByName", reflect.TypeOf((*MocksysManager)(nil).GetUserByName), username)
}

// MkDirIfNonExist mocks base method.
func (m *MocksysManager) MkDirIfNonExist(dir string, user *sysutil.User, perm os.FileMode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MkDirIfNonExist", dir, user, perm)
	ret0, _ := ret[0].(error)
	return ret0
}

// MkDirIfNonExist indicates an expected call of MkDirIfNonExist.
func (mr *MocksysManagerMockRecorder) MkDirIfNonExist(dir, user, perm any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MkDirIfNonExist", reflect.TypeOf((*MocksysManager)(nil).MkDirIfNonExist), dir, user, perm)
}

// ReadFile mocks base method.
func (m *MocksysManager) ReadFile(filename string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFile", filename)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFile indicates an expected call of ReadFile.
func (mr *MocksysManagerMockRecorder) ReadFile(filename any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFile", reflect.TypeOf((*MocksysManager)(nil).ReadFile), filename)
}

// ReadFileOfUser mocks base method.
func (m *MocksysManager) ReadFileOfUser(filename string, user *sysutil.User) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFileOfUser", filename, user)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFileOfUser indicates an expected call of ReadFileOfUser.
func (mr *MocksysManagerMockRecorder) ReadFileOfUser(filename, user any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFileOfUser", reflect.TypeOf((*MocksysManager)(nil).ReadFileOfUser), filename, user)
}

// RemoveFile mocks base method.
func (m *MocksysManager) RemoveFile(name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveFile", name)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveFile indicates an expected call of RemoveFile.
func (mr *MocksysManagerMockRecorder) RemoveFile(name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveFile", reflect.TypeOf((*MocksysManager)(nil).RemoveFile), name)
}

// RenameFile mocks base method.
func (m *MocksysManager) RenameFile(oldpath, newpath string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RenameFile", oldpath, newpath)
	ret0, _ := ret[0].(error)
	return ret0
}

// RenameFile indicates an expected call of RenameFile.
func (mr *MocksysManagerMockRecorder) RenameFile(oldpath, newpath any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RenameFile", reflect.TypeOf((*MocksysManager)(nil).RenameFile), oldpath, newpath)
}

// Sleep mocks base method.
func (m *MocksysManager) Sleep(d time.Duration) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Sleep", d)
}

// Sleep indicates an expected call of Sleep.
func (mr *MocksysManagerMockRecorder) Sleep(d any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sleep", reflect.TypeOf((*MocksysManager)(nil).Sleep), d)
}
