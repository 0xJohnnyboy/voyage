package output

import (
	"fmt"
	"strings"
	"time"

	"voyage/internal/ports"
)

type Colorizer func(string) string

type TextFormatterConfig struct {
	DanglingPrefix   string
	ColorizeDangling Colorizer
}

type TextFormatter struct {
	danglingPrefix   string
	colorizeDangling Colorizer
}

func NewTextFormatter(cfg TextFormatterConfig) TextFormatter {
	prefix := cfg.DanglingPrefix
	if strings.TrimSpace(prefix) == "" {
		prefix = "⚠"
	}
	colorize := cfg.ColorizeDangling
	if colorize == nil {
		colorize = func(s string) string { return s }
	}
	return TextFormatter{danglingPrefix: prefix, colorizeDangling: colorize}
}

func (f TextFormatter) FormatSimple(relations []ports.RenderedRelation) string {
	var b strings.Builder
	for _, r := range relations {
		if r.Kind == "dangling" {
			b.WriteString(f.renderDangling(r.Raw))
			b.WriteByte('\n')
			continue
		}
		b.WriteString(r.Title)
		b.WriteByte('\n')
	}
	return b.String()
}

func (f TextFormatter) FormatDetailed(relations []ports.RenderedRelation) string {
	var b strings.Builder
	for _, r := range relations {
		if r.Kind == "dangling" {
			b.WriteString(f.renderDangling(r.Raw))
			b.WriteByte('\n')
			continue
		}
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\n", humanSize(r.Size), time.Unix(r.ModUnix, 0).Format(time.RFC3339), r.Path, r.Title))
	}
	return b.String()
}

func (f TextFormatter) renderDangling(raw string) string {
	return f.colorizeDangling(f.danglingPrefix + " " + raw)
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
