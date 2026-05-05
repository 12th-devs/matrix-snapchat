package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/colej/mautrix-snapchat/internal/bridge"
	"github.com/colej/mautrix-snapchat/internal/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to bridge config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	app, err := bridge.New(cfg)
	if err != nil {
		log.Fatalf("create bridge: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err = app.Run(ctx); err != nil {
		log.Fatalf("run bridge: %v", err)
	}
}
