package fs

import (
	"os"
	"path/filepath"
	"strings"

	"voyage/internal/ports"
)

type LocalRepo struct{}

type localInfo struct {
	name    string
	size    int64
	modUnix int64
}

func (l localInfo) Name() string       { return l.name }
func (l localInfo) Size() int64        { return l.size }
func (l localInfo) ModTimeUnix() int64 { return l.modUnix }

func (LocalRepo) ListMarkdownFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".md") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func (LocalRepo) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (LocalRepo) Stat(path string) (ports.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return localInfo{name: info.Name(), size: info.Size(), modUnix: info.ModTime().Unix()}, nil
}
