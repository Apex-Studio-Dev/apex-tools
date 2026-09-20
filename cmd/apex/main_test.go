package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGreater(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"34.0.0", "33.0.2", true},
		{"37.2", "37.1.9", true},
		{"android-34", "android-33", true},
		{"33.0.0", "33.0.0", false},
		{"3.18.1", "4.1.2", false},
	}
	for _, c := range cases {
		if got := greater(c.a, c.b); got != c.want {
			t.Errorf("greater(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestManifestRoundTrip(t *testing.T) {
	root := t.TempDir()
	m, err := loadManifest(root)
	if err != nil || len(m.Entries) != 0 {
		t.Fatalf("empty manifest: %v %v", m, err)
	}
	m.add(ManifestEntry{Path: "platforms;android-34", Version: "34"})
	if err := m.save(root); err != nil {
		t.Fatal(err)
	}
	m2, err := loadManifest(root)
	if err != nil || len(m2.Entries) != 1 || m2.Entries[0].Path != "platforms;android-34" {
		t.Fatalf("roundtrip: %v %v", m2, err)
	}
	m2.remove("platforms;android-34")
	m2.save(root)
	m3, _ := loadManifest(root)
	if len(m3.Entries) != 0 {
		t.Fatal("remove failed")
	}
}

func TestExtractZipSlipGuard(t *testing.T) {
	dir := t.TempDir()
	if _, err := sanitize(dir, "../../evil"); err == nil {
		t.Fatal("zip-slip not blocked")
	}
	if _, err := sanitize(dir, "platforms/android-34/file"); err != nil {
		t.Fatalf("normal path rejected: %v", err)
	}
}

func TestPkgDir(t *testing.T) {
	d, err := pkgDir("/sdk", "platforms;android-34")
	if err != nil || d != filepath.Join("/sdk", "platforms", "android-34") {
		t.Fatalf("pkgDir: %v %v", d, err)
	}
	d, err = pkgDir("/sdk", "platform-tools")
	if err != nil || d != "/sdk/platform-tools" {
		t.Fatalf("pkgDir platform-tools: %v %v", d, err)
	}
	if _, err := pkgDir("/sdk", "bogus"); err == nil {
		t.Fatal("bogus path accepted")
	}
	_ = os.Environ
}
