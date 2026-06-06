package net

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockAddr implements net.Addr for testing.
type mockAddr struct {
	network string
	string  string
}

func (m mockAddr) Network() string { return m.network }
func (m mockAddr) String() string  { return m.string }

func TestExtractIP(t *testing.T) {
	// IPNet cases
	ipNet := &net.IPNet{IP: net.ParseIP("192.168.1.1")}
	assert.Equal(t, net.ParseIP("192.168.1.1"), extractIP(ipNet))

	ipNetV6 := &net.IPNet{IP: net.ParseIP("2001:db8::1")}
	assert.Equal(t, net.ParseIP("2001:db8::1"), extractIP(ipNetV6))

	// IPAddr cases
	ipAddr := &net.IPAddr{IP: net.ParseIP("10.0.0.1")}
	assert.Equal(t, net.ParseIP("10.0.0.1"), extractIP(ipAddr))

	// Unknown type
	assert.Nil(t, extractIP(mockAddr{"tcp", "foo:80"}))
}

func TestCollectIPCandidatesFiltersLoopback(t *testing.T) {
	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("127.0.0.1")},
		&net.IPNet{IP: net.ParseIP("::1")},
		&net.IPNet{IP: net.ParseIP("192.168.1.1")},
	}

	v4, v6 := collectIPCandidates(addrs, true)
	assert.Len(t, v4, 1)
	assert.Equal(t, "192.168.1.1", v4[0].String())
	assert.Len(t, v6, 0)
}

func TestCollectIPCandidatesFiltersUnspecified(t *testing.T) {
	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("0.0.0.0")},
		&net.IPNet{IP: net.ParseIP("::")},
		&net.IPNet{IP: net.ParseIP("10.0.0.1")},
	}

	v4, v6 := collectIPCandidates(addrs, true)
	assert.Len(t, v4, 1)
	assert.Equal(t, "10.0.0.1", v4[0].String())
	assert.Len(t, v6, 0)
}

func TestCollectIPCandidatesFiltersLinkLocalV6(t *testing.T) {
	linkLocal := net.ParseIP("fe80::1")
	global := net.ParseIP("2001:db8::1")

	addrs := []net.Addr{
		&net.IPNet{IP: linkLocal},
		&net.IPNet{IP: global},
	}

	// With link-local filtered
	v4, v6 := collectIPCandidates(addrs, false)
	assert.Len(t, v4, 0)
	assert.Len(t, v6, 1)
	assert.Equal(t, global, v6[0])

	// With link-local allowed
	v4, v6 = collectIPCandidates(addrs, true)
	assert.Len(t, v4, 0)
	assert.Len(t, v6, 2)
}

func TestCollectIPCandidatesSeparatesV4AndV6(t *testing.T) {
	addrs := []net.Addr{
		&net.IPNet{IP: net.ParseIP("192.168.1.1")},
		&net.IPNet{IP: net.ParseIP("10.0.0.1")},
		&net.IPNet{IP: net.ParseIP("2001:db8::1")},
	}

	v4, v6 := collectIPCandidates(addrs, true)
	assert.Len(t, v4, 2)
	assert.Len(t, v6, 1)
}

func TestSelectIPByFamilyIPv4(t *testing.T) {
	v4 := []net.IP{net.ParseIP("192.168.1.1")}
	v6 := []net.IP{net.ParseIP("2001:db8::1")}

	ip, err := selectIPByFamily("eth0", IP4, v4, v6, true)
	assert.NoError(t, err)
	assert.Equal(t, net.ParseIP("192.168.1.1"), ip)
}

func TestSelectIPByFamilyIPv4Missing(t *testing.T) {
	v6 := []net.IP{net.ParseIP("2001:db8::1")}

	ip, err := selectIPByFamily("eth0", IP4, nil, v6, true)
	assert.Error(t, err)
	assert.Nil(t, ip)
	assert.Contains(t, err.Error(), "no IPv4 address found")
}

func TestSelectIPByFamilyIPv6(t *testing.T) {
	v4 := []net.IP{net.ParseIP("192.168.1.1")}
	v6 := []net.IP{net.ParseIP("2001:db8::1")}

	ip, err := selectIPByFamily("eth0", IP6, v4, v6, true)
	assert.NoError(t, err)
	assert.Equal(t, net.ParseIP("2001:db8::1"), ip)
}

func TestSelectIPByFamilyIPv6MissingWithLinkLocalFilterHint(t *testing.T) {
	ip, err := selectIPByFamily("eth0", IP6, nil, nil, false)
	assert.Error(t, err)
	assert.Nil(t, ip)
	assert.Contains(t, err.Error(), "link-local filtered")
}

func TestSelectIPByFamilyIPv6MissingWithoutHint(t *testing.T) {
	ip, err := selectIPByFamily("eth0", IP6, nil, nil, true)
	assert.Error(t, err)
	assert.Nil(t, ip)
	assert.NotContains(t, err.Error(), "link-local")
}

func TestSelectIPByFamilyAutoPrefersV6(t *testing.T) {
	v4 := []net.IP{net.ParseIP("192.168.1.1")}
	v6 := []net.IP{net.ParseIP("2001:db8::1")}

	ip, err := selectIPByFamily("eth0", IPAuto, v4, v6, true)
	assert.NoError(t, err)
	assert.Equal(t, net.ParseIP("2001:db8::1"), ip)
}

func TestSelectIPByFamilyAutoFallsBackToV4(t *testing.T) {
	v4 := []net.IP{net.ParseIP("192.168.1.1")}

	ip, err := selectIPByFamily("eth0", IPAuto, v4, nil, true)
	assert.NoError(t, err)
	assert.Equal(t, net.ParseIP("192.168.1.1"), ip)
}

func TestSelectIPByFamilyAutoMissing(t *testing.T) {
	ip, err := selectIPByFamily("eth0", IPAuto, nil, nil, true)
	assert.Error(t, err)
	assert.Nil(t, ip)
	assert.Contains(t, err.Error(), "no suitable IP")
}

func TestSelectIPByFamilyUnknown(t *testing.T) {
	ip, err := selectIPByFamily("eth0", "42", nil, nil, true)
	assert.Error(t, err)
	assert.Nil(t, ip)
	assert.Contains(t, err.Error(), "unknown family")
}

func TestPickInterfaceIPInvalid(t *testing.T) {
	ip, err := PickInterfaceIP("nonexistent0", IP4, true)
	assert.Error(t, err)
	assert.Nil(t, ip)
	assert.Contains(t, err.Error(), "InterfaceByName")
}
