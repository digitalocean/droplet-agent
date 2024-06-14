// SPDX-License-Identifier: Apache-2.0

//go:build amd64
// +build amd64

package netutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"syscall"
	"testing"
	"unsafe"

	"github.com/digitalocean/droplet-agent/internal/netutil/internal/mocks"
	"go.uber.org/mock/gomock"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
)

func Test_tcpSnifferHelperImpl_ToBpfFilters(t *testing.T) {
	tests := []struct {
		name       string
		identifier *TCPPacketIdentifier
		want       []bpf.Instruction
		wantErr    error
	}{
		{
			name:       "should return ErrInvalidIdentifier if identifier is nil",
			identifier: nil,
			want:       nil,
			wantErr:    ErrInvalidIdentifier,
		},
		{
			name:       "should return ErrInvalidIdentifier if no valid identifier",
			identifier: &TCPPacketIdentifier{},
			want:       nil,
			wantErr:    ErrInvalidIdentifier,
		},
		{
			name: "should support target port",
			identifier: &TCPPacketIdentifier{
				TargetPort: 1030,
			},
			want: []bpf.Instruction{
				bpf.LoadAbsolute{Off: 22, Size: 2},
				bpf.JumpIf{Val: 1030, SkipFalse: 1},
				bpf.RetConstant{Val: maxPacketBuf},
				bpf.RetConstant{Val: 0x0},
			},
			wantErr: nil,
		},
		{
			name: "should support sequence number",
			identifier: &TCPPacketIdentifier{
				SeqNum: 10300114,
			},
			want: []bpf.Instruction{
				bpf.LoadAbsolute{Off: 24, Size: 4},
				bpf.JumpIf{Val: 10300114, SkipFalse: 1},
				bpf.RetConstant{Val: maxPacketBuf},
				bpf.RetConstant{Val: 0x0},
			},
			wantErr: nil,
		},
		{
			name: "should support acknowledgment number",
			identifier: &TCPPacketIdentifier{
				AckNum: 10300114,
			},
			want: []bpf.Instruction{
				bpf.LoadAbsolute{Off: 28, Size: 4},
				bpf.JumpIf{Val: 10300114, SkipFalse: 1},
				bpf.RetConstant{Val: maxPacketBuf},
				bpf.RetConstant{Val: 0x0},
			},
			wantErr: nil,
		},
		{
			name: "should support tcp flags",
			identifier: &TCPPacketIdentifier{
				TCPFlag: TCPFlagSYN | TCPFlagACK,
			},
			want: []bpf.Instruction{
				bpf.LoadAbsolute{Off: 32, Size: 2},
				bpf.JumpIf{Cond: bpf.JumpBitsSet, Val: TCPFlagSYN | TCPFlagACK, SkipFalse: 1},
				bpf.RetConstant{Val: maxPacketBuf},
				bpf.RetConstant{Val: 0x0},
			},
			wantErr: nil,
		},
		{
			name: "should check the identifiers in order",
			identifier: &TCPPacketIdentifier{
				TargetPort: 22,
				SeqNum:     68796879,
				AckNum:     848489,
				TCPFlag:    TCPFlagSYN,
			},
			want: []bpf.Instruction{
				bpf.LoadAbsolute{Off: 22, Size: 2},
				bpf.JumpIf{Val: 22, SkipFalse: 7},
				bpf.LoadAbsolute{Off: 24, Size: 4},
				bpf.JumpIf{Val: 68796879, SkipFalse: 5},
				bpf.LoadAbsolute{Off: 28, Size: 4},
				bpf.JumpIf{Val: 848489, SkipFalse: 3},
				bpf.LoadAbsolute{Off: 32, Size: 2},
				bpf.JumpIf{Cond: bpf.JumpBitsSet, Val: TCPFlagSYN, SkipFalse: 1},
				bpf.RetConstant{Val: maxPacketBuf},
				bpf.RetConstant{Val: 0x0},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &tcpSnifferHelperImpl{}
			got, err := h.ToBpfFilters(tt.identifier)
			if (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("ToBpfFilters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToBpfFilters() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tcpSnifferHelperImpl_SocketWithBPFFilter(t *testing.T) {
	bpfFilter := []bpf.Instruction{
		bpf.Jump{},
	}
	assembledFilter := []bpf.RawInstruction{{}}
	sampleFD := 255

	tests := []struct {
		name    string
		prepare func(fn *mocks.MockdependentFns)
		want    int
		wantErr error
	}{
		{
			name: "should return ErrCreateSocket if failed to create socket",
			prepare: func(fn *mocks.MockdependentFns) {
				fn.EXPECT().SockCreate(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP).Return(-1, errors.New("err-socket-create"))
			},
			want:    0,
			wantErr: ErrCreateSocket,
		},
		{
			name: "should return ErrApplyFilter if failed to assemble bpf filters",
			prepare: func(fn *mocks.MockdependentFns) {
				fn.EXPECT().SockCreate(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP).Return(sampleFD, nil)
				fn.EXPECT().BPFAssemble(bpfFilter).Return(nil, errors.New("err-assemble-filter"))
				fn.EXPECT().Close(sampleFD).Return(nil)
			},
			want:    0,
			wantErr: ErrApplyFilter,
		},
		{
			name: "should return ErrApplyFilter if failed to set socket option",
			prepare: func(fn *mocks.MockdependentFns) {
				fn.EXPECT().SockCreate(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP).Return(sampleFD, nil)
				fn.EXPECT().BPFAssemble(bpfFilter).Return(assembledFilter, nil)
				fn.EXPECT().Close(sampleFD).Return(nil)
				program := unix.SockFprog{
					Len:    uint16(len(assembledFilter)),
					Filter: (*unix.SockFilter)(unsafe.Pointer(&assembledFilter[0])),
				}

				b := (*[unix.SizeofSockFprog]byte)(unsafe.Pointer(&program))[:unix.SizeofSockFprog]
				fn.EXPECT().Syscall6(uintptr(syscall.SYS_SETSOCKOPT),
					uintptr(sampleFD),
					uintptr(syscall.SOL_SOCKET),
					uintptr(syscall.SO_ATTACH_FILTER),
					gomock.Any(),
					uintptr(len(b)),
					uintptr(0)).Return(uintptr(0), uintptr(0), syscall.EACCES)
			},
			want:    0,
			wantErr: ErrApplyFilter,
		},
		{
			name: "should return fd with options applied",
			prepare: func(fn *mocks.MockdependentFns) {
				fn.EXPECT().SockCreate(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP).Return(sampleFD, nil)
				fn.EXPECT().BPFAssemble(bpfFilter).Return(assembledFilter, nil)
				program := unix.SockFprog{
					Len:    uint16(len(assembledFilter)),
					Filter: (*unix.SockFilter)(unsafe.Pointer(&assembledFilter[0])),
				}

				b := (*[unix.SizeofSockFprog]byte)(unsafe.Pointer(&program))[:unix.SizeofSockFprog]
				fn.EXPECT().Syscall6(uintptr(syscall.SYS_SETSOCKOPT),
					uintptr(sampleFD),
					uintptr(syscall.SOL_SOCKET),
					uintptr(syscall.SO_ATTACH_FILTER),
					gomock.Any(),
					uintptr(len(b)),
					uintptr(0)).Return(uintptr(0), uintptr(0), syscall.Errno(0))
			},
			want:    sampleFD,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			dependentFnsMock := mocks.NewMockdependentFns(mockCtl)

			if tt.prepare != nil {
				tt.prepare(dependentFnsMock)
			}

			h := &tcpSnifferHelperImpl{
				dependentFns: dependentFnsMock,
			}
			got, err := h.SocketWithBPFFilter(bpfFilter)
			if (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("SocketWithBPFFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SocketWithBPFFilter() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tcpSnifferHelperImpl_UnmarshalTCPPacket(t *testing.T) {
	examplePacket := &TCPPacket{
		Source:      1234,
		Destination: 5678,
		SeqNum:      109876,
		AckNum:      54321,
		DataOffset:  5,
		Reserved:    0,
		ECN:         7,                                                                           //111
		Ctrl:        TCPFlagFIN | TCPFlagSYN | TCPFlagRST | TCPFlagPSH | TCPFlagACK | TCPFlagURG, // 0011 1111
		Window:      14600,
		Checksum:    6666,
		Urgent:      1,
	}
	tests := []struct {
		name    string
		in      []byte
		want    *TCPPacket
		wantErr error
	}{
		{
			"nil should error",
			nil,
			nil,
			ErrMessageTooShort,
		},
		{
			"input must be more than 20 bytes",
			[]byte{0, 1, 2},
			nil,
			ErrMessageTooShort,
		},
		{
			"properly set fields",
			marshalTCPHeader(examplePacket),
			examplePacket,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &tcpSnifferHelperImpl{}
			got, err := h.UnmarshalTCPPacket(tt.in)
			if (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("UnmarshalTCPPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalTCPPacket() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func marshalTCPHeader(packet *TCPPacket) []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.BigEndian, packet.Source)
	_ = binary.Write(buf, binary.BigEndian, packet.Destination)
	_ = binary.Write(buf, binary.BigEndian, packet.SeqNum)
	_ = binary.Write(buf, binary.BigEndian, packet.AckNum)

	mix := uint16(packet.DataOffset)<<12 | // top 4 bits
		uint16(packet.Reserved)<<9 | // 3 bits
		uint16(packet.ECN)<<6 | // 3 bits
		uint16(packet.Ctrl) // bottom 6 bits
	_ = binary.Write(buf, binary.BigEndian, mix)

	_ = binary.Write(buf, binary.BigEndian, packet.Window)
	_ = binary.Write(buf, binary.BigEndian, packet.Checksum)
	_ = binary.Write(buf, binary.BigEndian, packet.Urgent)

	out := buf.Bytes()

	// Pad to min tcp header size, which is 20 bytes (5 32-bit words)
	pad := 20 - len(out)
	for i := 0; i < pad; i++ {
		out = append(out, 0)
	}

	return out
}
