package cborrpc

import (
	"time"

	"github.com/sirupsen/logrus"
)

const (
	DialTimeout = 1 * time.Second
)

var log = logrus.New()
