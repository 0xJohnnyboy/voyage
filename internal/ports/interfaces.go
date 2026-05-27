package ports

import "voyage/internal/domain"

type NoteFile struct {
	Path    string
	Content []byte
}

type NoteRepository interface {
	ListMarkdownFiles(root string) ([]string, error)
	ReadFile(path string) ([]byte, error)
	Stat(path string) (FileInfo, error)
}

type FileInfo interface {
	Name() string
	Size() int64
	ModTimeUnix() int64
}

type ParsedNote struct {
	Title string
	Tags  []string
	Links []string
}

type NoteParser interface {
	Parse(content []byte) (ParsedNote, error)
}

type Logger interface {
	Debug(msg string)
	Warn(msg string)
}

type RelationStrategy interface {
	Related(note *domain.Note, index *domain.GraphIndex) []Relation
}

type Relation struct {
	Kind string
	ID   string
	Raw  string
}

type OutputFormatter interface {
	FormatSimple(relations []RenderedRelation) string
	FormatDetailed(relations []RenderedRelation) string
}

type RenderedRelation struct {
	Kind    string
	Title   string
	Path    string
	Size    int64
	ModUnix int64
	Raw     string
}
