package main

import "log/slog"

var (
	version = "0.0.0"
)

func main() {
	slog.Info("Starting netprobe", "version", version)
}