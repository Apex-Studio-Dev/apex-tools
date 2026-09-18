package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Catalog mirrors scripts/gen-catalog.sh output (schema 1).
type Catalog struct {
	Schema    int       `json:"schema"`
	Generated string    `json:"generated"`
	MinCLI    string    `json:"min_cli"`
	Packages  []PkgInfo `json:"packages"`
}

type PkgInfo struct {
	Path    string   `json:"path"` // e.g. "platforms;android-34"
	Version string   `json:"version"`
	Hosts   []string `json:"hosts"`
	URL     string   `json:"url"`
	SHA256  string   `json:"sha256"`
	Size    int64    `json:"size"`
	License string   `json:"license"`
}

const defaultCatalogURL = "https://raw.githubusercontent.com/Apex-Studio-Dev/apex-tools/catalog/catalog.json"

func catalogURL() string {
	if u := os.Getenv("APEX_CATALOG_URL"); u != "" {
		return u
	}
	return defaultCatalogURL
}

func loadCatalog() (*Catalog, error) {
	data, err := httpGet(catalogURL())
	if err != nil {
		return nil, fmt.Errorf("fetch catalog: %w", err)
	}
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	if c.Schema > 1 {
		return nil, errors.New("catalog schema newer than CLI; update apex")
	}
	return &c, nil
}

// hostTag maps GOOS/GOARCH to the host tag used in catalog entries.
func hostTag() string {
	arch := runtime.GOARCH
	switch arch {
	case "arm":
		arch = "arm"
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "arm64"
	}
	switch runtime.GOOS {
	case "linux":
		return "linux-" + arch
	case "android":
		return "linux-" + arch // bionic binaries also serve as linux-arm64/arm
	case "darwin":
		return "darwin-" + arch
	case "windows":
		return "windows-x64"
	case "freebsd":
		return "freebsd-" + arch
	default:
		return runtime.GOOS + "-" + arch
	}
}

// find locates a catalog entry by package path for the current host.
func (c *Catalog) find(pkgPath string) (*PkgInfo, error) {
	for i := range c.Packages {
		p := &c.Packages[i]
		if p.Path == pkgPath && strings.Contains(p.URL, hostTag()) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("package %q not found in catalog for host %s (run: apex --list)", pkgPath, hostTag())
}
