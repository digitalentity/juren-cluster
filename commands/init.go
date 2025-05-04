package commands

import (
	"context"
	"juren/config"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func RunInit(ctx context.Context, cfg *config.Config) {
	log.Info("RunInit()")
}
