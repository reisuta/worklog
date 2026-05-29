package report

import (
	"github.com/reisuta/worklog/internal/store"
)

// Summary is the aggregated view of a time range used by the CLI, status bar
// and Markdown reports.
type Summary struct {
	Range      Range
	TotalSec   int            // active (non-idle) seconds
	IdleSec    int            // idle seconds
	FocusSec   int            // seconds inside focus blocks
	ByProject  []store.Bucket // descending
	ByApp      []store.Bucket // descending
	ByCategory []store.Bucket // descending
	Focus      []Block        // descending by duration
}

// Build computes a Summary for r from the store using the given poll interval
// and focus-block threshold.
func Build(s *store.Store, r Range, intervalSec, focusMinMinutes int) (Summary, error) {
	from, to := r.Unix()
	out := Summary{Range: r}

	var err error
	if out.TotalSec, err = s.ActiveSeconds(from, to, intervalSec); err != nil {
		return out, err
	}
	if out.IdleSec, err = s.IdleSeconds(from, to, intervalSec); err != nil {
		return out, err
	}
	if out.ByProject, err = s.ByProject(from, to, intervalSec); err != nil {
		return out, err
	}
	if out.ByApp, err = s.ByApp(from, to, intervalSec); err != nil {
		return out, err
	}
	if out.ByCategory, err = s.ByCategory(from, to, intervalSec); err != nil {
		return out, err
	}

	events, err := s.Events(from, to)
	if err != nil {
		return out, err
	}
	out.Focus = FocusBlocks(events, intervalSec, focusMinMinutes)
	for _, b := range out.Focus {
		out.FocusSec += b.Seconds()
	}
	return out, nil
}

// TopProjects returns at most n projects, largest first.
func (s Summary) TopProjects(n int) []store.Bucket {
	if n > len(s.ByProject) {
		n = len(s.ByProject)
	}
	return s.ByProject[:n]
}
