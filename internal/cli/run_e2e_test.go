package cli

import (
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
	for _, want := range []string{"Home", "A", "B", "(cycle)", "⚠ missing"} {
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

func TestRunNoArgsShowsHelpBannerAndVersion(t *testing.T) {
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
	if !strings.Contains(stderr, "version 0.3.0-test") {
		t.Fatalf("expected version in help output, got %q", stderr)
	}
	if !strings.Contains(stderr, "usage: vo") {
		t.Fatalf("expected usage in help output, got %q", stderr)
	}
	if !strings.Contains(stderr, "_    __") {
		t.Fatalf("expected figlet banner in help output, got %q", stderr)
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
