package timer

import (
	"context"
	"math/rand"
	"reflect"
	"runtime"
	"time"

	"github.com/lthibault/jitterbug"

	log "github.com/sirupsen/logrus"
)

type Interval struct {
	Duration time.Duration
	Jitter   time.Duration
}

type tickerJitter struct {
	MaxJitter time.Duration
}

func (j tickerJitter) Jitter(d time.Duration) time.Duration {
	if j.MaxJitter >= d {
		log.Fatal("tickerJitter: MaxJitter is greater than duration")
	}

	if j.MaxJitter == 0 {
		return d
	}

	return d + (time.Duration(rand.Int63n(int64(2*j.MaxJitter))) - j.MaxJitter)
}

// Runs the provided function periodically with a given duration. Exits when a context is cancelled or when f() returns an error.
func RunWithTicker(ctx context.Context, interval *Interval, f func(ctx context.Context) error) error {
	funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()

	// Create a new jitterbug ticker
	j := jitterbug.New(interval.Duration, &tickerJitter{MaxJitter: interval.Jitter})
	defer j.Stop()

	log.Debugf("RunWithTicker: running %s with interval %v (jitter %v)", funcName, interval.Duration, interval.Jitter)

	for {
		select {
		case <-ctx.Done():
			log.Debugf("RunWithTicker: context cancelled for %s", funcName)
			return ctx.Err()
		case <-j.C: // Use the jitterbug ticker's channel
			if err := f(ctx); err != nil {
				log.Errorf("RunWithTicker: function %s returned error: %v", funcName, err)
				return err
			}
		}
	}
}
