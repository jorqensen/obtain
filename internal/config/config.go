// Package config loads the obtain "directory" of fetchable files.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Entry describes one fetchable file.
type Entry struct {
	// Filename is the local name the file is saved as.
	Filename string `yaml:"filename"`
	// URL is the remote to fetch from.
	URL string `yaml:"url"`
	// Vars are per-file defaults for {{ obtain.NAME }} placeholders. They
	// override the top-level Config.Vars.
	Vars map[string]string `yaml:"vars"`
	// Raw saves the file exactly as downloaded, without rendering placeholders.
	Raw bool `yaml:"raw"`
}

func (e Entry) VarNames() []string { return sortedKeys(e.Vars) }

// Config maps a key (e.g. "editorconfig") to an Entry.
type Config struct {
	// Vars are default values for {{ obtain.NAME }} placeholders in any file.
	Vars  map[string]string `yaml:"vars"`
	Files map[string]Entry  `yaml:"files"`
	// Packs maps a pack name to the file keys it bundles.
	Packs map[string][]string `yaml:"packs"`
}

// DefaultPath returns $XDG_CONFIG_HOME/obtain/config.yaml, falling back to
// ~/.config/obtain/config.yaml.
func DefaultPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "obtain", "config.yaml"), nil
}

// ReadFile reads the config file at path.
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("config file not found: %s (run `obtain edit` to create it)", path)
	}
	return data, err
}

// Load reads and validates the config at path. Use Check to see every problem.
func Load(path string) (*Config, error) {
	data, err := ReadFile(path)
	if err != nil {
		return nil, err
	}
	r := Check(data)
	if len(r.Problems) > 0 {
		return nil, fmt.Errorf("%s: %s (%d problem(s) found, run `obtain doctor` for details)",
			path, r.Problems[0], len(r.Problems))
	}
	return r.Config, nil
}

// FileKeys returns the file keys in sorted order.
func (c *Config) FileKeys() []string { return sortedKeys(c.Files) }

// PackKeys returns the pack names in sorted order.
func (c *Config) PackKeys() []string { return sortedKeys(c.Packs) }

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Starter is the commented template `obtain edit` seeds a new config with. It
// is valid on its own: everything is commented out, so it configures nothing.
const Starter = `# obtain config. Run ` + "`obtain doctor`" + ` to check it.
#
# files:
#   editorconfig:
#     filename: .editorconfig
#     url: https://example.com/.editorconfig
#
# packs:
#   go-project: [editorconfig]   # keys must exist under files
`

// Create writes the Starter config to path, creating parent directories. It
// does nothing if path already exists, and reports whether it wrote the file.
func Create(path string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := f.WriteString(Starter); err != nil {
		f.Close()
		return false, err
	}
	return true, f.Close()
}
