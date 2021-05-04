package netutil

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"

	"golang.org/x/net/bpf"
)

func newTCPPacketSnifferHelper() tcpPacketSnifferHelper {
	return &tcpSnifferHelperImpl{
		dependentFns: &dependentFnsImpl{},
	}
}

type dependentFns interface {
	SockCreate(domain, typ, proto int) (fd int, err error)
	BPFAssemble(insts []bpf.Instruction) ([]bpf.RawInstruction, error)
	Syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err unix.Errno)
	Close(fd int) (err error)
}

type dependentFnsImpl struct {
}

func (f *dependentFnsImpl) SockCreate(domain, typ, proto int) (fd int, err error) {
	return unix.Socket(domain, typ, proto)
}
func (f *dependentFnsImpl) BPFAssemble(insts []bpf.Instruction) ([]bpf.RawInstruction, error) {
	return bpf.Assemble(insts)
}
func (f *dependentFnsImpl) Syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err unix.Errno) {
	return unix.Syscall6(trap, a1, a2, a3, a4, a5, a6)
}
func (f *dependentFnsImpl) Close(fd int) (err error) {
	return unix.Close(fd)
}

type tcpSnifferHelperImpl struct {
	dependentFns
}

// ToBpfFilters generates corresponding BPF filter for the given identifier
// NOTE: the current implementation only supports IPv4 packet with 20 bytes IP header
func (h *tcpSnifferHelperImpl) ToBpfFilters(identifier *TCPPacketIdentifier) ([]bpf.Instruction, error) {
	if identifier == nil {
		return nil, ErrInvalidIdentifier
	}
	filter := make([]bpf.Instruction, 0, 10)
	if identifier.TargetPort != 0 {
		filter = append(filter, []bpf.Instruction{
			bpf.LoadAbsolute{Off: lenIPHeader + offDestPort, Size: 2},
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: uint32(identifier.TargetPort), SkipFalse: 1},
		}...)
	}
	if identifier.SeqNum != 0 {
		filter = append(filter, []bpf.Instruction{
			bpf.LoadAbsolute{Off: lenIPHeader + offSeqNum, Size: 4},
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: identifier.SeqNum, SkipFalse: 1},
		}...)
	}
	if identifier.AckNum != 0 {
		filter = append(filter, []bpf.Instruction{
			bpf.LoadAbsolute{Off: lenIPHeader + offAckNum, Size: 4},
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: identifier.AckNum, SkipFalse: 1},
		}...)
	}
	if identifier.TCPFlag != 0 {
		filter = append(filter, []bpf.Instruction{
			bpf.LoadAbsolute{Off: lenIPHeader + offTCPFlags, Size: 2},
			bpf.JumpIf{Cond: bpf.JumpBitsSet, Val: uint32(identifier.TCPFlag), SkipFalse: 1},
		}...)
	}
	if len(filter) == 0 {
		return nil, ErrInvalidIdentifier
	}
	filter = append(filter, []bpf.Instruction{
		bpf.RetConstant{Val: maxPacketBuf}, // return maximum `maxPacketBuf` bytes (or less) from packet
		bpf.RetConstant{Val: 0x0},
	}...)
	// Calculate relative offset for the jmp instructions
	tRet := len(filter) - 2
	for i := 1; i < tRet; i += 2 {
		ji := filter[i].(bpf.JumpIf)
		ji.SkipFalse = uint8(tRet - i)
		filter[i] = ji
	}
	return filter, nil
}

func (h *tcpSnifferHelperImpl) SocketWithBPFFilter(filter []bpf.Instruction) (retFD int, retErr error) {
	// Create the socket
	// Note: we are using AF_INET here not AF_PACKET for maximum compatibility
	fd, err := h.SockCreate(unix.AF_INET, unix.SOCK_RAW, unix.IPPROTO_TCP)
	if err != nil {
		return 0, fmt.Errorf("%w:%v", ErrCreateSocket, err)
	}
	defer func() {
		if retErr != nil {
			_ = h.Close(fd)
		}
	}()
	// Applying the BPF instructions
	assembled, err := h.BPFAssemble(filter)
	if err != nil {
		return 0, fmt.Errorf("%w:%v", ErrApplyFilter, err)
	}
	program := unix.SockFprog{
		Len:    uint16(len(assembled)),
		Filter: (*unix.SockFilter)(unsafe.Pointer(&assembled[0])),
	}

	b := (*[unix.SizeofSockFprog]byte)(unsafe.Pointer(&program))[:unix.SizeofSockFprog]
	_, _, errno := h.Syscall6(unix.SYS_SETSOCKOPT,
		uintptr(fd),
		uintptr(unix.SOL_SOCKET),
		uintptr(unix.SO_ATTACH_FILTER),
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)),
		0)
	if errno != 0 {
		return 0, fmt.Errorf("%w:%s", ErrApplyFilter, errno.Error())
	}
	return fd, nil
}

func (h *tcpSnifferHelperImpl) UnmarshalTCPPacket(in []byte) (*TCPPacket, error) {
	if len(in) < 20 {
		return nil, ErrMessageTooShort
	}
	ret := &TCPPacket{}
	ret.Source = binary.BigEndian.Uint16(in[offSrcPort:])
	ret.Destination = binary.BigEndian.Uint16(in[offDestPort:])
	ret.SeqNum = binary.BigEndian.Uint32(in[offSeqNum:])
	ret.AckNum = binary.BigEndian.Uint32(in[offAckNum:])
	mix := binary.BigEndian.Uint16(in[offTCPFlags:])
	ret.DataOffset = uint8(mix >> 12)    // first 4 bits is the DataOffset
	ret.Reserved = uint8((mix >> 9) & 7) // the following 3 bits are the reserved bits
	ret.ECN = uint8((mix >> 6) & 7)      // then 3 bits of ECN related flags
	ret.Ctrl = uint8(mix & 0x3f)         // fetch the last 6 bits of TCP flags. NOTE: 0x3f = 0011 1111
	ret.Window = binary.BigEndian.Uint16(in[offWindowSize:])
	ret.Checksum = binary.BigEndian.Uint16(in[offCheckSum:])
	ret.Urgent = binary.BigEndian.Uint16(in[offUrgent:])
	return ret, nil
}
