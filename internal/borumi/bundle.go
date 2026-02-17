package borumi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Bundle struct {
	BundlePath string `json:"bundle_path"`
	DBPath     string `json:"db_path"`
}

func ResolveBundle(path string) (Bundle, error) {
	if strings.TrimSpace(path) == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return Bundle{}, fmt.Errorf("could not resolve path %q: %w", path, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return Bundle{}, fmt.Errorf("could not stat path %q: %w", absPath, err)
	}

	if info.IsDir() {
		dbPath := filepath.Join(absPath, "project.bmproj")
		if _, err := os.Stat(dbPath); err == nil {
			return Bundle{
				BundlePath: absPath,
				DBPath:     dbPath,
			}, nil
		}
		return Bundle{}, fmt.Errorf("directory %q does not look like a Borumi bundle: missing project.bmproj", absPath)
	}

	if filepath.Base(absPath) == "project.bmproj" {
		return Bundle{
			BundlePath: filepath.Dir(absPath),
			DBPath:     absPath,
		}, nil
	}

	return Bundle{}, fmt.Errorf("path %q is not a Borumi bundle directory or project.bmproj file", absPath)
}
