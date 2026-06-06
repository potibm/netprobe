package net

import (
	"fmt"
	"net"
)

type IPFamily string

const (
	IPAuto IPFamily = "auto"
	IP4    IPFamily = "4"
	IP6    IPFamily = "6"
)

func PickInterfaceIP(ifaceName string, family IPFamily, allowLinkLocalV6 bool) (net.IP, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("InterfaceByName(%q): %w", ifaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("Addrs(%q): %w", ifaceName, err)
	}

	v4Candidates, v6Candidates := collectIPCandidates(addrs, allowLinkLocalV6)

	return selectIPByFamily(ifaceName, family, v4Candidates, v6Candidates, allowLinkLocalV6)
}

func selectIPByFamily(ifaceName string, family IPFamily, v4, v6 []net.IP, allowLinkLocalV6 bool) (net.IP, error) {
	switch family {
	case IP4:
		if len(v4) == 0 {
			return nil, fmt.Errorf("no IPv4 address found on %s", ifaceName)
		}

		return v4[0], nil

	case IP6:
		if len(v6) == 0 {
			msg := "no IPv6 address found on %s"
			if !allowLinkLocalV6 {
				msg += " (link-local filtered)"
			}

			return nil, fmt.Errorf(msg, ifaceName)
		}

		return v6[0], nil

	case IPAuto:
		if len(v6) > 0 {
			return v6[0], nil
		}

		if len(v4) > 0 {
			return v4[0], nil
		}

		return nil, fmt.Errorf("no suitable IP found on %s", ifaceName)

	default:
		return nil, fmt.Errorf("unknown family %q (use auto, 4, or 6)", family)
	}
}

func extractIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}

func collectIPCandidates(addrs []net.Addr, allowLinkLocalV6 bool) (v4, v6 []net.IP) {
	for _, a := range addrs {
		ip := extractIP(a)

		if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
			continue
		}

		if ip4 := ip.To4(); ip4 != nil {
			v4 = append(v4, ip4)

			continue
		}

		ip16 := ip.To16()
		if ip16 == nil {
			continue
		}

		if !allowLinkLocalV6 && ip16.IsLinkLocalUnicast() {
			continue
		}

		v6 = append(v6, ip16)
	}

	return v4, v6
}
