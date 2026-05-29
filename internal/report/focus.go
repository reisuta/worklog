package report

import (
	"time"

	"github.com/reisuta/worklog/internal/store"
)

// Block is a contiguous span of focused work on one project.
type Block struct {
	Start   time.Time
	End     time.Time
	App     string
	Project string
}

// Seconds returns the block's duration in seconds.
func (b Block) Seconds() int { return int(b.End.Sub(b.Start).Seconds()) }

// FocusBlocks groups consecutive non-idle events on the same project into
// focus blocks, keeping only blocks at least minMinutes long. A single missed
// sample (gap up to 2*interval) does not break a block; a longer gap or a
// project change does.
func FocusBlocks(events []store.Event, intervalSec, minMinutes int) []Block {
	if intervalSec <= 0 {
		intervalSec = 5
	}
	maxGap := int64(2 * intervalSec)
	minDur := int64(minMinutes) * 60

	var blocks []Block
	var cur *Block
	var lastTS int64
	var curApp string // most recent app within the block

	flush := func() {
		if cur == nil {
			return
		}
		// Extend the end by one interval so a block of N samples spans the
		// time those samples represent rather than the gap between them.
		cur.End = time.Unix(lastTS+int64(intervalSec), 0).In(cur.Start.Location())
		cur.App = curApp
		if int64(cur.Seconds()) >= minDur {
			blocks = append(blocks, *cur)
		}
		cur = nil
	}

	for _, e := range events {
		if e.IsIdle || e.Project == "" {
			flush()
			continue
		}
		if cur != nil && e.Project == cur.Project && e.TS-lastTS <= maxGap {
			lastTS = e.TS
			curApp = e.App
			continue
		}
		flush()
		loc := time.Local
		cur = &Block{Start: time.Unix(e.TS, 0).In(loc), Project: e.Project}
		curApp = e.App
		lastTS = e.TS
	}
	flush()
	return blocks
}
