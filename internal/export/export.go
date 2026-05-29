// Package export writes events in a time range as CSV or JSON.
package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/reisuta/worklog/internal/store"
)

// CSV writes events as CSV with a header row. Timestamps are rendered both as
// Unix seconds and RFC3339 local time for convenience.
func CSV(w io.Writer, events []store.Event) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{"ts", "datetime", "app", "title", "is_idle", "project", "category"}); err != nil {
		return err
	}
	for _, e := range events {
		rec := []string{
			strconv.FormatInt(e.TS, 10),
			time.Unix(e.TS, 0).Format(time.RFC3339),
			e.App,
			e.Title,
			strconv.FormatBool(e.IsIdle),
			e.Project,
			e.Category,
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// JSON writes events as a pretty-printed JSON array.
func JSON(w io.Writer, events []store.Event) error {
	type record struct {
		TS       int64  `json:"ts"`
		Datetime string `json:"datetime"`
		App      string `json:"app"`
		Title    string `json:"title"`
		IsIdle   bool   `json:"is_idle"`
		Project  string `json:"project"`
		Category string `json:"category"`
	}
	recs := make([]record, 0, len(events))
	for _, e := range events {
		recs = append(recs, record{
			TS:       e.TS,
			Datetime: time.Unix(e.TS, 0).Format(time.RFC3339),
			App:      e.App,
			Title:    e.Title,
			IsIdle:   e.IsIdle,
			Project:  e.Project,
			Category: e.Category,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(recs); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
