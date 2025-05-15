package commands

import (
	"context"
	"juren/config"

	log "github.com/sirupsen/logrus"
)

func RunInit(ctx context.Context, cfg *config.Config) {
	log.Info("RunInit()")

	// Save the config
	if err := cfg.Save(); err != nil {
		log.Fatalf("Failed to save config: %v", err)
	}
}
