package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func server(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	return s
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFileDownloads(t *testing.T) {
	s := server(t, http.StatusOK, "root = true\n")
	dest := filepath.Join(t.TempDir(), ".editorconfig")
	if err := File(context.Background(), s.URL, dest, false, Vars{}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dest); got != "root = true\n" {
		t.Errorf("content = %q", got)
	}
	info, _ := os.Stat(dest)
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, want 0644", info.Mode().Perm())
	}
}

func TestFileCreatesParentDirs(t *testing.T) {
	s := server(t, http.StatusOK, "x")
	dest := filepath.Join(t.TempDir(), "a", "b", "f")
	if err := File(context.Background(), s.URL, dest, false, Vars{}); err != nil {
		t.Fatal(err)
	}
	if readFile(t, dest) != "x" {
		t.Error("file not written")
	}
}

func TestFileRefusesOverwriteWithoutForce(t *testing.T) {
	s := server(t, http.StatusOK, "new")
	dest := filepath.Join(t.TempDir(), "f")
	os.WriteFile(dest, []byte("old"), 0o644)

	err := File(context.Background(), s.URL, dest, false, Vars{})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
	if got := readFile(t, dest); got != "old" {
		t.Errorf("file was modified: %q", got)
	}
}

func TestFileForceOverwrites(t *testing.T) {
	s := server(t, http.StatusOK, "new")
	dest := filepath.Join(t.TempDir(), "f")
	os.WriteFile(dest, []byte("old"), 0o644)

	if err := File(context.Background(), s.URL, dest, true, Vars{}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dest); got != "new" {
		t.Errorf("content = %q", got)
	}
}

func TestFileBadStatusLeavesDestUntouched(t *testing.T) {
	s := server(t, http.StatusNotFound, "nope")
	dir := t.TempDir()
	dest := filepath.Join(dir, "f")
	os.WriteFile(dest, []byte("old"), 0o644)

	err := File(context.Background(), s.URL, dest, true, Vars{})
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("err = %v", err)
	}
	if got := readFile(t, dest); got != "old" {
		t.Errorf("dest clobbered: %q", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("leftover files in dir: %v", entries)
	}
}

func TestFileBadStatusDoesNotCreateDest(t *testing.T) {
	s := server(t, http.StatusInternalServerError, "")
	dest := filepath.Join(t.TempDir(), "f")
	if err := File(context.Background(), s.URL, dest, false, Vars{}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("dest should not exist, stat err = %v", err)
	}
}

func TestFileInvalidURL(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "f")
	if err := File(context.Background(), "http://127.0.0.1:0/none", dest, false, Vars{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestFileCanceledContext(t *testing.T) {
	s := server(t, http.StatusOK, "x")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := File(ctx, s.URL, filepath.Join(t.TempDir(), "f"), false, Vars{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestFileRawSkipsRendering(t *testing.T) {
	s := server(t, http.StatusOK, "hi {{ obtain.NAME }}\n")
	dest := filepath.Join(t.TempDir(), "f")
	if err := File(context.Background(), s.URL, dest, false, Vars{Raw: true}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dest); got != "hi {{ obtain.NAME }}\n" {
		t.Errorf("got %q", got)
	}
}

func TestRender(t *testing.T) {
	in := "(c) {{ obtain.YEAR }} {{obtain.NAME}} {{ obtain.std.YEAR }} ${{ secrets.X }} {{ NAME }} {{ other.NAME }}"
	vars := Vars{
		User: map[string]string{"YEAR": "user", "NAME": "Ada"},
		Std:  map[string]string{"YEAR": "2026"},
	}
	got, err := Render([]byte(in), vars)
	if err != nil {
		t.Fatal(err)
	}
	if want := "(c) user Ada 2026 ${{ secrets.X }} {{ NAME }} {{ other.NAME }}"; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderReportsAllMissing(t *testing.T) {
	_, err := Render([]byte("{{ obtain.B }} {{ obtain.A }} {{ obtain.B }}"), Vars{})
	if err == nil || !strings.Contains(err.Error(), "variable(s) A, B") {
		t.Errorf("err = %v", err)
	}
}

func TestRenderUnknownBuiltin(t *testing.T) {
	_, err := Render([]byte("{{ obtain.std.NOPE }}"), Vars{Std: map[string]string{"YEAR": "2026", "DATE": "x"}})
	if err == nil || !strings.Contains(err.Error(), "builtin(s) NOPE (available: DATE, YEAR)") {
		t.Errorf("err = %v", err)
	}
}

func TestBuiltins(t *testing.T) {
	std := Builtins(time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC), filepath.Join(t.TempDir(), "myproj"))
	want := map[string]string{"YEAR": "2026", "MONTH": "03", "DAY": "09", "DATE": "2026-03-09", "DIRNAME": "myproj"}
	for k, v := range want {
		if std[k] != v {
			t.Errorf("%s = %q, want %q", k, std[k], v)
		}
	}
}

func TestFileRendersVars(t *testing.T) {
	s := server(t, http.StatusOK, "Copyright {{ obtain.NAME }} {{ obtain.std.YEAR }}\n")
	dest := filepath.Join(t.TempDir(), "LICENSE")
	vars := Vars{User: map[string]string{"NAME": "Ada"}, Std: map[string]string{"YEAR": "2026"}}
	if err := File(context.Background(), s.URL, dest, false, vars); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dest); got != "Copyright Ada 2026\n" {
		t.Errorf("got %q", got)
	}
}

func TestFileMissingVarDoesNotWrite(t *testing.T) {
	s := server(t, http.StatusOK, "{{ obtain.NAME }}")
	dest := filepath.Join(t.TempDir(), "LICENSE")
	if err := File(context.Background(), s.URL, dest, false, Vars{}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Error("file written despite missing variable")
	}
}
