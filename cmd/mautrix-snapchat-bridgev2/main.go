package main

import (
	"os"

	"maunium.net/go/mautrix/bridgev2/matrix/mxmain"

	snapconnector "github.com/colej/mautrix-snapchat/pkg/connector"
)

var (
	Tag       = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	normalizeConfigFlagAlias()
	conn := snapconnector.NewConnector()
	m := mxmain.BridgeMain{
		Name:        "mautrix-snapchat",
		Description: "Bridgev2 Snapchat Web bridge",
		URL:         "https://github.com/colej/mautrix-snapchat",
		Version:     "0.1.0",
		Connector:   conn,
	}
	m.InitVersion(Tag, Commit, BuildTime)
	m.Run()
}

func normalizeConfigFlagAlias() {
	for i, arg := range os.Args {
		if arg == "-c" {
			os.Args[i] = "--config"
		}
	}
}
