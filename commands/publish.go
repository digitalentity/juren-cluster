package commands

import (
	"context"
	"juren/config"

	log "github.com/sirupsen/logrus"
)

func RunPublish(ctx context.Context, cfg *config.Config) {
	log.Infof("Publishing ipfs-go-storage...")
}
