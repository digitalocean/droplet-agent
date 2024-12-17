// SPDX-License-Identifier: Apache-2.0

package reservedipv6

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/jsimonetti/rtnetlink/v2"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

const (
	loIface   string = "lo"
	eth0Iface string = "eth0"
	prefixLen uint8  = 128
)

type Manager interface {
	Assign(ip string) error
	Unassign() error
}

type mgr struct {
	nlConn  *rtnetlink.Conn
	loIdx   uint32
	eth0Idx uint32
}

func NewManager(netlink *rtnetlink.Conn) (Manager, error) {
	lo, err := net.InterfaceByName(loIface)
	if err != nil {
		return nil, fmt.Errorf("failed to determine index for interface '%s': %w", loIface, err)
	}

	eth0, err := net.InterfaceByName(eth0Iface)
	if err != nil {
		return nil, fmt.Errorf("failed to determine index for interface '%s': %w", eth0Iface, err)
	}

	return &mgr{
		nlConn:  netlink,
		loIdx:   uint32(lo.Index),
		eth0Idx: uint32(eth0.Index),
	}, nil
}

// Assign creates a new global-scoped IPv6 address on
func (m *mgr) Assign(ip string) error {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return fmt.Errorf("invalid IP: %w", err)
	}
	if addr.Is4() || addr.Is4In6() {
		return fmt.Errorf("IP must be an IPv6 address")
	}

	// Equivalent to `ip -6 addr replace "${ip}/128" dev lo scope global`
	req := reservedIPv6Addr(m.loIdx, addr)
	flags := netlink.Request | netlink.Replace | netlink.Acknowledge
	if _, err := m.nlConn.Execute(req, unix.RTM_NEWADDR, flags); err != nil {
		return fmt.Errorf("failed to assign address: %w", err)
	}

	if err := m.nlConn.Route.Replace(defaultIPv6Route(m.eth0Idx)); err != nil {
		return fmt.Errorf("failed to replace default IPv6 route on eth0: %w", err)
	}

	return nil
}

// Unassign removes all global-scoped IPv6 addresses on localhost
func (m *mgr) Unassign() error {
	addrs, err := m.nlConn.Address.List()
	if err != nil {
		return fmt.Errorf("failed to list addreses: %w", err)
	}

	for _, a := range addrs {
		if a.Index == m.loIdx && a.Family == unix.AF_INET6 && a.Scope == unix.RT_SCOPE_UNIVERSE {
			if err := m.nlConn.Address.Delete(&a); err != nil {
				return fmt.Errorf("failed to delete address '%s' from interface '%s': %w", a.Attributes.Address, loIface, err)
			}
		}
	}

	// remove the default route if it existed
	route := defaultIPv6Route(m.eth0Idx)
	if _, err := m.nlConn.Route.Get(route); err == nil {
		if err := m.nlConn.Route.Delete(route); err != nil {
			return fmt.Errorf("failed to remove default IPv6 route on %s: %w", eth0Iface, err)
		}
	}

	return nil
}

func reservedIPv6Addr(intfIdx uint32, addr netip.Addr) *rtnetlink.AddressMessage {
	return &rtnetlink.AddressMessage{
		Family:       unix.AF_INET6,
		PrefixLength: prefixLen,
		Scope:        unix.RT_SCOPE_UNIVERSE, // global
		Index:        intfIdx,
		Attributes: &rtnetlink.AddressAttributes{
			Address: net.IP(addr.AsSlice()),
		},
	}
}

// defaultIPv6Route returns a route equivalent to `ip -6 route replace default dev eth0`
func defaultIPv6Route(intfIdx uint32) *rtnetlink.RouteMessage {
	return &rtnetlink.RouteMessage{
		Family:    unix.AF_INET6,
		Table:     unix.RT_TABLE_MAIN,
		Protocol:  unix.RTPROT_BOOT,
		Type:      unix.RTN_UNICAST,
		Scope:     unix.RT_SCOPE_UNIVERSE,
		DstLength: 0, // default route
		Attributes: rtnetlink.RouteAttributes{
			Dst:      nil, // default route
			OutIface: intfIdx,
			Priority: 1024,
		},
	}
}
