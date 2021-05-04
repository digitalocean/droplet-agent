package netutil

import (
	"errors"
)

// TCP flags
const (
	TCPFlagFIN = 1  // 0000 0001
	TCPFlagSYN = 2  // 0000 0010
	TCPFlagRST = 4  // 0000 0100
	TCPFlagPSH = 8  // 0000 1000
	TCPFlagACK = 16 // 0001 0000
	TCPFlagURG = 32 // 0010 0000
)

// TCPPacketSniffer supports capturing tcp packets following designated pattern
type TCPPacketSniffer interface {
	Capture(identifier *TCPPacketIdentifier) (<-chan *TCPPacket, error)
	Stop()
}

// Possible return errors
var (
	ErrInvalidIdentifier = errors.New("invalid tcp packet identifier")
	ErrCreateSocket      = errors.New("failed to create socket")
	ErrApplyFilter       = errors.New("failed to apply bpf filter")
	ErrMessageTooShort   = errors.New("input message is too short")
)

// TCPPacketIdentifier provides instructions for filtering the packets
type TCPPacketIdentifier struct {
	TargetPort uint16
	SeqNum     uint32
	AckNum     uint32
	TCPFlag    uint8
}

// TCPPacket describes a tcp packet
type TCPPacket struct {
	// Header fields
	Source      uint16
	Destination uint16
	SeqNum      uint32
	AckNum      uint32
	DataOffset  uint8 // 4 bits
	Reserved    uint8 // 3 bits
	ECN         uint8 // 3 bits
	Ctrl        uint8 // 6 bits
	Window      uint16
	Checksum    uint16 // Kernel will set this if it's 0
	Urgent      uint16
	//Options and payload are omitted as they are not currently being used
}
