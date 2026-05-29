package store

import (
	"fmt"
	"sort"
)

// Bucket is a named total in seconds, used for per-app / per-project breakdowns.
type Bucket struct {
	Name    string
	Seconds int
}

// Each event represents roughly intervalSec seconds of activity, so totals are
// derived by counting events and multiplying by the poll interval. This is an
// approximation (boundaries are imprecise) but stable and cheap.

// ActiveSeconds returns the total active (non-idle) time in [from, to).
func (s *Store) ActiveSeconds(from, to int64, intervalSec int) (int, error) {
	return s.countSeconds(from, to, intervalSec, "is_idle = 0")
}

// IdleSeconds returns the total idle time in [from, to).
func (s *Store) IdleSeconds(from, to int64, intervalSec int) (int, error) {
	return s.countSeconds(from, to, intervalSec, "is_idle = 1")
}

func (s *Store) countSeconds(from, to int64, intervalSec int, where string) (int, error) {
	var n int
	err := s.db.QueryRow(
		fmt.Sprintf(`SELECT COUNT(*) FROM events WHERE ts >= ? AND ts < ? AND %s`, where),
		from, to,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count seconds: %w", err)
	}
	return n * intervalSec, nil
}

// ByApp returns active time per app in [from, to), sorted descending.
func (s *Store) ByApp(from, to int64, intervalSec int) ([]Bucket, error) {
	return s.groupBy(from, to, intervalSec, "app")
}

// ByProject returns active time per project in [from, to), sorted descending.
func (s *Store) ByProject(from, to int64, intervalSec int) ([]Bucket, error) {
	return s.groupBy(from, to, intervalSec, "COALESCE(NULLIF(project,''),'other')")
}

// ByCategory returns active time per category in [from, to), sorted descending.
func (s *Store) ByCategory(from, to int64, intervalSec int) ([]Bucket, error) {
	return s.groupBy(from, to, intervalSec, "COALESCE(NULLIF(category,''),'other')")
}

func (s *Store) groupBy(from, to int64, intervalSec int, expr string) ([]Bucket, error) {
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT %s AS k, COUNT(*) FROM events
			WHERE ts >= ? AND ts < ? AND is_idle = 0
			GROUP BY k`, expr),
		from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("group by %s: %w", expr, err)
	}
	defer rows.Close()

	var out []Bucket
	for rows.Next() {
		var b Bucket
		var n int
		if err := rows.Scan(&b.Name, &n); err != nil {
			return nil, fmt.Errorf("scan bucket: %w", err)
		}
		b.Seconds = n * intervalSec
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Seconds != out[j].Seconds {
			return out[i].Seconds > out[j].Seconds
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
