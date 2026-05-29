package rules

import "testing"

func mustParse(t *testing.T, data string) Set {
	t.Helper()
	s, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return s
}

func TestMatch(t *testing.T) {
	const cfg = `
[[rules]]
match_app = "Cursor"
match_title = ".*/my-blog.*"
project = "blog"
category = "work"

[[rules]]
match_app = "Google Chrome"
match_title = "(?i).*(twitter|x\\.com).*"
project = "sns"
category = "sns"

[[rules]]
match_app = "Slack"
project = "communication"
category = "work"

[default]
project = "other"
category = "other"
`
	s := mustParse(t, cfg)

	tests := []struct {
		name              string
		app, title        string
		wantProj, wantCat string
	}{
		{"app+title match", "Cursor", "main.go — ~/dev/my-blog", "blog", "work"},
		{"app matches but title doesn't", "Cursor", "main.go — other-project", "other", "other"},
		{"chrome twitter", "Google Chrome", "Home / X", "other", "other"},
		{"chrome x.com case-insensitive", "Google Chrome", "see X.COM now", "sns", "sns"},
		{"slack any title", "Slack", "anything", "communication", "work"},
		{"no match falls back", "Mail", "Inbox", "other", "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, c := s.Match(tt.app, tt.title)
			if p != tt.wantProj || c != tt.wantCat {
				t.Errorf("Match(%q,%q) = (%q,%q), want (%q,%q)", tt.app, tt.title, p, c, tt.wantProj, tt.wantCat)
			}
		})
	}
}

func TestMatchFirstWins(t *testing.T) {
	const cfg = `
[[rules]]
match_app = "Cursor"
project = "first"
category = "work"

[[rules]]
match_app = "Cursor"
project = "second"
category = "work"
`
	s := mustParse(t, cfg)
	if p, _ := s.Match("Cursor", ""); p != "first" {
		t.Errorf("expected first matching rule to win, got %q", p)
	}
}

func TestParseInvalidRegex(t *testing.T) {
	_, err := Parse([]byte(`
[[rules]]
match_title = "([unclosed"
project = "x"
`))
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
}

func TestDefaultFilledIn(t *testing.T) {
	s := mustParse(t, `
[[rules]]
match_app = "Cursor"
project = "x"
`)
	if s.Default.Project != "other" || s.Default.Category != "other" {
		t.Errorf("expected default other/other, got %q/%q", s.Default.Project, s.Default.Category)
	}
}
