package domain

import (
	"log/slog"
	"net/http"
	"time"

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
	ErrorCode    ErrorCode
	ErrorMessage string
	Duration     time.Duration
}

type Target struct {
	ID       string
	Hostname string
	Checks   []Check
}
