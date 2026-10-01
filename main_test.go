package main

import (
	"context"
	"github.com/jorqensen/obtain/internal/config"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// run executes the app with args and returns what it wrote to stdout.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := newApp().Run(context.Background(), append([]string{"obtain"}, args...))
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out), runErr
}

// fixture starts a file server and writes a config using it. It returns the
// config path and an empty output directory.
func fixture(t *testing.T) (cfgPath, outDir string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/editorconfig":
			io.WriteString(w, "root = true\n")
		case "/license":
			io.WriteString(w, "MIT\n")
		case "/std":
			io.WriteString(w, "{{ obtain.std.YEAR }} {{ obtain.std.DIRNAME }}\n")
		case "/templ":
			io.WriteString(w, "(c) {{ obtain.YEAR }} {{obtain.NAME}} ${{ secrets.X }}\n")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath = filepath.Join(t.TempDir(), "config.yaml")
	cfg := `
vars:
  NAME: Global Name
files:
  editorconfig:
    filename: .editorconfig
    url: ` + srv.URL + `/editorconfig
  license:
    filename: LICENSE
    url: ` + srv.URL + `/license
  templ:
    filename: TEMPL
    url: ` + srv.URL + `/templ
    vars:
      YEAR: "2026"
  broken:
    filename: broken
    url: ` + srv.URL + `/missing
packs:
  base: [editorconfig, license]
  bad: [editorconfig, broken, license]
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfgPath, t.TempDir()
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

func TestFileCommand(t *testing.T) {
	cfg, dir := fixture(t)
	out, err := run(t, "-c", cfg, "-d", dir, "file", "editorconfig", "license")
	if err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dir, ".editorconfig"), "root = true\n")
	assertFile(t, filepath.Join(dir, "LICENSE"), "MIT\n")
	if !strings.Contains(out, "editorconfig ->") || !strings.Contains(out, "license ->") {
		t.Errorf("output = %q", out)
	}
}

func TestFileCommandUnknownKey(t *testing.T) {
	cfg, dir := fixture(t)
	_, err := run(t, "-c", cfg, "-d", dir, "file", "nope")
	if err == nil || !strings.Contains(err.Error(), `unknown file "nope"`) {
		t.Errorf("err = %v", err)
	}
}

func TestFileCommandRefusesOverwriteUnlessForced(t *testing.T) {
	cfg, dir := fixture(t)
	dest := filepath.Join(dir, "LICENSE")
	os.WriteFile(dest, []byte("mine"), 0o644)

	_, err := run(t, "-c", cfg, "-d", dir, "file", "license")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
	assertFile(t, dest, "mine")

	if _, err := run(t, "-c", cfg, "-d", dir, "--force", "file", "license"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, dest, "MIT\n")
}

func TestFileCommandWithoutArgsShowsHelp(t *testing.T) {
	cfg, dir := fixture(t)
	if _, err := run(t, "-c", cfg, "-d", dir, "file"); err != nil {
		t.Errorf("err = %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("nothing should be fetched, got %v", entries)
	}
}

func TestPackCommand(t *testing.T) {
	cfg, dir := fixture(t)
	if _, err := run(t, "-c", cfg, "-d", dir, "pack", "base"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dir, ".editorconfig"), "root = true\n")
	assertFile(t, filepath.Join(dir, "LICENSE"), "MIT\n")
}

func TestPackCommandUnknownPack(t *testing.T) {
	cfg, dir := fixture(t)
	_, err := run(t, "-c", cfg, "-d", dir, "pack", "nope")
	if err == nil || !strings.Contains(err.Error(), `unknown pack "nope"`) {
		t.Errorf("err = %v", err)
	}
}

func TestPackCommandStopsAtFirstFailure(t *testing.T) {
	cfg, dir := fixture(t)
	_, err := run(t, "-c", cfg, "-d", dir, "pack", "bad")
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".editorconfig")); err != nil {
		t.Error("file before the failure should have been fetched")
	}
	if _, err := os.Stat(filepath.Join(dir, "LICENSE")); !os.IsNotExist(err) {
		t.Error("file after the failure should not have been fetched")
	}
}

func TestPackHasNoBundleAlias(t *testing.T) {
	for _, c := range newApp().Commands {
		if c.Name == "bundle" {
			t.Error("unexpected bundle command")
		}
		for _, a := range c.Aliases {
			if a == "bundle" {
				t.Errorf("command %q has a bundle alias", c.Name)
			}
		}
	}
}

func TestConfigFromEnv(t *testing.T) {
	cfg, dir := fixture(t)
	t.Setenv("OBTAIN_CONFIG", cfg)
	if _, err := run(t, "-d", dir, "file", "license"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dir, "LICENSE"), "MIT\n")
}

func TestListCommand(t *testing.T) {
	cfg, _ := fixture(t)
	out, err := run(t, "-c", cfg, "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"FILES (4)", "PACKS (2)", "KEY", "SAVES AS", "SOURCE", "editorconfig", ".editorconfig", "base", "editorconfig, license"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	// alias
	if out2, err := run(t, "-c", cfg, "ls"); err != nil || out2 != out {
		t.Errorf("ls alias differs (err=%v)", err)
	}
}

func TestListCommandEmptyConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	os.WriteFile(p, nil, 0o644)
	out, err := run(t, "-c", p, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "nothing configured") {
		t.Errorf("output = %q", out)
	}
}

func TestDoctorCommand(t *testing.T) {
	write := func(content string) string {
		p := filepath.Join(t.TempDir(), "c.yaml")
		os.WriteFile(p, []byte(content), 0o644)
		return p
	}

	t.Run("valid", func(t *testing.T) {
		cfg, _ := fixture(t)
		out, err := run(t, "-c", cfg, "doctor")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "ok: config is valid (4 files, 2 packs)") {
			t.Errorf("output = %q", out)
		}
	})
	t.Run("empty config is ok but noted", func(t *testing.T) {
		out, err := run(t, "-c", write(""), "doctor")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "no errors, but nothing configured") {
			t.Errorf("output = %q", out)
		}
	})
	t.Run("no packs is not an error", func(t *testing.T) {
		if _, err := run(t, "-c", write("files:\n  a: {filename: a, url: u}\n"), "doctor"); err != nil {
			t.Error(err)
		}
	})
	t.Run("typo reports suggestion and fails", func(t *testing.T) {
		out, err := run(t, "-c", write("files:\n  a: {filename: a, url: u}\npack:\n  p: [a]\n"), "doctor")
		if err == nil || !strings.Contains(err.Error(), "1 problem(s) found") {
			t.Errorf("err = %v", err)
		}
		if !strings.Contains(out, `unknown key "pack", did you mean "packs"?`) {
			t.Errorf("output = %q", out)
		}
	})
	t.Run("missing file fails", func(t *testing.T) {
		_, err := run(t, "-c", filepath.Join(t.TempDir(), "nope.yaml"), "doctor")
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestReadCommand(t *testing.T) {
	cfg, _ := fixture(t)
	t.Setenv("PAGER", "cat")
	out, err := run(t, "-c", cfg, "read")
	if err != nil {
		t.Fatal(err)
	}
	// The pager writes straight to the terminal fd, not our pipe, so only
	// check success here; the missing-pager fallback is covered below.
	_ = out
}

func TestReadCommandFallsBackToStdout(t *testing.T) {
	cfg, _ := fixture(t)
	t.Setenv("PAGER", "definitely-not-a-real-pager")
	out, err := run(t, "-c", cfg, "read")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "editorconfig:") {
		t.Errorf("output = %q", out)
	}
}

func TestReadCommandMissingConfig(t *testing.T) {
	_, err := run(t, "-c", filepath.Join(t.TempDir(), "nope.yaml"), "read")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("err = %v", err)
	}
}

func TestEditCommandPrefersVisualAndCreatesFile(t *testing.T) {
	// "touch" as the editor creates the file, like a real editor saving it.
	p := filepath.Join(t.TempDir(), "sub", "config.yaml")
	t.Setenv("VISUAL", "touch")
	t.Setenv("EDITOR", "definitely-not-an-editor")
	if _, err := run(t, "-c", p, "edit"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("config not created: %v", err)
	}
}

func TestEditCommandSeedsStarterConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.yaml")
	t.Setenv("VISUAL", "true")
	if _, err := run(t, "-c", p, "edit"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, p, config.Starter)
	if r := config.Check([]byte(config.Starter)); len(r.Problems) > 0 {
		t.Errorf("starter config has problems: %v", r.Problems)
	}
}

func TestEditCommandKeepsExistingConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte("files: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", "true")
	if _, err := run(t, "-c", p, "edit"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, p, "files: {}\n")
}

func TestEditCommandFallsBackToEditor(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "touch")
	if _, err := run(t, "-c", p, "edit"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("config not created: %v", err)
	}
}

func TestEditCommandEditorFailurePropagates(t *testing.T) {
	t.Setenv("VISUAL", "false")
	_, err := run(t, "-c", filepath.Join(t.TempDir(), "c.yaml"), "edit")
	if err == nil || !strings.Contains(err.Error(), "false: exit status 1") {
		t.Errorf("err = %v", err)
	}
}

func TestNoCommandShowsHelp(t *testing.T) {
	out, err := run(t)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"file", "pack", "list", "read", "edit", "doctor"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q:\n%s", want, out)
		}
	}
}

func TestFileCommandRendersVars(t *testing.T) {
	cfg, dir := fixture(t)
	if _, err := run(t, "-c", cfg, "-d", dir, "file", "templ"); err != nil {
		t.Fatal(err)
	}
	// NAME from top-level vars, YEAR from the file's vars; ${{ }} untouched.
	assertFile(t, filepath.Join(dir, "TEMPL"), "(c) 2026 Global Name ${{ secrets.X }}\n")
}

func TestFileCommandVarPrecedence(t *testing.T) {
	cfg, dir := fixture(t)
	if _, err := run(t, "-c", cfg, "-d", dir, "-v", "NAME=Ada", "-v", "YEAR=1999", "file", "templ"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dir, "TEMPL"), "(c) 1999 Ada ${{ secrets.X }}\n")
}

func TestFileCommandFileVarsOverrideGlobal(t *testing.T) {
	cfg, dir := fixture(t)
	data, _ := os.ReadFile(cfg)
	data = []byte(strings.Replace(string(data), `YEAR: "2026"`, "YEAR: \"2026\"\n      NAME: File Name", 1))
	os.WriteFile(cfg, data, 0o644)
	if _, err := run(t, "-c", cfg, "-d", dir, "file", "templ"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dir, "TEMPL"), "(c) 2026 File Name ${{ secrets.X }}\n")
}

func TestFileCommandMissingVar(t *testing.T) {
	cfg, dir := fixture(t)
	data, _ := os.ReadFile(cfg)
	os.WriteFile(cfg, []byte(strings.Replace(string(data), "  NAME: Global Name\n", "", 1)), 0o644)
	_, err := run(t, "-c", cfg, "-d", dir, "file", "templ")
	if err == nil || !strings.Contains(err.Error(), "no value for variable(s) NAME") {
		t.Errorf("err = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "TEMPL")); statErr == nil {
		t.Error("file written despite missing variable")
	}
}

func TestFileCommandBuiltins(t *testing.T) {
	cfg, dir := fixture(t)
	data, _ := os.ReadFile(cfg)
	os.WriteFile(cfg, []byte(strings.Replace(string(data), "/templ", "/std", 1)), 0o644)
	if _, err := run(t, "-c", cfg, "-d", dir, "file", "templ"); err != nil {
		t.Fatal(err)
	}
	want := time.Now().Format("2006") + " " + filepath.Base(dir) + "\n"
	assertFile(t, filepath.Join(dir, "TEMPL"), want)
}

func TestFileCommandRaw(t *testing.T) {
	cfg, dir := fixture(t)
	data, _ := os.ReadFile(cfg)
	data = []byte(strings.Replace(string(data), "    vars:\n      YEAR: \"2026\"\n", "    raw: true\n", 1))
	os.WriteFile(cfg, data, 0o644)
	if _, err := run(t, "-c", cfg, "-d", dir, "file", "templ"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dir, "TEMPL"), "(c) {{ obtain.YEAR }} {{obtain.NAME}} ${{ secrets.X }}\n")
}

func TestFileCommandBadVarFlag(t *testing.T) {
	cfg, dir := fixture(t)
	_, err := run(t, "-c", cfg, "-d", dir, "-v", "NAME", "file", "templ")
	if err == nil || !strings.Contains(err.Error(), "expected NAME=value") {
		t.Errorf("err = %v", err)
	}
}

func TestVersionFlag(t *testing.T) {
	out, err := run(t, "--version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, version) {
		t.Errorf("--version output %q does not contain %q", out, version)
	}
}
