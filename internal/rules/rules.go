// Package rules infers a project and category for an observed (app, title)
// pair using a list of user-defined rules loaded from TOML.
package rules

import (
	"fmt"
	"regexp"
)

// Rule is a single match rule. An empty MatchApp matches any app; an empty
// MatchTitle matches any title. MatchTitle is a regular expression while
// MatchApp is an exact (case-sensitive) app-name match.
type Rule struct {
	MatchApp   string `toml:"match_app"`
	MatchTitle string `toml:"match_title"`
	Project    string `toml:"project"`
	Category   string `toml:"category"`

	titleRE *regexp.Regexp
}

// Fallback is applied when no rule matches.
type Fallback struct {
	Project  string `toml:"project"`
	Category string `toml:"category"`
}

// Set is an ordered collection of rules plus a default fallback.
type Set struct {
	Rules   []Rule   `toml:"rules"`
	Default Fallback `toml:"default"`
}

// DefaultSet returns a Set with no rules and an "other" fallback.
func DefaultSet() Set {
	return Set{Default: Fallback{Project: "other", Category: "other"}}
}

// compile validates rules and pre-compiles title regexps. It also fills in a
// sensible fallback if the user left it empty.
func (s *Set) compile() error {
	for i := range s.Rules {
		r := &s.Rules[i]
		if r.MatchTitle != "" {
			re, err := regexp.Compile(r.MatchTitle)
			if err != nil {
				return fmt.Errorf("rule %d: invalid match_title %q: %w", i, r.MatchTitle, err)
			}
			r.titleRE = re
		}
	}
	if s.Default.Project == "" {
		s.Default.Project = "other"
	}
	if s.Default.Category == "" {
		s.Default.Category = "other"
	}
	return nil
}

// Match returns the project and category for the given app and title. The
// first matching rule wins; the fallback is used when none match.
func (s Set) Match(app, title string) (project, category string) {
	for i := range s.Rules {
		r := s.Rules[i]
		if r.MatchApp != "" && r.MatchApp != app {
			continue
		}
		if r.titleRE != nil && !r.titleRE.MatchString(title) {
			continue
		}
		// A rule with neither matcher set is treated as a catch-all.
		return r.Project, r.Category
	}
	return s.Default.Project, s.Default.Category
}
