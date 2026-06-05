package domain

import (
	"log/slog"
	"net/http"
)

type Check interface {
	Name() string
	Execute(client *http.Client, log *slog.Logger) CheckResult
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
