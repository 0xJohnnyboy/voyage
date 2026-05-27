package app

import (
	"testing"

	"voyage/internal/adapters/logging"
	"voyage/internal/adapters/parser"
	"voyage/internal/domain"
	"voyage/internal/ports"
)

type fakeRepo struct {
	files map[string]string
}

func (f fakeRepo) ListMarkdownFiles(root string) ([]string, error) {
	out := make([]string, 0, len(f.files))
	for k := range f.files {
		out = append(out, k)
	}
	return out, nil
}
func (f fakeRepo) ReadFile(path string) ([]byte, error)     { return []byte(f.files[path]), nil }
func (f fakeRepo) Stat(path string) (ports.FileInfo, error) { return fakeInfo{}, nil }

type fakeInfo struct{}

func (fakeInfo) Name() string       { return "x" }
func (fakeInfo) Size() int64        { return 10 }
func (fakeInfo) ModTimeUnix() int64 { return 100 }

func TestIndexerResolveTitleThenBasenameAndDangling(t *testing.T) {
	repo := fakeRepo{files: map[string]string{
		"/n/a.md": "---\ntitle: A\n---\n[[B title]] [[c]] [[missing]]",
		"/n/b.md": "---\ntitle: B title\n---\n",
		"/n/c.md": "",
	}}
	idx, err := NewIndexer(repo, parser.MarkdownParser{}, logging.New("silent")).Build("/n")
	if err != nil {
		t.Fatal(err)
	}
	a := idx.Notes["/n/a.md"]
	if len(a.ResolvedLinks) != 2 || len(a.DanglingLinks) != 1 {
		t.Fatalf("unexpected links: %+v", a)
	}
}

func TestQuerySortAlpha(t *testing.T) {
	idx := &domain.GraphIndex{Notes: map[string]*domain.Note{
		"/n/a.md": {ID: "/n/a.md", ResolvedLinks: []string{"/n/z.md", "/n/b.md"}},
		"/n/z.md": {ID: "/n/z.md", Title: "Zulu", Path: "/n/z.md"},
		"/n/b.md": {ID: "/n/b.md", Title: "Alpha", Path: "/n/b.md"},
	}}
	q := NewQuery(fakeRepo{}, fakeStrategy{}, fakeFormatter{})
	out, err := q.Render(idx, "/n/a.md", QueryOptions{Sort: "alpha", ShowDangling: true})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Alpha,Zulu" {
		t.Fatalf("got %q", out)
	}
}

type fakeStrategy struct{}

func (fakeStrategy) Related(note *domain.Note, index *domain.GraphIndex) []ports.Relation {
	var r []ports.Relation
	for _, id := range note.ResolvedLinks {
		r = append(r, ports.Relation{Kind: "resolved", ID: id})
	}
	return r
}

type fakeFormatter struct{}

func (fakeFormatter) FormatSimple(rel []ports.RenderedRelation) string {
	if len(rel) == 2 {
		return rel[0].Title + "," + rel[1].Title
	}
	return ""
}
func (fakeFormatter) FormatDetailed(rel []ports.RenderedRelation) string { return "" }
