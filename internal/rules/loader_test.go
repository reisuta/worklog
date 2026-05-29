package rules

import (
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "absent.toml"))
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if s.Default.Project != "other" {
		t.Errorf("default project = %q, want other", s.Default.Project)
	}
}

func TestLoadExampleFile(t *testing.T) {
	s, err := Load(filepath.Join("..", "..", "testdata", "rules", "example.toml"))
	if err != nil {
		t.Fatalf("Load example: %v", err)
	}
	if len(s.Rules) != 4 {
		t.Fatalf("got %d rules, want 4", len(s.Rules))
	}
	if p, c := s.Match("Cursor", "x — ~/dev/my-app/main.go"); p != "my-app" || c != "work" {
		t.Errorf("my-app rule mismatch: %q/%q", p, c)
	}
}
