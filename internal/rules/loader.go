package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// Load reads and compiles a rule set from a TOML file. A missing file yields
// DefaultSet with no error so worklog runs before any rules are configured.
func Load(path string) (Set, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return DefaultSet(), nil
	}
	if err != nil {
		return DefaultSet(), fmt.Errorf("read rules %s: %w", path, err)
	}
	return Parse(data)
}

// Parse decodes and compiles a rule set from raw TOML bytes.
func Parse(data []byte) (Set, error) {
	s := DefaultSet()
	if err := toml.Unmarshal(data, &s); err != nil {
		return DefaultSet(), fmt.Errorf("parse rules: %w", err)
	}
	if err := s.compile(); err != nil {
		return DefaultSet(), err
	}
	return s, nil
}
