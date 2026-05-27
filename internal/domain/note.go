package domain

type Note struct {
	ID            string
	Title         string
	Path          string
	RawLinks      []string
	ResolvedLinks []string
	Backlinks     []string
	DanglingLinks []string
	Tags          []string
}

type GraphIndex struct {
	Notes   map[string]*Note
	Ordered []string
}
