package app

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"voyage/internal/domain"
	"voyage/internal/ports"
)

type QueryService struct {
	repo      ports.NoteRepository
	strategy  ports.RelationStrategy
	formatter ports.OutputFormatter
}

type QueryOptions struct {
	Sort             string
	ShowDangling     bool
	Detailed         bool
	Tree             bool
	Depth            int
	DanglingPrefix   string
	ColorizeDangling func(string) string
}

func NewQuery(repo ports.NoteRepository, strategy ports.RelationStrategy, formatter ports.OutputFormatter) QueryService {
	return QueryService{repo: repo, strategy: strategy, formatter: formatter}
}

func (s QueryService) Render(index *domain.GraphIndex, targetPath string, opts QueryOptions) (string, error) {
	targetID := filepath.Clean(targetPath)
	note := index.Notes[targetID]
	if note == nil {
		return "", fmt.Errorf("target note not found in index: %s", targetPath)
	}

	if opts.Tree {
		if opts.Depth < 1 {
			return "", fmt.Errorf("depth must be >= 1")
		}
		return s.renderTree(note, index, opts)
	}

	rels := s.strategy.Related(note, index)
	rendered := make([]ports.RenderedRelation, 0, len(rels))
	for _, r := range rels {
		if r.Kind == "dangling" {
			if !opts.ShowDangling {
				continue
			}
			rendered = append(rendered, ports.RenderedRelation{Kind: "dangling", Raw: r.Raw, Title: r.Raw})
			continue
		}
		rn := index.Notes[r.ID]
		if rn == nil {
			continue
		}
		st, err := s.repo.Stat(rn.Path)
		if err != nil {
			return "", err
		}
		title := rn.Title
		if strings.TrimSpace(title) == "" {
			title = strings.TrimSuffix(filepath.Base(rn.Path), filepath.Ext(rn.Path))
		}
		rendered = append(rendered, ports.RenderedRelation{Kind: "resolved", Title: title, Path: rn.Path, Size: st.Size(), ModUnix: st.ModTimeUnix()})
	}

	if opts.Sort == "alpha" {
		sort.SliceStable(rendered, func(i, j int) bool {
			return strings.ToLower(rendered[i].Title) < strings.ToLower(rendered[j].Title)
		})
	}

	if opts.Detailed {
		return s.formatter.FormatDetailed(rendered), nil
	}
	return s.formatter.FormatSimple(rendered), nil
}

func (s QueryService) renderTree(root *domain.Note, index *domain.GraphIndex, opts QueryOptions) (string, error) {
	var b strings.Builder
	b.WriteString(noteLabel(root))
	b.WriteByte('\n')
	visited := map[string]bool{root.ID: true}
	if err := s.renderTreeChildren(&b, root, index, opts, 1, "", visited); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (s QueryService) renderTreeChildren(b *strings.Builder, parent *domain.Note, index *domain.GraphIndex, opts QueryOptions, level int, prefix string, visited map[string]bool) error {
	if level > opts.Depth {
		return nil
	}
	rels := s.strategy.Related(parent, index)
	children := make([]ports.RenderedRelation, 0, len(rels))
	for _, r := range rels {
		if r.Kind == "dangling" {
			if !opts.ShowDangling {
				continue
			}
			children = append(children, ports.RenderedRelation{Kind: "dangling", Raw: r.Raw, Title: r.Raw})
			continue
		}
		n := index.Notes[r.ID]
		if n == nil {
			continue
		}
		st, err := s.repo.Stat(n.Path)
		if err != nil {
			return err
		}
		children = append(children, ports.RenderedRelation{
			Kind:    "resolved",
			Title:   noteLabel(n),
			Path:    n.Path,
			Size:    st.Size(),
			ModUnix: st.ModTimeUnix(),
			Raw:     r.ID,
		})
	}

	if opts.Sort == "alpha" {
		sort.SliceStable(children, func(i, j int) bool {
			return strings.ToLower(children[i].Title) < strings.ToLower(children[j].Title)
		})
	}

	for i, c := range children {
		last := i == len(children)-1
		branch := "├── "
		nextPrefix := prefix + "│   "
		if last {
			branch = "└── "
			nextPrefix = prefix + "    "
		}

		if c.Kind == "dangling" {
			b.WriteString(prefix + branch + renderDangling(c.Raw, opts) + "\n")
			continue
		}

		line := c.Title
		if opts.Detailed {
			line = humanSize(c.Size) + "\t" + time.Unix(c.ModUnix, 0).Format(time.RFC3339) + "\t" + c.Path + "\t" + c.Title
		}
		childID := c.Raw
		if visited[childID] {
			b.WriteString(prefix + branch + line + " (cycle)\n")
			continue
		}
		b.WriteString(prefix + branch + line + "\n")
		if level < opts.Depth {
			visited[childID] = true
			if err := s.renderTreeChildren(b, index.Notes[childID], index, opts, level+1, nextPrefix, visited); err != nil {
				return err
			}
			delete(visited, childID)
		}
	}

	return nil
}

func renderDangling(raw string, opts QueryOptions) string {
	prefix := opts.DanglingPrefix
	if strings.TrimSpace(prefix) == "" {
		prefix = "⚠"
	}
	line := prefix + " " + raw
	if opts.ColorizeDangling != nil {
		return opts.ColorizeDangling(line)
	}
	return line
}

func noteLabel(n *domain.Note) string {
	if strings.TrimSpace(n.Title) != "" {
		return n.Title
	}
	return strings.TrimSuffix(filepath.Base(n.Path), filepath.Ext(n.Path))
}

func humanSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%dB", size)
	}
	d := float64(size)
	for _, s := range []string{"KB", "MB", "GB"} {
		d /= unit
		if d < unit {
			return fmt.Sprintf("%.1f%s", d, s)
		}
	}
	return fmt.Sprintf("%.1fTB", d/unit)
}
