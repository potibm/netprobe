package domain

import (
	"log/slog"
	"net/http"

	netprobe_net "github.com/potibm/netprobe/src/internal/net"
)

type ClientList map[netprobe_net.IPFamily]*http.Client


type Check interface {
	Name() string
	Execute(clients ClientList, log *slog.Logger) CheckResult
}

type CheckResult struct {
	CheckName    string
	Success      bool
	ErrorMessage string
}

type Target struct {
	ID       string
	Hostname string
	Checks   []Check
}
