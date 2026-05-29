// Command worklog is the main CLI and daemon for worklog.
package main

import (
	"os"

	"github.com/reisuta/worklog/internal/cli"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cli.Version = version
	os.Exit(cli.Main(os.Args[1:]))
}
