package commands

import (
	"context"
	"juren/config"

	log "github.com/sirupsen/logrus"
)

func RunInit(ctx context.Context, cfg *config.Config) {
	log.Info("RunInit()")
}
