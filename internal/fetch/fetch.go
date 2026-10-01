// Package fetch downloads remote files to disk.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Placeholders are {{ obtain.NAME }} (user variable) or {{ obtain.std.NAME }}
// (builtin).
var placeholder = regexp.MustCompile(`\{\{\s*obtain\.(std\.)?([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// Vars holds the values available to placeholders.
type Vars struct {
	User map[string]string // {{ obtain.NAME }}
	Std  map[string]string // {{ obtain.std.NAME }}, see Builtins
	Raw  bool              // save the download as is, without rendering
}

// Render replaces placeholders in data. It fails, naming every offender, if
// any placeholder has no value.
func Render(data []byte, vars Vars) ([]byte, error) {
	missing, badStd := map[string]bool{}, map[string]bool{}
	out := placeholder.ReplaceAllFunc(data, func(m []byte) []byte {
		sub := placeholder.FindSubmatch(m)
		name := string(sub[2])
		src, bad := vars.User, missing
		if len(sub[1]) > 0 {
			src, bad = vars.Std, badStd
		}
		v, ok := src[name]
		if !ok {
			bad[name] = true
		}
		return []byte(v)
	})
	var msgs []string
	if len(missing) > 0 {
		msgs = append(msgs, fmt.Sprintf("no value for variable(s) %s (set with --var NAME=value or under vars: in the config)",
			joinKeys(missing)))
	}
	if len(badStd) > 0 {
		msgs = append(msgs, fmt.Sprintf("unknown or unavailable builtin(s) %s (available: %s)",
			joinKeys(badStd), joinKeys(toSet(vars.Std))))
	}
	if len(msgs) > 0 {
		return nil, errors.New(strings.Join(msgs, "; "))
	}
	return out, nil
}

func toSet(m map[string]string) map[string]bool {
	s := make(map[string]bool, len(m))
	for k := range m {
		s[k] = true
	}
	return s
}

func joinKeys(set map[string]bool) string {
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// File downloads url to dest. It refuses to overwrite an existing file unless
// force is set. Placeholders are rendered with vars (see Render).
func File(ctx context.Context, url, dest string, force bool, vars Vars) error {
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", dest)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	rendered := body
	if !vars.Raw {
		if rendered, err = Render(body, vars); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	// Write to a temp file first so a failed download never clobbers dest.
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".obtain-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(rendered); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dest)
}
