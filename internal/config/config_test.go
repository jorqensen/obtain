package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Run("xdg without file defaults to yaml", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(xdg, "obtain", "config.yaml"); got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})
	t.Run("yml is ignored", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)
		writeFile(t, filepath.Join(xdg, "obtain", "config.yml"), "")
		got, _ := DefaultPath()
		if want := filepath.Join(xdg, "obtain", "config.yaml"); got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})
	t.Run("falls back to ~/.config", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", home)
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(home, ".config", "obtain", "config.yaml"); got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		// wantProblems are substrings, one per expected problem, in line order.
		wantProblems []string
	}{
		{name: "empty file", yaml: ""},
		{name: "comment only", yaml: "# nothing here\n"},
		{name: "empty files key", yaml: "files:\n"},
		{name: "files without packs", yaml: "files:\n  a: {filename: a, url: u}\n"},
		{
			name: "valid with packs",
			yaml: "files:\n  a: {filename: a, url: u}\npacks:\n  p: [a]\n",
		},
		{name: "valid vars", yaml: "files:\n  a:\n    filename: a\n    url: u\n    vars:\n      NAME: Ada\n      YEAR: \"2026\"\n"},
		{name: "valid raw", yaml: "files:\n  a:\n    filename: a\n    url: u\n    raw: true\n"},
		{name: "raw false with vars", yaml: "files:\n  a:\n    filename: a\n    url: u\n    raw: false\n    vars:\n      N: x\n"},
		{
			name:         "raw not a bool",
			yaml:         "files:\n  a:\n    filename: a\n    url: u\n    raw: yes please\n",
			wantProblems: []string{`raw must be true or false`},
		},
		{
			name:         "raw with vars",
			yaml:         "files:\n  a:\n    filename: a\n    url: u\n    raw: true\n    vars:\n      N: x\n",
			wantProblems: []string{`line 6: file "a": vars has no effect with raw: true`},
		},
		{
			name:         "vars not a mapping",
			yaml:         "files:\n  a:\n    filename: a\n    url: u\n    vars: [NAME]\n",
			wantProblems: []string{`vars must be a mapping`},
		},
		{
			name:         "invalid var name",
			yaml:         "files:\n  a:\n    filename: a\n    url: u\n    vars:\n      \"my name\": x\n",
			wantProblems: []string{`invalid variable name "my name"`},
		},
		{
			name:         "var default not a string",
			yaml:         "files:\n  a:\n    filename: a\n    url: u\n    vars:\n      NAME: [x]\n",
			wantProblems: []string{`variable "NAME" needs a string value`},
		},
		{
			name:         "var without value",
			yaml:         "files:\n  a:\n    filename: a\n    url: u\n    vars:\n      NAME:\n",
			wantProblems: []string{`variable "NAME" needs a string value`},
		},
		{name: "valid top-level vars", yaml: "vars:\n  NAME: Ada\n  EMAIL: a@b.c\n"},
		{
			name:         "top-level vars not a mapping",
			yaml:         "vars: [a]\n",
			wantProblems: []string{`top level: vars must be a mapping`},
		},
		{
			name:         "top-level invalid var name",
			yaml:         "vars:\n  1x: a\n",
			wantProblems: []string{`top level: invalid variable name "1x"`},
		},
		{
			name:         "typo in vars key",
			yaml:         "var:\n  NAME: a\n",
			wantProblems: []string{`unknown key "var", did you mean "vars"?`},
		},
		{name: "empty pack is fine", yaml: "files:\n  a: {filename: a, url: u}\npacks:\n  p:\n"},
		{
			name:         "syntax error",
			yaml:         "files: [\n",
			wantProblems: []string{"syntax error"},
		},
		{
			name:         "top level not a mapping",
			yaml:         "- a\n- b\n",
			wantProblems: []string{"top level must be a mapping"},
		},
		{
			name:         "typo in top-level key suggests packs",
			yaml:         "files:\n  a: {filename: a, url: u}\npack:\n  p: [a]\n",
			wantProblems: []string{`unknown key "pack", did you mean "packs"?`},
		},
		{
			name:         "unknown top-level key without suggestion",
			yaml:         "banana: 1\n",
			wantProblems: []string{`unknown key "banana"`},
		},
		{
			name: "typo in entry field",
			yaml: "files:\n  a:\n    filename: a\n    urls: u\n",
			wantProblems: []string{
				`file "a" is missing url`, // reported at the entry's line
				`unknown field "urls" in file "a", did you mean "url"?`,
			},
		},
		{
			name:         "missing filename",
			yaml:         "files:\n  a: {url: u}\n",
			wantProblems: []string{`file "a" is missing filename`},
		},
		{
			name:         "empty url counts as missing",
			yaml:         "files:\n  a: {filename: a, url: \"\"}\n",
			wantProblems: []string{`file "a" is missing url`},
		},
		{
			name:         "entry not a mapping",
			yaml:         "files:\n  a: nope\n",
			wantProblems: []string{`file "a" must be a mapping`},
		},
		{
			name:         "files not a mapping",
			yaml:         "files: [a]\n",
			wantProblems: []string{`"files" must be a mapping`},
		},
		{
			name:         "packs not a mapping",
			yaml:         "packs: [a]\n",
			wantProblems: []string{`"packs" must be a mapping`},
		},
		{
			name:         "pack not a list",
			yaml:         "files:\n  a: {filename: a, url: u}\npacks:\n  p: a\n",
			wantProblems: []string{`pack "p" must be a list`},
		},
		{
			name:         "pack entry not a scalar",
			yaml:         "files:\n  a: {filename: a, url: u}\npacks:\n  p: [[a]]\n",
			wantProblems: []string{`pack "p": entries must be file keys`},
		},
		{
			name:         "pack references unknown file with suggestion",
			yaml:         "files:\n  license: {filename: L, url: u}\npacks:\n  p: [licence]\n",
			wantProblems: []string{`pack "p" references unknown file "licence", did you mean "license"?`},
		},
		{
			name:         "pack references unknown file without suggestion",
			yaml:         "files:\n  a: {filename: a, url: u}\npacks:\n  p: [zzzzzz]\n",
			wantProblems: []string{`pack "p" references unknown file "zzzzzz"`},
		},
		{
			name:         "packs before files still resolve",
			yaml:         "packs:\n  p: [a]\nfiles:\n  a: {filename: a, url: u}\n",
			wantProblems: nil,
		},
		{
			name: "all problems reported, not just the first",
			yaml: "files:\n  a: {filename: a}\npack: {}\npacks:\n  p: [b]\n",
			wantProblems: []string{
				`file "a" is missing url`,
				`unknown key "pack"`,
				`pack "p" references unknown file "b"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Check([]byte(tt.yaml))
			if len(r.Problems) != len(tt.wantProblems) {
				t.Fatalf("got %d problems, want %d: %v", len(r.Problems), len(tt.wantProblems), r.Problems)
			}
			for i, want := range tt.wantProblems {
				if !strings.Contains(r.Problems[i].String(), want) {
					t.Errorf("problem %d = %q, want it to contain %q", i, r.Problems[i], want)
				}
			}
			if (len(tt.wantProblems) == 0) != (r.Config != nil) {
				t.Errorf("Config nil = %v with %d problems", r.Config == nil, len(r.Problems))
			}
		})
	}
}

func TestCheckReportsLineNumbers(t *testing.T) {
	r := Check([]byte("files:\n  a: {filename: a, url: u}\npack:\n  p: [a]\n"))
	if len(r.Problems) != 1 || r.Problems[0].Line != 3 {
		t.Fatalf("want one problem on line 3, got %v", r.Problems)
	}
	if !strings.HasPrefix(r.Problems[0].String(), "line 3: ") {
		t.Errorf("String() = %q", r.Problems[0])
	}
}

func TestCheckSortsProblemsByLine(t *testing.T) {
	// The unknown key is on a later line than the missing field but is found first.
	r := Check([]byte("banana: 1\nfiles:\n  a: {filename: a}\n"))
	if len(r.Problems) != 2 {
		t.Fatalf("got %v", r.Problems)
	}
	if r.Problems[0].Line > r.Problems[1].Line {
		t.Errorf("problems not sorted by line: %v", r.Problems)
	}
}

func TestCheckDecodesConfig(t *testing.T) {
	r := Check([]byte(`
files:
  editorconfig:
    filename: .editorconfig
    url: https://example.com/e
  license:
    filename: LICENSE
    url: https://example.com/l
packs:
  go: [editorconfig, license]
`))
	if len(r.Problems) != 0 {
		t.Fatal(r.Problems)
	}
	want := &Config{
		Files: map[string]Entry{
			"editorconfig": {Filename: ".editorconfig", URL: "https://example.com/e"},
			"license":      {Filename: "LICENSE", URL: "https://example.com/l"},
		},
		Packs: map[string][]string{"go": {"editorconfig", "license"}},
	}
	if !reflect.DeepEqual(r.Config, want) {
		t.Errorf("got %+v, want %+v", r.Config, want)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		_, err := Load(filepath.Join(dir, "nope.yaml"))
		if err == nil || !strings.Contains(err.Error(), "config file not found") ||
			!strings.Contains(err.Error(), "obtain edit") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("valid", func(t *testing.T) {
		p := filepath.Join(dir, "ok.yaml")
		writeFile(t, p, "files:\n  a: {filename: a, url: u}\n")
		cfg, err := Load(p)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Files["a"].URL != "u" {
			t.Errorf("cfg = %+v", cfg)
		}
	})
	t.Run("problems fail the load and point to doctor", func(t *testing.T) {
		p := filepath.Join(dir, "bad.yaml")
		writeFile(t, p, "pack: {}\nbanana: 1\n")
		_, err := Load(p)
		if err == nil {
			t.Fatal("expected error")
		}
		for _, want := range []string{"2 problem(s)", "obtain doctor", `did you mean "packs"?`} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("err %q missing %q", err, want)
			}
		}
	})
}

func TestKeysAreSorted(t *testing.T) {
	c := &Config{
		Files: map[string]Entry{"b": {}, "c": {}, "a": {}},
		Packs: map[string][]string{"y": nil, "x": nil},
	}
	if got, want := c.FileKeys(), []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("FileKeys = %v", got)
	}
	if got, want := c.PackKeys(), []string{"x", "y"}; !reflect.DeepEqual(got, want) {
		t.Errorf("PackKeys = %v", got)
	}
}

func TestSuggest(t *testing.T) {
	allowed := []string{"files", "packs"}
	tests := []struct{ in, want string }{
		{"pack", "packs"},
		{"Packs", "packs"}, // case-insensitive
		{"fles", "files"},
		{"banana", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := suggest(tt.in, allowed); got != tt.want {
			t.Errorf("suggest(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "", 3},
		{"pack", "packs", 1},
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCreate(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a", "b", "config.yaml")
	wrote, err := Create(p)
	if err != nil || !wrote {
		t.Fatalf("Create = %v, %v", wrote, err)
	}
	if got, _ := os.ReadFile(p); string(got) != Starter {
		t.Errorf("content = %q", got)
	}
	if err := os.WriteFile(p, []byte("files: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wrote, err = Create(p)
	if err != nil || wrote {
		t.Fatalf("second Create = %v, %v", wrote, err)
	}
	if got, _ := os.ReadFile(p); string(got) != "files: {}\n" {
		t.Errorf("existing file overwritten: %q", got)
	}
}
