package fetch

import (
	"os/user"
	"path/filepath"
	"time"
)

// Builtins returns the values for {{ obtain.std.NAME }} placeholders: YEAR,
// MONTH, DAY, DATE (YYYY-MM-DD), USER (the OS username, if known) and DIRNAME
// (the name of the directory files are saved into).
func Builtins(now time.Time, dir string) map[string]string {
	std := map[string]string{
		"YEAR":  now.Format("2006"),
		"MONTH": now.Format("01"),
		"DAY":   now.Format("02"),
		"DATE":  now.Format("2006-01-02"),
	}
	if abs, err := filepath.Abs(dir); err == nil {
		std["DIRNAME"] = filepath.Base(abs)
	}
	if u, err := user.Current(); err == nil {
		std["USER"] = u.Username
	}
	return std
}
