package netutil

import (
	"syscall"

	"github.com/digitalocean/droplet-agent/internal/log"
	"golang.org/x/net/bpf"
)

// offsets in TCP header
const (
	offSrcPort    = 0
	offDestPort   = 2
	offSeqNum     = 4
	offAckNum     = 8
	offTCPFlags   = 12
	offWindowSize = 14
	offCheckSum   = 16
	offUrgent     = 18
	offOption     = 20
)

const (
	lenIPHeader = 20
)

const maxPacketBuf = 512

// NewTCPPacketSniffer returns a new TCP packet sniffer
func NewTCPPacketSniffer() TCPPacketSniffer {
	return &tcpPacketSniffer{
		tcpPacketSnifferHelper: newTCPPacketSnifferHelper(),
	}
}

type tcpPacketSnifferHelper interface {
	ToBpfFilters(identifier *TCPPacketIdentifier) ([]bpf.Instruction, error)
	SocketWithBPFFilter(filter []bpf.Instruction) (int, error)
	UnmarshalTCPPacket(in []byte) (*TCPPacket, error)
}

// tcpPacketSniffer implementation for linux
type tcpPacketSniffer struct {
	tcpPacketSnifferHelper

	fd int
}

func (s *tcpPacketSniffer) Capture(identifier *TCPPacketIdentifier) (<-chan *TCPPacket, error) {
	filter, err := s.ToBpfFilters(identifier)
	if err != nil {
		return nil, err
	}

	fd, err := s.SocketWithBPFFilter(filter)
	if err != nil {
		return nil, err
	}
	s.fd = fd
	packetChan := make(chan *TCPPacket)
	go s.snifferLoop(packetChan)
	return packetChan, nil
}

func (s *tcpPacketSniffer) Stop() {
	if s.fd != 0 {
		_ = syscall.Close(s.fd)
	}
}

func (s *tcpPacketSniffer) snifferLoop(packetChan chan<- *TCPPacket) {
	buffer := make([]byte, maxPacketBuf)
	minMsgLen := lenIPHeader + offOption
	for {
		n, err := syscall.Read(s.fd, buffer)
		if err != nil {
			log.Error("failed to read from socket. %v", err)
			continue
		}
		if n < minMsgLen {
			if n == 0 {
				log.Info("Sniffer quit")
				return
			}
			// less than 40 bytes (len(IP packet header) + len(minimum TCP header))
			log.Error("invalid message: insufficient read [%d]", n)
			continue
		}
		packet, err := s.UnmarshalTCPPacket(buffer[lenIPHeader:])
		if err != nil {
			log.Error("failed to unmarshal TCP packet: %v", err)
			continue
		}
		packetChan <- packet
	}
}
