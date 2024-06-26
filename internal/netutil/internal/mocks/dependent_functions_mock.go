// Code generated by MockGen. DO NOT EDIT.
// Source: internal/netutil/tcp_sniffer_helper_linux.go
//
// Generated by this command:
//
//	mockgen -source=internal/netutil/tcp_sniffer_helper_linux.go -package=mocks -destination=internal/netutil/internal/mocks/dependent_functions_mock.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	bpf "golang.org/x/net/bpf"
	unix "golang.org/x/sys/unix"
)

// MockdependentFns is a mock of dependentFns interface.
type MockdependentFns struct {
	ctrl     *gomock.Controller
	recorder *MockdependentFnsMockRecorder
}

// MockdependentFnsMockRecorder is the mock recorder for MockdependentFns.
type MockdependentFnsMockRecorder struct {
	mock *MockdependentFns
}

// NewMockdependentFns creates a new mock instance.
func NewMockdependentFns(ctrl *gomock.Controller) *MockdependentFns {
	mock := &MockdependentFns{ctrl: ctrl}
	mock.recorder = &MockdependentFnsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockdependentFns) EXPECT() *MockdependentFnsMockRecorder {
	return m.recorder
}

// BPFAssemble mocks base method.
func (m *MockdependentFns) BPFAssemble(insts []bpf.Instruction) ([]bpf.RawInstruction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BPFAssemble", insts)
	ret0, _ := ret[0].([]bpf.RawInstruction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BPFAssemble indicates an expected call of BPFAssemble.
func (mr *MockdependentFnsMockRecorder) BPFAssemble(insts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BPFAssemble", reflect.TypeOf((*MockdependentFns)(nil).BPFAssemble), insts)
}

// Close mocks base method.
func (m *MockdependentFns) Close(fd int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close", fd)
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockdependentFnsMockRecorder) Close(fd any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockdependentFns)(nil).Close), fd)
}

// SockCreate mocks base method.
func (m *MockdependentFns) SockCreate(domain, typ, proto int) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SockCreate", domain, typ, proto)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SockCreate indicates an expected call of SockCreate.
func (mr *MockdependentFnsMockRecorder) SockCreate(domain, typ, proto any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SockCreate", reflect.TypeOf((*MockdependentFns)(nil).SockCreate), domain, typ, proto)
}

// Syscall6 mocks base method.
func (m *MockdependentFns) Syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (uintptr, uintptr, unix.Errno) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Syscall6", trap, a1, a2, a3, a4, a5, a6)
	ret0, _ := ret[0].(uintptr)
	ret1, _ := ret[1].(uintptr)
	ret2, _ := ret[2].(unix.Errno)
	return ret0, ret1, ret2
}

// Syscall6 indicates an expected call of Syscall6.
func (mr *MockdependentFnsMockRecorder) Syscall6(trap, a1, a2, a3, a4, a5, a6 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Syscall6", reflect.TypeOf((*MockdependentFns)(nil).Syscall6), trap, a1, a2, a3, a4, a5, a6)
}
