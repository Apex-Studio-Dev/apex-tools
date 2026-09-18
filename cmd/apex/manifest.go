package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ManifestEntry struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	SHA256      string `json:"sha256"`
	InstalledAt string `json:"installed_at"`
}

type Manifest struct {
	Entries []ManifestEntry `json:"packages"`
}

func manifestPath(root string) string { return filepath.Join(root, ".apex", "manifest.json") }

func loadManifest(root string) (*Manifest, error) {
	data, err := os.ReadFile(manifestPath(root))
	if os.IsNotExist(err) {
		return &Manifest{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse local manifest: %w", err)
	}
	return &m, nil
}

func (m *Manifest) find(pkgPath string) *ManifestEntry {
	for i := range m.Entries {
		if m.Entries[i].Path == pkgPath {
			return &m.Entries[i]
		}
	}
	return nil
}

func (m *Manifest) add(e ManifestEntry) {
	if cur := m.find(e.Path); cur != nil {
		*cur = e
		return
	}
	m.Entries = append(m.Entries, e)
}

func (m *Manifest) remove(pkgPath string) bool {
	for i := range m.Entries {
		if m.Entries[i].Path == pkgPath {
			m.Entries = append(m.Entries[:i], m.Entries[i+1:]...)
			return true
		}
	}
	return false
}

func (m *Manifest) save(root string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	p := manifestPath(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p+".tmp", append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(p+".tmp", p)
}

func newEntry(pkg *PkgInfo) ManifestEntry {
	return ManifestEntry{
		Path:        pkg.Path,
		Version:     pkg.Version,
		SHA256:      pkg.SHA256,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
	}
}
