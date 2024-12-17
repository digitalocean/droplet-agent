//go:build !linux
// +build !linux

package unix

const (
	AF_INET                                    = 0x2
	AF_INET6                                   = 0xa
	AF_UNSPEC                                  = 0x0
	NETLINK_ROUTE                              = 0x0
	SizeofIfAddrmsg                            = 0x8
	SizeofIfInfomsg                            = 0x10
	SizeofNdMsg                                = 0xc
	SizeofRtMsg                                = 0xc
	SizeofRtNexthop                            = 0x8
	RTM_NEWADDR                                = 0x14
	RTM_DELADDR                                = 0x15
	RTM_GETADDR                                = 0x16
	RTM_NEWLINK                                = 0x10
	RTM_DELLINK                                = 0x11
	RTM_GETLINK                                = 0x12
	RTM_SETLINK                                = 0x13
	RTM_NEWROUTE                               = 0x18
	RTM_DELROUTE                               = 0x19
	RTM_GETROUTE                               = 0x1a
	RTM_NEWNEIGH                               = 0x1c
	RTM_DELNEIGH                               = 0x1d
	RTM_GETNEIGH                               = 0x1e
	IFA_UNSPEC                                 = 0x0
	IFA_ADDRESS                                = 0x1
	IFA_LOCAL                                  = 0x2
	IFA_LABEL                                  = 0x3
	IFA_BROADCAST                              = 0x4
	IFA_ANYCAST                                = 0x5
	IFA_CACHEINFO                              = 0x6
	IFA_MULTICAST                              = 0x7
	IFA_FLAGS                                  = 0x8
	IFF_UP                                     = 0x1
	IFF_BROADCAST                              = 0x2
	IFF_LOOPBACK                               = 0x8
	IFF_POINTOPOINT                            = 0x10
	IFF_MULTICAST                              = 0x1000
	IFLA_UNSPEC                                = 0x0
	IFLA_ADDRESS                               = 0x1
	IFLA_BOND_UNSPEC                           = 0x0
	IFLA_BOND_MODE                             = 0x1
	IFLA_BOND_ACTIVE_SLAVE                     = 0x2
	IFLA_BOND_MIIMON                           = 0x3
	IFLA_BOND_UPDELAY                          = 0x4
	IFLA_BOND_DOWNDELAY                        = 0x5
	IFLA_BOND_USE_CARRIER                      = 0x6
	IFLA_BOND_ARP_INTERVAL                     = 0x7
	IFLA_BOND_ARP_IP_TARGET                    = 0x8
	IFLA_BOND_ARP_VALIDATE                     = 0x9
	IFLA_BOND_ARP_ALL_TARGETS                  = 0xa
	IFLA_BOND_PRIMARY                          = 0xb
	IFLA_BOND_PRIMARY_RESELECT                 = 0xc
	IFLA_BOND_FAIL_OVER_MAC                    = 0xd
	IFLA_BOND_XMIT_HASH_POLICY                 = 0xe
	IFLA_BOND_RESEND_IGMP                      = 0xf
	IFLA_BOND_NUM_PEER_NOTIF                   = 0x10
	IFLA_BOND_ALL_SLAVES_ACTIVE                = 0x11
	IFLA_BOND_MIN_LINKS                        = 0x12
	IFLA_BOND_LP_INTERVAL                      = 0x13
	IFLA_BOND_PACKETS_PER_SLAVE                = 0x14
	IFLA_BOND_AD_LACP_RATE                     = 0x15
	IFLA_BOND_AD_SELECT                        = 0x16
	IFLA_BOND_AD_INFO                          = 0x17
	IFLA_BOND_AD_ACTOR_SYS_PRIO                = 0x18
	IFLA_BOND_AD_USER_PORT_KEY                 = 0x19
	IFLA_BOND_AD_ACTOR_SYSTEM                  = 0x1a
	IFLA_BOND_TLB_DYNAMIC_LB                   = 0x1b
	IFLA_BOND_PEER_NOTIF_DELAY                 = 0x1c
	IFLA_BOND_AD_LACP_ACTIVE                   = 0x1d
	IFLA_BOND_MISSED_MAX                       = 0x1e
	IFLA_BOND_NS_IP6_TARGET                    = 0x1f
	IFLA_BOND_AD_INFO_UNSPEC                   = 0x0
	IFLA_BOND_AD_INFO_AGGREGATOR               = 0x1
	IFLA_BOND_AD_INFO_NUM_PORTS                = 0x2
	IFLA_BOND_AD_INFO_ACTOR_KEY                = 0x3
	IFLA_BOND_AD_INFO_PARTNER_KEY              = 0x4
	IFLA_BOND_AD_INFO_PARTNER_MAC              = 0x5
	IFLA_BOND_SLAVE_UNSPEC                     = 0x0
	IFLA_BOND_SLAVE_STATE                      = 0x1
	IFLA_BOND_SLAVE_MII_STATUS                 = 0x2
	IFLA_BOND_SLAVE_LINK_FAILURE_COUNT         = 0x3
	IFLA_BOND_SLAVE_PERM_HWADDR                = 0x4
	IFLA_BOND_SLAVE_QUEUE_ID                   = 0x5
	IFLA_BOND_SLAVE_AD_AGGREGATOR_ID           = 0x6
	IFLA_BOND_SLAVE_AD_ACTOR_OPER_PORT_STATE   = 0x7
	IFLA_BOND_SLAVE_AD_PARTNER_OPER_PORT_STATE = 0x8
	IFLA_BOND_SLAVE_PRIO                       = 0x9
	IFLA_BROADCAST                             = 0x2
	IFLA_IFNAME                                = 0x3
	IFLA_MTU                                   = 0x4
	IFLA_LINK                                  = 0x5
	IFLA_QDISC                                 = 0x6
	IFLA_OPERSTATE                             = 0x10
	IFLA_STATS                                 = 0x7
	IFLA_STATS64                               = 0x17
	IFLA_TXQLEN                                = 0xd
	IFLA_GROUP                                 = 0x1b
	IFLA_LINKINFO                              = 0x12
	IFLA_LINKMODE                              = 0x11
	IFLA_IFALIAS                               = 0x14
	IFLA_MASTER                                = 0xa
	IFLA_CARRIER                               = 0x21
	IFLA_CARRIER_CHANGES                       = 0x23
	IFLA_CARRIER_UP_COUNT                      = 0x2f
	IFLA_CARRIER_DOWN_COUNT                    = 0x30
	IFLA_PHYS_PORT_ID                          = 0x22
	IFLA_PHYS_SWITCH_ID                        = 0x24
	IFLA_PHYS_PORT_NAME                        = 0x26
	IFLA_INFO_KIND                             = 0x1
	IFLA_INFO_SLAVE_KIND                       = 0x4
	IFLA_INFO_DATA                             = 0x2
	IFLA_INFO_SLAVE_DATA                       = 0x5
	IFLA_NET_NS_PID                            = 0x13
	IFLA_NET_NS_FD                             = 0x1c
	IFLA_NETKIT_UNSPEC                         = 0x0
	IFLA_NETKIT_PEER_INFO                      = 0x1
	IFLA_NETKIT_PRIMARY                        = 0x2
	IFLA_NETKIT_POLICY                         = 0x3
	IFLA_NETKIT_PEER_POLICY                    = 0x4
	IFLA_NETKIT_MODE                           = 0x5
	IFLA_XDP                                   = 0x2b
	IFLA_XDP_FD                                = 0x1
	IFLA_XDP_ATTACHED                          = 0x2
	IFLA_XDP_FLAGS                             = 0x3
	IFLA_XDP_PROG_ID                           = 0x4
	IFLA_XDP_EXPECTED_FD                       = 0x8
	XDP_FLAGS_DRV_MODE                         = 0x4
	XDP_FLAGS_SKB_MODE                         = 0x2
	XDP_FLAGS_HW_MODE                          = 0x8
	XDP_FLAGS_MODES                            = 0xe
	XDP_FLAGS_MASK                             = 0x1f
	XDP_FLAGS_REPLACE                          = 0x10
	XDP_FLAGS_UPDATE_IF_NOEXIST                = 0x1
	LWTUNNEL_ENCAP_MPLS                        = 0x1
	MPLS_IPTUNNEL_DST                          = 0x1
	MPLS_IPTUNNEL_TTL                          = 0x2
	NDA_UNSPEC                                 = 0x0
	NDA_DST                                    = 0x1
	NDA_LLADDR                                 = 0x2
	NDA_CACHEINFO                              = 0x3
	NDA_IFINDEX                                = 0x8
	RTA_UNSPEC                                 = 0x0
	RTA_DST                                    = 0x1
	RTA_ENCAP                                  = 0x16
	RTA_ENCAP_TYPE                             = 0x15
	RTA_PREFSRC                                = 0x7
	RTA_GATEWAY                                = 0x5
	RTA_OIF                                    = 0x4
	RTA_PRIORITY                               = 0x6
	RTA_TABLE                                  = 0xf
	RTA_MARK                                   = 0x10
	RTA_EXPIRES                                = 0x17
	RTA_METRICS                                = 0x8
	RTA_MULTIPATH                              = 0x9
	RTA_PREF                                   = 0x14
	RTAX_ADVMSS                                = 0x8
	RTAX_FEATURES                              = 0xc
	RTAX_INITCWND                              = 0xb
	RTAX_INITRWND                              = 0xe
	RTAX_MTU                                   = 0x2
	NTF_PROXY                                  = 0x8
	RTN_UNICAST                                = 0x1
	RT_TABLE_MAIN                              = 0xfe
	RTPROT_BOOT                                = 0x3
	RTPROT_STATIC                              = 0x4
	RT_SCOPE_UNIVERSE                          = 0x0
	RT_SCOPE_HOST                              = 0xfe
	RT_SCOPE_LINK                              = 0xfd
	RTM_NEWRULE                                = 0x20
	RTM_GETRULE                                = 0x22
	RTM_DELRULE                                = 0x21
	FRA_UNSPEC                                 = 0x0
	FRA_DST                                    = 0x1
	FRA_SRC                                    = 0x2
	FRA_IIFNAME                                = 0x3
	FRA_GOTO                                   = 0x4
	FRA_UNUSED2                                = 0x5
	FRA_PRIORITY                               = 0x6
	FRA_UNUSED3                                = 0x7
	FRA_UNUSED4                                = 0x8
	FRA_UNUSED5                                = 0x9
	FRA_FWMARK                                 = 0xa
	FRA_FLOW                                   = 0xb
	FRA_TUN_ID                                 = 0xc
	FRA_SUPPRESS_IFGROUP                       = 0xd
	FRA_SUPPRESS_PREFIXLEN                     = 0xe
	FRA_TABLE                                  = 0xf
	FRA_FWMASK                                 = 0x10
	FRA_OIFNAME                                = 0x11
	FRA_PAD                                    = 0x12
	FRA_L3MDEV                                 = 0x13
	FRA_UID_RANGE                              = 0x14
	FRA_PROTOCOL                               = 0x15
	FRA_IP_PROTO                               = 0x16
	FRA_SPORT_RANGE                            = 0x17
	FRA_DPORT_RANGE                            = 0x18
	NETKIT_NEXT                                = -0x1
	NETKIT_PASS                                = 0x0
	NETKIT_DROP                                = 0x2
	NETKIT_REDIRECT                            = 0x7
	NETKIT_L2                                  = 0x0
	NETKIT_L3                                  = 0x1
	CLONE_NEWNET                               = 0x40000000
	O_RDONLY                                   = 0x0
	O_CLOEXEC                                  = 0x80000
)

func Unshare(_ int) error {
	return nil
}

func Gettid() int {
	return 0
}