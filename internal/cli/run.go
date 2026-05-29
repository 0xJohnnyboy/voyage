package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"voyage/internal/adapters/fs"
	"voyage/internal/adapters/logging"
	"voyage/internal/adapters/output"
	"voyage/internal/adapters/parser"
	"voyage/internal/adapters/strategy"
	"voyage/internal/app"
)

const (
	ansiReset          = "\x1b[0m"
	ansiColorDangling  = "\x1b[38;5;208m"
	ansiColorCycleMark = "\x1b[38;5;39m"
	asciiBanner        = "\n _    __\n| |  / /___  __  ______ _____ ____\n| | / / __ \\/ / / / __ `/ __ `/ _ \\\n| |/ / /_/ / /_/ / /_/ / /_/ /  __/\n|___/\\____/\\__, /\\__,_/\\__, /\\___/\n          /____/      /____/\n\n"
)

type jsonErrorPayload struct {
	SchemaVersion string `json:"schema_version"`
	Error         struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details,omitempty"`
	} `json:"error"`
}

type runError struct {
	asJSON   bool
	code     string
	message  string
	exitCode int
}

func (e *runError) Error() string {
	return e.message
}

type runConfig struct {
	sortOpt      string
	formatOpt    string
	showOpt      string
	longFormat   bool
	showDangling bool
	noDangling   bool
	logLevel     string
	treeView     bool
	modeOpt      string
	depth        int
	colorOpt     string
	showVersion  bool
}

func Run(args []string) int {
	if len(args) == 0 {
		printShortUsage()
		return 0
	}

	cmd := newRootCmd()
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		if re, ok := err.(*runError); ok {
			return cliErr(re.asJSON, re.code, re.message, re.exitCode)
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	cfg := &runConfig{}

	cmd := &cobra.Command{
		Use:           "vo [options] <path-note.md>",
		Short:         "Relational navigation CLI for Markdown notes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRun(cmd, args, cfg)
		},
	}
	cmd.SetHelpFunc(func(_ *cobra.Command, _ []string) {
		printHelp()
	})

	cmd.Flags().StringVarP(&cfg.sortOpt, "sort", "s", "discovery", "sort order: discovery|alpha")
	cmd.Flags().StringVarP(&cfg.formatOpt, "format", "f", "simple", "output format: simple|detailed|json")
	cmd.Flags().StringVarP(&cfg.showOpt, "show", "w", "title", "display field: title|path")
	cmd.Flags().BoolVarP(&cfg.longFormat, "long", "l", false, "alias for --format detailed")
	cmd.Flags().BoolVarP(&cfg.showDangling, "dangling", "d", true, "show dangling links")
	cmd.Flags().BoolVarP(&cfg.noDangling, "no-dangling", "D", false, "hide dangling links")
	cmd.Flags().StringVarP(&cfg.logLevel, "log-level", "L", "warn", "log level: silent|warn|debug")
	cmd.Flags().BoolVarP(&cfg.treeView, "tree", "t", false, "render relations as a tree")
	cmd.Flags().StringVarP(&cfg.modeOpt, "mode", "m", "links", "relation mode: links|tags|categories")
	cmd.Flags().IntVarP(&cfg.depth, "depth", "n", 1, "tree depth (>=1, tree mode only)")
	cmd.Flags().StringVarP(&cfg.colorOpt, "color", "c", "auto", "color mode: auto|always|never")
	cmd.Flags().BoolVarP(&cfg.showVersion, "version", "v", false, "print version")

	return cmd
}

func executeRun(cmd *cobra.Command, args []string, cfg *runConfig) error {
	if cfg.showVersion {
		fmt.Println(Version)
		return nil
	}
	if cfg.longFormat {
		cfg.formatOpt = "detailed"
	}
	if cfg.noDangling {
		cfg.showDangling = false
	}
	depthFlagSet := cmd.Flags().Changed("depth")
	if depthFlagSet && !cfg.treeView {
		return &runError{asJSON: cfg.formatOpt == "json", code: "depth_requires_tree", message: "--depth is only valid with --tree", exitCode: 2}
	}
	if cfg.treeView && cfg.depth < 1 {
		return &runError{asJSON: cfg.formatOpt == "json", code: "invalid_depth", message: "--depth must be >= 1", exitCode: 2}
	}
	if len(args) != 1 {
		return &runError{asJSON: false, code: "usage", message: "usage: vo [-s|--sort discovery|alpha] [-f|--format simple|detailed|json] [-w|--show title|path] [-m|--mode links|tags|categories] [-l|--long] [-t|--tree] [-n|--depth N] [-d|--dangling] [-D|--no-dangling] [-L|--log-level silent|warn|debug] [-v|--version] <path-note>", exitCode: 2}
	}
	if cfg.sortOpt != "discovery" && cfg.sortOpt != "alpha" {
		return &runError{asJSON: cfg.formatOpt == "json", code: "invalid_sort", message: "invalid --sort value", exitCode: 2}
	}
	if cfg.modeOpt != "links" && cfg.modeOpt != "tags" && cfg.modeOpt != "categories" {
		return &runError{asJSON: cfg.formatOpt == "json", code: "invalid_mode", message: "invalid --mode value", exitCode: 2}
	}
	if cfg.formatOpt != "simple" && cfg.formatOpt != "detailed" && cfg.formatOpt != "json" {
		return &runError{asJSON: cfg.formatOpt == "json", code: "invalid_format", message: "invalid --format value", exitCode: 2}
	}
	if cfg.showOpt != "title" && cfg.showOpt != "path" {
		return &runError{asJSON: cfg.formatOpt == "json", code: "invalid_show", message: "invalid --show value", exitCode: 2}
	}
	if cfg.formatOpt == "json" && !cfg.treeView {
		return &runError{asJSON: true, code: "json_requires_tree", message: "--format json is only valid with --tree", exitCode: 2}
	}
	if cfg.colorOpt != "auto" && cfg.colorOpt != "always" && cfg.colorOpt != "never" {
		return &runError{asJSON: cfg.formatOpt == "json", code: "invalid_color", message: "invalid --color value", exitCode: 2}
	}
	target := filepath.Clean(args[0])
	if !strings.EqualFold(filepath.Ext(target), ".md") {
		return &runError{asJSON: cfg.formatOpt == "json", code: "invalid_target_extension", message: "target must be a markdown file (.md)", exitCode: 2}
	}
	if _, err := os.Stat(target); err != nil {
		return &runError{asJSON: cfg.formatOpt == "json", code: "target_not_found", message: err.Error(), exitCode: 2}
	}

	repo := fs.LocalRepo{}
	log := logging.New(cfg.logLevel)
	indexer := app.NewIndexer(repo, parser.MarkdownParser{}, log)
	root := filepath.Dir(target)
	idx, err := indexer.Build(root)
	if err != nil {
		return &runError{asJSON: cfg.formatOpt == "json", code: "index_build_failed", message: err.Error(), exitCode: 1}
	}

	useColor := shouldUseColor(cfg.colorOpt)
	colorizeDangling := func(s string) string {
		if !useColor {
			return s
		}
		return ansiColorDangling + s + ansiReset
	}
	colorizeCycle := func(s string) string {
		if !useColor {
			return s
		}
		return ansiColorCycleMark + s + ansiReset
	}

	query := app.NewQuery(repo, strategy.Outgoing{}, output.NewTextFormatter(output.TextFormatterConfig{
		DanglingPrefix:   "⚠",
		ColorizeDangling: colorizeDangling,
	}))
	opts := app.QueryOptions{
		Sort:             cfg.sortOpt,
		ShowDangling:     cfg.showDangling,
		Detailed:         cfg.formatOpt == "detailed",
		Show:             cfg.showOpt,
		Tree:             cfg.treeView,
		Depth:            cfg.depth,
		Mode:             cfg.modeOpt,
		DanglingPrefix:   "⚠",
		CycleMarker:      "↺",
		ColorizeDangling: colorizeDangling,
		ColorizeCycle:    colorizeCycle,
	}

	var out string
	if cfg.formatOpt == "json" {
		out, err = query.RenderTreeJSON(idx, target, opts)
	} else {
		out, err = query.Render(idx, target, opts)
	}
	if err != nil {
		return &runError{asJSON: cfg.formatOpt == "json", code: "query_failed", message: err.Error(), exitCode: 1}
	}
	fmt.Print(out)
	return nil
}

func cliErr(asJSON bool, code, msg string, exitCode int) int {
	if !asJSON {
		fmt.Fprintln(os.Stderr, msg)
		return exitCode
	}
	payload := jsonErrorPayload{SchemaVersion: app.ErrorJSONSchemaVersion}
	payload.Error.Code = code
	payload.Error.Message = msg
	b, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, msg)
		return exitCode
	}
	fmt.Fprintln(os.Stdout, string(b))
	return exitCode
}

func printHelp() {
	fmt.Fprint(os.Stderr, asciiBanner)
	fmt.Fprintf(os.Stderr, "version %s\n\n", Version)
	fmt.Fprintln(os.Stderr, "usage: vo [-s|--sort discovery|alpha] [-f|--format simple|detailed|json] [-w|--show title|path] [-m|--mode links|tags|categories] [-l|--long] [-t|--tree] [-n|--depth N] [-d|--dangling] [-D|--no-dangling] [-L|--log-level silent|warn|debug] [-c|--color auto|always|never] [-v|--version] <path-note>")
	fmt.Fprintln(os.Stderr)
	printOptionHelp()
}

func printShortUsage() {
	fmt.Fprint(os.Stderr, asciiBanner)
	fmt.Fprintf(os.Stderr, "vo %s\n", Version)
	fmt.Fprintln(os.Stderr, "usage: vo [options] <path-note.md>")
	fmt.Fprintln(os.Stderr, "run `vo -h` for full help")
}

func printOptionHelp() {
	lines := []string{
		"  -v, --version                print version and exit",
		"  -s, --sort                   sort order: discovery|alpha (default: discovery)",
		"  -f, --format                 output format: simple|detailed|json (default: simple)",
		"  -w, --show                   display field: title|path (default: title)",
		"  -m, --mode                   relation mode: links|tags|categories (default: links)",
		"  -l, --long                   alias for --format detailed",
		"  -d, --dangling               show dangling links (default: true)",
		"  -D, --no-dangling            hide dangling links",
		"  -L, --log-level              log level: silent|warn|debug (default: warn)",
		"  -t, --tree                   render relations as a tree",
		"  -n, --depth                  tree depth (>=1, tree mode only, default: 1)",
		"  -c, --color                  color mode: auto|always|never (default: auto)",
	}
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, line)
	}
}

func shouldUseColor(mode string) bool {
	if mode == "always" {
		return true
	}
	if mode == "never" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
