package parser

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"voyage/internal/ports"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

type MarkdownParser struct{}

type frontmatter struct {
	Title      string    `yaml:"title"`
	Tags       yaml.Node `yaml:"tags"`
	Categories yaml.Node `yaml:"categories"`
}

func (MarkdownParser) Parse(content []byte) (ports.ParsedNote, error) {
	text := string(content)
	parsed := ports.ParsedNote{}

	body := text
	if strings.HasPrefix(text, "---\n") {
		rest := strings.TrimPrefix(text, "---\n")
		parts := strings.SplitN(rest, "\n---\n", 2)
		if len(parts) == 2 {
			var fm frontmatter
			if err := yaml.Unmarshal([]byte(parts[0]), &fm); err != nil {
				return parsed, err
			}
			parsed.Title = strings.TrimSpace(fm.Title)
			parsed.Tags = parseStringOrList(fm.Tags)
			parsed.Categories = parseStringOrList(fm.Categories)
			body = parts[1]
		}
	}

	matches := wikilinkRe.FindAllStringSubmatch(body, -1)
	for _, m := range matches {
		if len(m) > 1 {
			parsed.Links = append(parsed.Links, strings.TrimSpace(m[1]))
		}
	}

	return parsed, nil
}

func parseStringOrList(node yaml.Node) []string {
	out := []string{}
	switch node.Kind {
	case 0:
		return out
	case yaml.ScalarNode:
		v := strings.TrimSpace(node.Value)
		if v != "" {
			out = append(out, v)
		}
	case yaml.SequenceNode:
		for _, c := range node.Content {
			if c.Kind != yaml.ScalarNode {
				continue
			}
			v := strings.TrimSpace(c.Value)
			if v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}
