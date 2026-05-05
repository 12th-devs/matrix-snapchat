package main

import (
	"maunium.net/go/mautrix/bridgev2/matrix/mxmain"

	"github.com/colej/mautrix-snapchat/internal/bridgev2"
)

var (
	Tag       = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	conn := bridgev2.NewConnector()
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

