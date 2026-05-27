package app

import (
	"path/filepath"
	"regexp"
	"strings"

	"voyage/internal/domain"
	"voyage/internal/ports"
)

var fallbackWikiRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

type IndexerService struct {
	repo   ports.NoteRepository
	parser ports.NoteParser
	log    ports.Logger
}

func NewIndexer(repo ports.NoteRepository, parser ports.NoteParser, log ports.Logger) IndexerService {
	return IndexerService{repo: repo, parser: parser, log: log}
}

func (s IndexerService) Build(root string) (*domain.GraphIndex, error) {
	paths, err := s.repo.ListMarkdownFiles(root)
	if err != nil {
		return nil, err
	}
	idx := &domain.GraphIndex{Notes: map[string]*domain.Note{}}
	titleIndex := map[string][]string{}
	baseIndex := map[string][]string{}

	for _, p := range paths {
		content, err := s.repo.ReadFile(p)
		if err != nil {
			return nil, err
		}
		parsed, err := s.parser.Parse(content)
		if err != nil {
			s.log.Warn("frontmatter invalide: " + p)
			parsed = parseLinksOnly(content)
		}
		id := filepath.Clean(p)
		n := &domain.Note{ID: id, Title: parsed.Title, Path: p, RawLinks: parsed.Links, Tags: parsed.Tags}
		idx.Notes[id] = n
		idx.Ordered = append(idx.Ordered, id)
		if n.Title != "" {
			titleIndex[norm(n.Title)] = append(titleIndex[norm(n.Title)], id)
		}
		base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		baseIndex[norm(base)] = append(baseIndex[norm(base)], id)
	}

	for _, id := range idx.Ordered {
		n := idx.Notes[id]
		for _, raw := range n.RawLinks {
			resolvedID := resolve(raw, titleIndex, baseIndex)
			if resolvedID == "" {
				n.DanglingLinks = append(n.DanglingLinks, raw)
				continue
			}
			n.ResolvedLinks = append(n.ResolvedLinks, resolvedID)
			idx.Notes[resolvedID].Backlinks = append(idx.Notes[resolvedID].Backlinks, n.ID)
		}
	}
	return idx, nil
}

func parseLinksOnly(content []byte) ports.ParsedNote {
	p := ports.ParsedNote{}
	for _, m := range fallbackWikiRe.FindAllStringSubmatch(string(content), -1) {
		if len(m) > 1 {
			p.Links = append(p.Links, strings.TrimSpace(m[1]))
		}
	}
	return p
}

func resolve(raw string, titleIndex, baseIndex map[string][]string) string {
	key := norm(raw)
	if ids := titleIndex[key]; len(ids) == 1 {
		return ids[0]
	}
	if ids := baseIndex[key]; len(ids) == 1 {
		return ids[0]
	}
	return ""
}

func norm(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
