package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunE2E_WithCorpus(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// 10 notes corpus.
	mustWrite("vault/index.md", "---\ntitle: Home\n---\n[[Project Alpha]] [[todo]] [[Ghost Note]] [[DupTitle]]")
	mustWrite("vault/project-alpha.md", "---\ntitle: Project Alpha\ntags:\n  - work\n  - alpha\n---\n[[Roadmap]]")
	mustWrite("vault/todo.md", "---\ntitle: Todo\n---\n[[Roadmap]]")
	mustWrite("vault/roadmap.md", "---\ntitle: Roadmap\n---\n[[Appendix]]")
	mustWrite("vault/appendix.md", "No frontmatter")
	mustWrite("vault/dup-title-a.md", "---\ntitle: DupTitle\n---\n")
	mustWrite("vault/dup-title-b.md", "---\ntitle: DupTitle\n---\n")
	mustWrite("vault/team.md", "---\ntitle: Team\n---\n")
	mustWrite("vault/invalid.md", "---\ntitle: broken\ntags: [a, b\n---\n[[Home]]")
	mustWrite("vault/nested/deep.md", "---\ntitle: Deep\n---\n")

	target := filepath.Join(root, "vault", "index.md")
	code, stdout, stderr := runAndCapture([]string{"--sort", "alpha", "--format", "simple", target})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}

	lines := nonEmptyLines(stdout)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %v", len(lines), lines)
	}
	got := strings.Join(lines, "\n")
	for _, want := range []string{"Project Alpha", "Todo", "⚠ Ghost Note", "⚠ DupTitle"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got %q", want, got)
		}
	}

	if !strings.Contains(stderr, "frontmatter invalide") {
		t.Fatalf("expected warning for invalid frontmatter, got %q", stderr)
	}
}

func TestRunE2E_NoDanglingDetailed(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("vault/index.md", "[[B Note]] [[missing]]")
	mustWrite("vault/b-note.md", "---\ntitle: B Note\n---\n")

	target := filepath.Join(root, "vault", "index.md")
	code, stdout, _ := runAndCapture([]string{"--long", "-D", target})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d", code)
	}
	if strings.Contains(stdout, "dangling") {
		t.Fatalf("did not expect dangling output, got %q", stdout)
	}
	if !strings.Contains(stdout, "b-note.md") || !strings.Contains(stdout, "B Note") {
		t.Fatalf("expected detailed output with path and title, got %q", stdout)
	}
}

func TestRunVersionFlag(t *testing.T) {
	prev := Version
	Version = "0.1.0-test"
	t.Cleanup(func() { Version = prev })

	code, stdout, stderr := runAndCapture([]string{"-v"})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}
	if strings.TrimSpace(stdout) != "0.1.0-test" {
		t.Fatalf("unexpected version output: %q", stdout)
	}
}

func TestRunVersionLongFlag(t *testing.T) {
	prev := Version
	Version = "0.2.0-test"
	t.Cleanup(func() { Version = prev })

	code, stdout, stderr := runAndCapture([]string{"--version"})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}
	if strings.TrimSpace(stdout) != "0.2.0-test" {
		t.Fatalf("unexpected version output: %q", stdout)
	}
}

func TestRunTreeDepthAndCycle(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("vault/index.md", "---\ntitle: Home\n---\n[[A]] [[missing]]")
	write("vault/a.md", "---\ntitle: A\n---\n[[B]]")
	write("vault/b.md", "---\ntitle: B\n---\n[[Home]]")

	target := filepath.Join(root, "vault", "index.md")
	code, stdout, stderr := runAndCapture([]string{"-t", "-n", "3", target})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}
	for _, want := range []string{"Home", "A", "B", "↺", "⚠ missing"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected tree output to contain %q, got %q", want, stdout)
		}
	}
}

func TestRunTreeDepthValidation(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "index.md")
	if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runAndCapture([]string{"-n", "2", target})
	if code != 2 || !strings.Contains(stderr, "only valid with --tree") {
		t.Fatalf("expected depth-without-tree error, got code=%d stderr=%q", code, stderr)
	}

	code, _, stderr = runAndCapture([]string{"--tree", "--depth", "0", target})
	if code != 2 || !strings.Contains(stderr, "must be >= 1") {
		t.Fatalf("expected depth>=1 error, got code=%d stderr=%q", code, stderr)
	}
}

func TestRunTreeLongAndNoDangling(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("vault/index.md", "[[Child]] [[missing]]")
	write("vault/child.md", "---\ntitle: Child\n---\n")
	target := filepath.Join(root, "vault", "index.md")

	code, stdout, stderr := runAndCapture([]string{"-t", "--long", "--no-dangling", target})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}
	if strings.Contains(stdout, "⚠ ") {
		t.Fatalf("did not expect dangling entries in tree: %q", stdout)
	}
	if !strings.Contains(stdout, "child.md") || !strings.Contains(stdout, "Child") {
		t.Fatalf("expected long tree output with path/title, got %q", stdout)
	}
}

func TestRunColorAlwaysOnDangling(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("vault/index.md", "[[missing]]")
	target := filepath.Join(root, "vault", "index.md")
	code, stdout, stderr := runAndCapture([]string{"--color", "always", target})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "\x1b[38;5;208m") || !strings.Contains(stdout, "\x1b[0m") {
		t.Fatalf("expected ansi color codes, got %q", stdout)
	}
}

func TestRunNoArgsShowsShortUsageAndVersion(t *testing.T) {
	prev := Version
	Version = "0.3.0-test"
	t.Cleanup(func() { Version = prev })

	code, stdout, stderr := runAndCapture(nil)
	if code != 0 {
		t.Fatalf("expected exit=0 got %d", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected no stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "vo 0.3.0-test") {
		t.Fatalf("expected version in help output, got %q", stderr)
	}
	if !strings.Contains(stderr, "usage: vo [options] <path-note.md>") {
		t.Fatalf("expected usage in help output, got %q", stderr)
	}
	if !strings.Contains(stderr, "run `vo -h` for full help") {
		t.Fatalf("expected short-help hint, got %q", stderr)
	}
	if !strings.Contains(stderr, "_    __") {
		t.Fatalf("expected figlet banner in short output, got %q", stderr)
	}
}

func TestRunTreeJSONSuccessAndDeterminism(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("vault/index.md", "---\ntitle: Home\n---\n[[A]] [[missing]]")
	write("vault/a.md", "---\ntitle: A\n---\n")
	target := filepath.Join(root, "vault", "index.md")

	code1, stdout1, stderr1 := runAndCapture([]string{"--tree", "--format", "json", "--depth", "2", target})
	if code1 != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code1, stderr1)
	}
	if strings.TrimSpace(stderr1) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr1)
	}
	var payload1 struct {
		SchemaVersion string `json:"schema_version"`
		Mode          string `json:"mode"`
		Root          struct {
			ID       string `json:"id"`
			Label    string `json:"label"`
			Path     string `json:"path"`
			Dangling bool   `json:"dangling"`
			NodeKind string `json:"node_kind"`
			Children []struct {
				ID       string `json:"id"`
				Label    string `json:"label"`
				Path     string `json:"path"`
				Dangling bool   `json:"dangling"`
				NodeKind string `json:"node_kind"`
				Children []any  `json:"children"`
			} `json:"children"`
		} `json:"root"`
	}
	if err := json.Unmarshal([]byte(stdout1), &payload1); err != nil {
		t.Fatalf("expected valid json, got err=%v body=%q", err, stdout1)
	}
	if payload1.SchemaVersion != "1.1.0" {
		t.Fatalf("expected schema_version=1.1.0, got %q", payload1.SchemaVersion)
	}
	if payload1.Mode != "links" {
		t.Fatalf("expected mode=links, got %q", payload1.Mode)
	}
	if payload1.Root.Label != "Home" || !filepath.IsAbs(payload1.Root.Path) || payload1.Root.Dangling || payload1.Root.NodeKind != "note" {
		t.Fatalf("unexpected root node: %+v", payload1.Root)
	}
	if len(payload1.Root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(payload1.Root.Children))
	}
	if !payload1.Root.Children[1].Dangling || payload1.Root.Children[1].Path != "" {
		t.Fatalf("expected dangling child with empty path, got %+v", payload1.Root.Children[1])
	}

	code2, stdout2, stderr2 := runAndCapture([]string{"--tree", "--format", "json", "--depth", "2", target})
	if code2 != 0 || strings.TrimSpace(stderr2) != "" {
		t.Fatalf("second run failed code=%d stderr=%q", code2, stderr2)
	}
	if stdout1 != stdout2 {
		t.Fatalf("expected deterministic json output, got\n1=%q\n2=%q", stdout1, stdout2)
	}
}

func TestRunModeTagsTreeDepthAndJSONKinds(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("vault/index.md", "---\ntitle: Home\ntags: [work]\n---\n")
	write("vault/a.md", "---\ntitle: A\ntags: [work, alpha]\n---\n")
	write("vault/b.md", "---\ntitle: B\ntags: [alpha]\n---\n")
	target := filepath.Join(root, "vault", "index.md")

	code, stdout, stderr := runAndCapture([]string{"--mode", "tags", "--tree", "--depth", "1", target})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "work") || !strings.Contains(stdout, "Home") || !strings.Contains(stdout, "A") {
		t.Fatalf("unexpected tags tree output: %q", stdout)
	}
	if strings.Contains(stdout, "alpha") {
		t.Fatalf("depth=1 should not recurse to second attribute hop, got %q", stdout)
	}

	code, stdout, stderr = runAndCapture([]string{"--mode", "tags", "--tree", "--depth", "1", "--format", "json", target})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}
	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Mode          string `json:"mode"`
		Root          struct {
			NodeKind string `json:"node_kind"`
			Children []struct {
				NodeKind string `json:"node_kind"`
				Path     string `json:"path"`
			} `json:"children"`
		} `json:"root"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("expected valid json, got err=%v body=%q", err, stdout)
	}
	if payload.SchemaVersion != "1.1.0" || payload.Mode != "tags" || payload.Root.NodeKind != "note" {
		t.Fatalf("unexpected payload header: %+v", payload)
	}
	if len(payload.Root.Children) == 0 || payload.Root.Children[0].NodeKind != "tag" || payload.Root.Children[0].Path != "" {
		t.Fatalf("expected first child to be tag node, got %+v", payload.Root.Children)
	}
}

func TestRunTreeJSONErrorsStructured(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "index.md")
	if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runAndCapture([]string{"--format", "json", target})
	if code != 2 {
		t.Fatalf("expected exit=2 got %d", code)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Error         struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("expected valid json error, got err=%v body=%q", err, stdout)
	}
	if payload.SchemaVersion != "1.0.0" || payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("unexpected error payload: %+v", payload)
	}
	if strings.Contains(stdout, "\"root\"") {
		t.Fatalf("unexpected root in error payload: %q", stdout)
	}
}

func TestRunShowPathInFlatAndTree(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("vault/index.md", "[[Child]]")
	write("vault/child.md", "---\ntitle: Child Title\n---\n")
	target := filepath.Join(root, "vault", "index.md")
	childPath := filepath.Join(root, "vault", "child.md")

	code, stdout, stderr := runAndCapture([]string{"--show", "path", target})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, childPath) || strings.Contains(stdout, "Child Title") {
		t.Fatalf("expected flat output to show path only, got %q", stdout)
	}

	code, stdout, stderr = runAndCapture([]string{"--tree", "--show", "path", target})
	if code != 0 {
		t.Fatalf("expected exit=0 got %d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, target) || !strings.Contains(stdout, childPath) || strings.Contains(stdout, "Child Title") {
		t.Fatalf("expected tree output to show paths, got %q", stdout)
	}
}

func TestRunInvalidShowValue(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "index.md")
	if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runAndCapture([]string{"--show", "both", target})
	if code != 2 || !strings.Contains(stderr, "invalid --show value") {
		t.Fatalf("expected invalid --show error, got code=%d stderr=%q", code, stderr)
	}
}

func runAndCapture(args []string) (int, string, string) {
	origOut := os.Stdout
	origErr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	code := Run(args)

	_ = wOut.Close()
	_ = wErr.Close()
	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	os.Stdout = origOut
	os.Stderr = origErr
	return code, string(outBytes), string(errBytes)
}

func nonEmptyLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, l := range raw {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}
