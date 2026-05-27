package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"voyage/internal/adapters/fs"
	"voyage/internal/adapters/logging"
	"voyage/internal/adapters/output"
	"voyage/internal/adapters/parser"
	"voyage/internal/adapters/strategy"
	"voyage/internal/app"
)

func Run(args []string) int {
	fsFlags := flag.NewFlagSet("vo", flag.ContinueOnError)
	fsFlags.SetOutput(os.Stderr)
	fsFlags.Usage = func() {
		printHelp(fsFlags)
	}
	sortOpt := fsFlags.String("sort", "discovery", "sort order: discovery|alpha")
	fsFlags.StringVar(sortOpt, "s", "discovery", "sort order: discovery|alpha")

	formatOpt := fsFlags.String("format", "simple", "output format: simple|detailed|json")
	fsFlags.StringVar(formatOpt, "f", "simple", "output format: simple|detailed|json")
	longFormat := fsFlags.Bool("long", false, "alias for --format detailed")
	fsFlags.BoolVar(longFormat, "l", false, "alias for --format detailed")

	showDangling := fsFlags.Bool("dangling", true, "show dangling links")
	fsFlags.BoolVar(showDangling, "d", true, "show dangling links")
	noDangling := fsFlags.Bool("no-dangling", false, "hide dangling links")
	fsFlags.BoolVar(noDangling, "D", false, "hide dangling links")

	logLevel := fsFlags.String("log-level", "warn", "log level: silent|warn|debug")
	fsFlags.StringVar(logLevel, "L", "warn", "log level: silent|warn|debug")
	treeView := fsFlags.Bool("tree", false, "render relations as a tree")
	fsFlags.BoolVar(treeView, "t", false, "render relations as a tree")
	depth := fsFlags.Int("depth", 1, "tree depth (>=1, tree mode only)")
	fsFlags.IntVar(depth, "n", 1, "tree depth (>=1, tree mode only)")
	colorOpt := fsFlags.String("color", "auto", "color mode: auto|always|never")
	fsFlags.StringVar(colorOpt, "c", "auto", "color mode: auto|always|never")

	showVersion := fsFlags.Bool("v", false, "print version")
	fsFlags.BoolVar(showVersion, "version", false, "print version")

	if len(args) == 0 {
		printHelp(fsFlags)
		return 0
	}
	if err := fsFlags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Println(Version)
		return 0
	}
	if *longFormat {
		*formatOpt = "detailed"
	}
	if *noDangling {
		*showDangling = false
	}
	depthFlagSet := hasDepthFlag(args)
	if depthFlagSet && !*treeView {
		return cliErr(*formatOpt == "json", "depth_requires_tree", "--depth is only valid with --tree", 2)
	}
	if *treeView && *depth < 1 {
		return cliErr(*formatOpt == "json", "invalid_depth", "--depth must be >= 1", 2)
	}
	if fsFlags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: vo [-s|--sort discovery|alpha] [-f|--format simple|detailed|json] [-l|--long] [-t|--tree] [-n|--depth N] [-d|--dangling] [-D|--no-dangling] [-L|--log-level silent|warn|debug] [-v|--version] <path-note>")
		return 2
	}
	if *sortOpt != "discovery" && *sortOpt != "alpha" {
		return cliErr(*formatOpt == "json", "invalid_sort", "invalid --sort value", 2)
	}
	if *formatOpt != "simple" && *formatOpt != "detailed" && *formatOpt != "json" {
		return cliErr(*formatOpt == "json", "invalid_format", "invalid --format value", 2)
	}
	if *formatOpt == "json" && !*treeView {
		return cliErr(true, "json_requires_tree", "--format json is only valid with --tree", 2)
	}
	if *colorOpt != "auto" && *colorOpt != "always" && *colorOpt != "never" {
		return cliErr(*formatOpt == "json", "invalid_color", "invalid --color value", 2)
	}
	target := filepath.Clean(fsFlags.Arg(0))
	if !strings.EqualFold(filepath.Ext(target), ".md") {
		return cliErr(*formatOpt == "json", "invalid_target_extension", "target must be a markdown file (.md)", 2)
	}
	if _, err := os.Stat(target); err != nil {
		return cliErr(*formatOpt == "json", "target_not_found", err.Error(), 2)
	}

	repo := fs.LocalRepo{}
	log := logging.New(*logLevel)
	indexer := app.NewIndexer(repo, parser.MarkdownParser{}, log)
	root := filepath.Dir(target)
	idx, err := indexer.Build(root)
	if err != nil {
		return cliErr(*formatOpt == "json", "index_build_failed", err.Error(), 1)
	}
	useColor := shouldUseColor(*colorOpt)
	colorizeDangling := func(s string) string {
		if !useColor {
			return s
		}
		return "\x1b[38;5;208m" + s + "\x1b[0m"
	}
	query := app.NewQuery(repo, strategy.Outgoing{}, output.NewTextFormatter(output.TextFormatterConfig{
		DanglingPrefix:   "⚠",
		ColorizeDangling: colorizeDangling,
	}))
	opts := app.QueryOptions{
		Sort:             *sortOpt,
		ShowDangling:     *showDangling,
		Detailed:         *formatOpt == "detailed",
		Tree:             *treeView,
		Depth:            *depth,
		DanglingPrefix:   "⚠",
		ColorizeDangling: colorizeDangling,
	}
	var out string
	if *formatOpt == "json" {
		out, err = query.RenderTreeJSON(idx, target, opts)
	} else {
		out, err = query.Render(idx, target, opts)
	}
	if err != nil {
		return cliErr(*formatOpt == "json", "query_failed", err.Error(), 1)
	}
	fmt.Print(out)
	return 0
}

type jsonErrorPayload struct {
	SchemaVersion string `json:"schema_version"`
	Error         struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details,omitempty"`
	} `json:"error"`
}

func cliErr(asJSON bool, code, msg string, exitCode int) int {
	if !asJSON {
		fmt.Fprintln(os.Stderr, msg)
		return exitCode
	}
	payload := jsonErrorPayload{SchemaVersion: app.TreeJSONSchemaVersion}
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

func printHelp(fsFlags *flag.FlagSet) {
	fmt.Fprint(os.Stderr, "\n _    __\n| |  / /___  __  ______ _____ ____\n| | / / __ \\/ / / / __ `/ __ `/ _ \\\n| |/ / /_/ / /_/ / /_/ / /_/ /  __/\n|___/\\____/\\__, /\\__,_/\\__, /\\___/\n          /____/      /____/\n\n")
	fmt.Fprintf(os.Stderr, "version %s\n\n", Version)
	fmt.Fprintln(os.Stderr, "usage: vo [-s|--sort discovery|alpha] [-f|--format simple|detailed|json] [-l|--long] [-t|--tree] [-n|--depth N] [-d|--dangling] [-D|--no-dangling] [-L|--log-level silent|warn|debug] [-c|--color auto|always|never] [-v|--version] <path-note>")
	fmt.Fprintln(os.Stderr)
	fsFlags.PrintDefaults()
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

func hasDepthFlag(args []string) bool {
	for i := range args {
		if args[i] == "--depth" || strings.HasPrefix(args[i], "--depth=") || args[i] == "-n" || strings.HasPrefix(args[i], "-n=") {
			return true
		}
	}
	return false
}
