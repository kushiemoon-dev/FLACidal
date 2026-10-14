package api

import (
	"context"
	"math/rand"
	"time"

	"flacidal/internal/app"
)

const (
	healthProbeInitialDelay = 10 * time.Second
	healthProbeInterval     = 30 * time.Minute
	healthProbeJitter       = 0.2              // +/-20 percent of the interval
	healthProbeSkipRecent   = 10 * time.Minute // endpoint used by a real download recently
)

type probeSchedule struct {
	initial, interval time.Duration
	jitter            float64
	rand              func() float64 // [0,1), injectable
}

// runProbeLoop waits sc.initial, then calls probe once per interval (with
// jitter) until ctx is cancelled, including while waiting.
func runProbeLoop(ctx context.Context, sc probeSchedule, probe func(context.Context)) {
	wait := sc.initial
	for {
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
		probe(ctx)
		wait = time.Duration(float64(sc.interval) * (1 + sc.jitter*(2*sc.rand()-1)))
	}
}

// StartHealthProbe starts the background endpoint probe: first pass shortly
// after startup, then every half hour. It uses the per-endpoint probe, which
// never changes routing on failure, and skips endpoints a real download used
// recently. NewServer does not start it, so tests make no network request.
func (s *Server) StartHealthProbe() {
	if s.ctx == nil {
		return
	}
	sc := probeSchedule{healthProbeInitialDelay, healthProbeInterval, healthProbeJitter, rand.Float64}
	go runProbeLoop(s.ctx, sc, func(ctx context.Context) {
		app.ProbeAllEndpoints(ctx, s.healthSources(), healthProbeSkipRecent)
	})
}
