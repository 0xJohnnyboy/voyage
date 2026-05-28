package app

import (
	"encoding/json"
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
	Mode             string
	DanglingPrefix   string
	CycleMarker      string
	ColorizeDangling func(string) string
	ColorizeCycle    func(string) string
}

const TreeJSONSchemaVersion = "1.1.0"
const ErrorJSONSchemaVersion = "1.0.0"

type TreeJSONNode struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Path     string         `json:"path"`
	Dangling bool           `json:"dangling"`
	NodeKind string         `json:"node_kind"`
	Children []TreeJSONNode `json:"children"`
}

type TreeJSONSuccess struct {
	SchemaVersion string       `json:"schema_version"`
	Mode          string       `json:"mode"`
	Root          TreeJSONNode `json:"root"`
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

	mode := strings.TrimSpace(opts.Mode)
	if mode == "" {
		mode = "links"
	}

	if opts.Tree {
		if opts.Depth < 1 {
			return "", fmt.Errorf("depth must be >= 1")
		}
		if mode == "links" {
			return s.renderLinksTree(note, index, opts)
		}
		return s.renderSemanticTree(note, index, opts, mode)
	}

	if mode == "links" {
		return s.renderLinksFlat(note, index, opts)
	}
	return s.renderSemanticFlat(note, index, opts, mode)
}

func (s QueryService) RenderTreeJSON(index *domain.GraphIndex, targetPath string, opts QueryOptions) (string, error) {
	targetID := filepath.Clean(targetPath)
	note := index.Notes[targetID]
	if note == nil {
		return "", fmt.Errorf("target note not found in index: %s", targetPath)
	}
	if opts.Depth < 1 {
		return "", fmt.Errorf("depth must be >= 1")
	}

	mode := strings.TrimSpace(opts.Mode)
	if mode == "" {
		mode = "links"
	}

	var root TreeJSONNode
	var err error
	if mode == "links" {
		root, err = s.buildLinksTreeJSONNode(note, index, opts, 0, map[string]bool{note.ID: true})
	} else {
		root, err = s.buildSemanticTreeJSONNode(note, index, opts, mode, opts.Depth, map[string]bool{note.ID: true})
	}
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(TreeJSONSuccess{SchemaVersion: TreeJSONSchemaVersion, Mode: mode, Root: root})
	if err != nil {
		return "", err
	}
	return string(out) + "\n", nil
}

func (s QueryService) renderLinksFlat(note *domain.Note, index *domain.GraphIndex, opts QueryOptions) (string, error) {
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
		rendered = append(rendered, ports.RenderedRelation{Kind: "resolved", Title: noteLabel(rn), Path: rn.Path, Size: st.Size(), ModUnix: st.ModTimeUnix()})
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

func (s QueryService) renderSemanticFlat(note *domain.Note, index *domain.GraphIndex, opts QueryOptions, mode string) (string, error) {
	terms := termsForMode(note, mode)
	if opts.Sort == "alpha" {
		sort.SliceStable(terms, func(i, j int) bool {
			return strings.ToLower(terms[i]) < strings.ToLower(terms[j])
		})
	}

	var b strings.Builder
	for _, term := range terms {
		b.WriteString(term)
		b.WriteByte('\n')
		notes := notesForTerm(index, mode, term)
		for _, n := range notes {
			if opts.Detailed {
				st, err := s.repo.Stat(n.Path)
				if err != nil {
					return "", err
				}
				b.WriteString("  - " + humanSize(st.Size()) + "\t" + time.Unix(st.ModTimeUnix(), 0).Format(time.RFC3339) + "\t" + n.Path + "\t" + noteLabel(n) + "\n")
			} else {
				b.WriteString("  - " + noteLabel(n) + "\n")
			}
		}
	}
	return b.String(), nil
}

func (s QueryService) renderLinksTree(root *domain.Note, index *domain.GraphIndex, opts QueryOptions) (string, error) {
	var b strings.Builder
	b.WriteString(noteLabel(root))
	b.WriteByte('\n')
	visited := map[string]bool{root.ID: true}
	if err := s.renderLinksTreeChildren(&b, root, index, opts, 1, "", visited); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (s QueryService) renderSemanticTree(root *domain.Note, index *domain.GraphIndex, opts QueryOptions, mode string) (string, error) {
	var b strings.Builder
	b.WriteString(noteLabel(root))
	b.WriteByte('\n')
	visited := map[string]bool{root.ID: true}
	if err := s.renderSemanticTreeNote(&b, root, index, opts, mode, opts.Depth, "", visited); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (s QueryService) renderSemanticTreeNote(b *strings.Builder, note *domain.Note, index *domain.GraphIndex, opts QueryOptions, mode string, hops int, prefix string, visited map[string]bool) error {
	if hops < 1 {
		return nil
	}
	terms := termsForMode(note, mode)
	if opts.Sort == "alpha" {
		sort.SliceStable(terms, func(i, j int) bool {
			return strings.ToLower(terms[i]) < strings.ToLower(terms[j])
		})
	}

	for ti, term := range terms {
		termLast := ti == len(terms)-1
		termBranch := "├── "
		termPrefix := prefix + "│   "
		if termLast {
			termBranch = "└── "
			termPrefix = prefix + "    "
		}
		b.WriteString(prefix + termBranch + term + "\n")

		notes := notesForTerm(index, mode, term)
		for ni, child := range notes {
			noteLast := ni == len(notes)-1
			noteBranch := "├── "
			nextPrefix := termPrefix + "│   "
			if noteLast {
				noteBranch = "└── "
				nextPrefix = termPrefix + "    "
			}
			line := noteLabel(child)
			if opts.Detailed {
				st, err := s.repo.Stat(child.Path)
				if err != nil {
					return err
				}
				line = humanSize(st.Size()) + "\t" + time.Unix(st.ModTimeUnix(), 0).Format(time.RFC3339) + "\t" + child.Path + "\t" + noteLabel(child)
			}
				if visited[child.ID] {
					b.WriteString(termPrefix + noteBranch + line + " " + renderCycle(opts) + "\n")
					continue
				}
			b.WriteString(termPrefix + noteBranch + line + "\n")
			visited[child.ID] = true
			if err := s.renderSemanticTreeNote(b, child, index, opts, mode, hops-1, nextPrefix, visited); err != nil {
				return err
			}
			delete(visited, child.ID)
		}
	}
	return nil
}

func (s QueryService) buildLinksTreeJSONNode(note *domain.Note, index *domain.GraphIndex, opts QueryOptions, level int, visited map[string]bool) (TreeJSONNode, error) {
	node := TreeJSONNode{ID: note.ID, Label: noteLabel(note), Path: note.Path, Dangling: false, NodeKind: "note", Children: []TreeJSONNode{}}
	if level >= opts.Depth {
		return node, nil
	}

	rels := s.strategy.Related(note, index)
	resolved := make([]*domain.Note, 0, len(rels))
	dangling := make([]string, 0, len(rels))
	for _, r := range rels {
		if r.Kind == "dangling" {
			if opts.ShowDangling {
				dangling = append(dangling, r.Raw)
			}
			continue
		}
		if n := index.Notes[r.ID]; n != nil {
			resolved = append(resolved, n)
		}
	}
	if opts.Sort == "alpha" {
		sort.SliceStable(resolved, func(i, j int) bool { return strings.ToLower(noteLabel(resolved[i])) < strings.ToLower(noteLabel(resolved[j])) })
		sort.SliceStable(dangling, func(i, j int) bool { return strings.ToLower(dangling[i]) < strings.ToLower(dangling[j]) })
	}

	for _, child := range resolved {
		childNode := TreeJSONNode{ID: child.ID, Label: noteLabel(child), Path: child.Path, Dangling: false, NodeKind: "note", Children: []TreeJSONNode{}}
		if !visited[child.ID] {
			visited[child.ID] = true
			desc, err := s.buildLinksTreeJSONNode(child, index, opts, level+1, visited)
			delete(visited, child.ID)
			if err != nil {
				return TreeJSONNode{}, err
			}
			childNode.Children = desc.Children
		}
		node.Children = append(node.Children, childNode)
	}
	for _, raw := range dangling {
		node.Children = append(node.Children, TreeJSONNode{ID: "dangling:" + raw, Label: raw, Path: "", Dangling: true, NodeKind: "note", Children: []TreeJSONNode{}})
	}
	return node, nil
}

func (s QueryService) buildSemanticTreeJSONNode(note *domain.Note, index *domain.GraphIndex, opts QueryOptions, mode string, hops int, visited map[string]bool) (TreeJSONNode, error) {
	node := TreeJSONNode{ID: note.ID, Label: noteLabel(note), Path: note.Path, Dangling: false, NodeKind: "note", Children: []TreeJSONNode{}}
	if hops < 1 {
		return node, nil
	}

	terms := termsForMode(note, mode)
	if opts.Sort == "alpha" {
		sort.SliceStable(terms, func(i, j int) bool { return strings.ToLower(terms[i]) < strings.ToLower(terms[j]) })
	}
	for _, term := range terms {
		kind := "tag"
		prefix := "tag:"
		if mode == "categories" {
			kind = "category"
			prefix = "category:"
		}
		termNode := TreeJSONNode{ID: prefix + strings.ToLower(strings.TrimSpace(term)), Label: term, Path: "", Dangling: false, NodeKind: kind, Children: []TreeJSONNode{}}
		for _, child := range notesForTerm(index, mode, term) {
			childNode := TreeJSONNode{ID: child.ID, Label: noteLabel(child), Path: child.Path, Dangling: false, NodeKind: "note", Children: []TreeJSONNode{}}
			if !visited[child.ID] {
				visited[child.ID] = true
				desc, err := s.buildSemanticTreeJSONNode(child, index, opts, mode, hops-1, visited)
				delete(visited, child.ID)
				if err != nil {
					return TreeJSONNode{}, err
				}
				childNode.Children = desc.Children
			}
			termNode.Children = append(termNode.Children, childNode)
		}
		node.Children = append(node.Children, termNode)
	}
	return node, nil
}

func (s QueryService) renderLinksTreeChildren(b *strings.Builder, parent *domain.Note, index *domain.GraphIndex, opts QueryOptions, level int, prefix string, visited map[string]bool) error {
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
		children = append(children, ports.RenderedRelation{Kind: "resolved", Title: noteLabel(n), Path: n.Path, Size: st.Size(), ModUnix: st.ModTimeUnix(), Raw: r.ID})
	}

	if opts.Sort == "alpha" {
		sort.SliceStable(children, func(i, j int) bool { return strings.ToLower(children[i].Title) < strings.ToLower(children[j].Title) })
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
			b.WriteString(prefix + branch + line + " " + renderCycle(opts) + "\n")
			continue
		}
		b.WriteString(prefix + branch + line + "\n")
		if level < opts.Depth {
			visited[childID] = true
			if err := s.renderLinksTreeChildren(b, index.Notes[childID], index, opts, level+1, nextPrefix, visited); err != nil {
				return err
			}
			delete(visited, childID)
		}
	}
	return nil
}

func termsForMode(note *domain.Note, mode string) []string {
	if mode == "categories" {
		return append([]string{}, note.Categories...)
	}
	return append([]string{}, note.Tags...)
}

func notesForTerm(index *domain.GraphIndex, mode, term string) []*domain.Note {
	out := make([]*domain.Note, 0)
	needle := strings.ToLower(strings.TrimSpace(term))
	for _, id := range index.Ordered {
		n := index.Notes[id]
		if n == nil {
			continue
		}
		values := n.Tags
		if mode == "categories" {
			values = n.Categories
		}
		for _, v := range values {
			if strings.ToLower(strings.TrimSpace(v)) == needle {
				out = append(out, n)
				break
			}
		}
	}
	return out
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

func renderCycle(opts QueryOptions) string {
	marker := opts.CycleMarker
	if strings.TrimSpace(marker) == "" {
		marker = "↺"
	}
	if opts.ColorizeCycle != nil {
		return opts.ColorizeCycle(marker)
	}
	return marker
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
