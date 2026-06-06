package checks

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	netprobe_net "github.com/potibm/netprobe/src/internal/net"
)

func createClientForInterface(iface string, family netprobe_net.IPFamily, timeout time.Duration) (*http.Client, error) {
	ip, err := netprobe_net.PickInterfaceIP(iface, family, false)
	if err != nil {
		return nil, err
	}

	slog.Info("ᯤ Using interface", "interface", iface, "ip", ip.String(), "ipfamily", family)

	dialer := &net.Dialer{
		Timeout: timeout,
		LocalAddr: &net.TCPAddr{
			IP: ip,
		},
	}

	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}
