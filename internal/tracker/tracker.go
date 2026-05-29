package tracker

import (
	"context"
	"log"
	"time"

	"github.com/reisuta/worklog/internal/config"
	"github.com/reisuta/worklog/internal/rules"
	"github.com/reisuta/worklog/internal/store"
)

// nowFn is overridable in tests.
var nowFn = time.Now

// Tracker runs the monitoring loop, turning samples into stored events.
type Tracker struct {
	store  *store.Store
	rules  rules.Set
	cfg    config.Config
	logger *log.Logger

	// probe is the sampling function; overridable in tests.
	probe func() (Sample, error)
}

// New builds a Tracker.
func New(s *store.Store, rs rules.Set, cfg config.Config, logger *log.Logger) *Tracker {
	return &Tracker{store: s, rules: rs, cfg: cfg, logger: logger, probe: Probe}
}

// Run polls until ctx is cancelled, recording one event per tick.
func (t *Tracker) Run(ctx context.Context) error {
	interval := time.Duration(t.cfg.Tracker.PollIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	t.logf("worklog tracker started (interval=%s)", interval)
	t.tick() // record immediately so short sessions aren't lost
	for {
		select {
		case <-ctx.Done():
			t.logf("worklog tracker stopped")
			return ctx.Err()
		case <-ticker.C:
			t.tick()
		}
	}
}

// tick takes one sample and stores the resulting event. Errors are logged but
// never stop the loop: a transient osascript hiccup shouldn't kill the daemon.
func (t *Tracker) tick() {
	s, err := t.probe()
	if err != nil {
		t.logf("probe error: %v", err)
		return
	}
	e := t.event(s, nowFn().Unix())
	if err := t.store.Insert(e); err != nil {
		t.logf("store error: %v", err)
	}
}

// event converts a sample into an event, applying idle, privacy and rule logic.
func (t *Tracker) event(s Sample, ts int64) store.Event {
	idle := s.IdleSec >= t.cfg.Tracker.IdleThresholdSeconds

	title := s.Title
	if !t.cfg.Privacy.RecordWindowTitle || t.cfg.IsExcluded(s.App) {
		title = ""
	}

	project, category := t.rules.Match(s.App, title)
	return store.Event{
		TS:       ts,
		App:      s.App,
		Title:    title,
		IsIdle:   idle,
		Project:  project,
		Category: category,
	}
}

func (t *Tracker) logf(format string, args ...any) {
	if t.logger != nil {
		t.logger.Printf(format, args...)
	}
}
