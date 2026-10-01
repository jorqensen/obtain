package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/jorqensen/obtain/internal/config"
	"github.com/jorqensen/obtain/internal/fetch"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := newApp().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newApp() *cli.Command {
	return &cli.Command{
		Name:    "obtain",
		Usage:   "fetch frequently needed remote files",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "path to config file",
				Sources: cli.EnvVars("OBTAIN_CONFIG"),
			},
			&cli.StringFlag{
				Name:    "dir",
				Aliases: []string{"d"},
				Usage:   "directory to save files into",
				Value:   ".",
			},
			&cli.StringSliceFlag{
				Name:    "var",
				Aliases: []string{"v"},
				Usage:   "set a template variable, NAME=value",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "overwrite existing files",
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "file",
				Usage:     "fetch one or more files by key",
				ArgsUsage: "<key>...",
				Action:    fileAction,
			},
			{
				Name:      "pack",
				Usage:     "fetch all files in one or more packs",
				ArgsUsage: "<pack>...",
				Action:    packAction,
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "list configured files and packs",
				Action:  listAction,
			},
			{
				Name:   "read",
				Usage:  "show the config file in $PAGER (falls back to less, then stdout)",
				Action: readAction,
			},
			{
				Name:   "edit",
				Usage:  "open the config file in $VISUAL or $EDITOR (falls back to vi)",
				Action: editAction,
			},
			{
				Name:   "doctor",
				Usage:  "check that the config file exists and is valid",
				Action: doctorAction,
			},
		},
	}
}

func configPath(cmd *cli.Command) (string, error) {
	if path := cmd.String("config"); path != "" {
		return path, nil
	}
	return config.DefaultPath()
}

func loadConfig(cmd *cli.Command) (*config.Config, error) {
	path, err := configPath(cmd)
	if err != nil {
		return nil, err
	}
	return config.Load(path)
}

func readAction(ctx context.Context, cmd *cli.Command) error {
	path, err := configPath(cmd)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("config file not found: %s", path)
	}

	pager := strings.Fields(os.Getenv("PAGER"))
	if len(pager) == 0 {
		pager = []string{"less"}
	}
	if _, err := exec.LookPath(pager[0]); err != nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(data)
		return err
	}
	return runInteractive(ctx, pager, path)
}

func editAction(ctx context.Context, cmd *cli.Command) error {
	path, err := configPath(cmd)
	if err != nil {
		return err
	}
	editor := strings.Fields(os.Getenv("VISUAL"))
	if len(editor) == 0 {
		editor = strings.Fields(os.Getenv("EDITOR"))
	}
	if len(editor) == 0 {
		editor = []string{"vi"}
	}
	// Seed a starter config on first use; an existing file is left alone.
	if _, err := config.Create(path); err != nil {
		return err
	}
	return runInteractive(ctx, editor, path)
}

// runInteractive runs argv (a command plus its arguments) with path appended,
// attached to the terminal.
func runInteractive(ctx context.Context, argv []string, path string) error {
	c := exec.CommandContext(ctx, argv[0], append(argv[1:], path)...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		// %v, not %w: cli treats an *exec.ExitError as an exit code and would
		// exit silently instead of returning the error to main.
		return fmt.Errorf("%s: %v", argv[0], err)
	}
	return nil
}

func doctorAction(_ context.Context, cmd *cli.Command) error {
	path, err := configPath(cmd)
	if err != nil {
		return err
	}
	fmt.Println("config:", path)
	data, err := config.ReadFile(path)
	if err != nil {
		return err
	}
	r := config.Check(data)
	if len(r.Problems) > 0 {
		for _, p := range r.Problems {
			fmt.Println("  -", p)
		}
		return fmt.Errorf("%d problem(s) found", len(r.Problems))
	}
	if len(r.Config.Files) == 0 && len(r.Config.Packs) == 0 {
		fmt.Println("ok: no errors, but nothing configured (run `obtain edit` to add files)")
		return nil
	}
	fmt.Printf("ok: config is valid (%d files, %d packs)\n", len(r.Config.Files), len(r.Config.Packs))
	return nil
}

func fileAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() == 0 {
		return cli.ShowSubcommandHelp(cmd)
	}
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	return fetchKeys(ctx, cmd, cfg, cmd.Args().Slice())
}

func packAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() == 0 {
		return cli.ShowSubcommandHelp(cmd)
	}
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	var keys []string
	for _, name := range cmd.Args().Slice() {
		pack, ok := cfg.Packs[name]
		if !ok {
			return fmt.Errorf("unknown pack %q (see `obtain list`)", name)
		}
		keys = append(keys, pack...)
	}
	return fetchKeys(ctx, cmd, cfg, keys)
}

// mergeVars combines variable sources; later maps win.
func mergeVars(sources ...map[string]string) map[string]string {
	vars := map[string]string{}
	for _, src := range sources {
		maps.Copy(vars, src)
	}
	return vars
}

func parseVars(pairs []string) (map[string]string, error) {
	given := map[string]string{}
	for _, p := range pairs {
		name, value, ok := strings.Cut(p, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid --var %q, expected NAME=value", p)
		}
		given[name] = value
	}
	return given, nil
}

func fetchKeys(ctx context.Context, cmd *cli.Command, cfg *config.Config, keys []string) error {
	given, err := parseVars(cmd.StringSlice("var"))
	if err != nil {
		return err
	}
	std := fetch.Builtins(time.Now(), cmd.String("dir"))
	for _, key := range keys {
		entry, ok := cfg.Files[key]
		if !ok {
			return fmt.Errorf("unknown file %q (see `obtain list`)", key)
		}
		// Precedence: --var, then the file's own vars, then the top-level vars.
		vars := mergeVars(cfg.Vars, entry.Vars, given)
		dest := filepath.Join(cmd.String("dir"), entry.Filename)
		if err := fetch.File(ctx, entry.URL, dest, cmd.Bool("force"), fetch.Vars{User: vars, Std: std, Raw: entry.Raw}); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		fmt.Printf("%s -> %s\n", key, dest)
	}
	return nil
}

func listAction(_ context.Context, cmd *cli.Command) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if len(cfg.Files) == 0 && len(cfg.Packs) == 0 {
		fmt.Println("nothing configured (run `obtain edit` to add files)")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if len(cfg.Files) > 0 {
		fmt.Fprintf(w, "FILES (%d)\n", len(cfg.Files))
		fmt.Fprintln(w, "  KEY\tSAVES AS\tSOURCE")
		for _, key := range cfg.FileKeys() {
			e := cfg.Files[key]
			fmt.Fprintf(w, "  %s\t%s\t%s\n", key, e.Filename, e.URL)
		}
	}
	if len(cfg.Packs) > 0 {
		if len(cfg.Files) > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "PACKS (%d)\n", len(cfg.Packs))
		fmt.Fprintln(w, "  NAME\tFILES")
		for _, name := range cfg.PackKeys() {
			fmt.Fprintf(w, "  %s\t%s\n", name, strings.Join(cfg.Packs[name], ", "))
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}
