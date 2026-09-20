package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	all := fs.Bool("all", false, "show packages for all hosts")
	asJSON := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, _ := sdkRoot()
	var m *Manifest
	if root != "" {
		m, _ = loadManifest(root)
	}
	c, err := loadCatalog()
	if err != nil {
		return err
	}
	tag := hostTag()
	var rows []PkgInfo
	for _, p := range c.Packages {
		if *all || len(p.Hosts) == 0 || contains(p.Hosts, tag) || strings.Contains(p.URL, tag) {
			rows = append(rows, p)
		}
	}
	if *asJSON {
		type row struct {
			PkgInfo
			Installed bool `json:"installed"`
		}
		out := make([]row, 0, len(rows))
		for _, p := range rows {
			inst := m != nil && m.find(p.Path) != nil && dirExists(mustDir(root, p.Path))
			out = append(out, row{p, inst})
		}
		return json.NewEncoder(os.Stdout).Encode(out)
	}
	fmt.Printf("%-28s %-14s %-10s %s\n", "PATH", "VERSION", "STATUS", "SIZE")
	for _, p := range rows {
		status := "available"
		if m != nil && m.find(p.Path) != nil {
			status = "installed"
		}
		fmt.Printf("%-28s %-14s %-10s %d\n", p.Path, p.Version, status, p.Size)
	}
	return nil
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func mustDir(root, pkgPath string) string {
	d, err := pkgDir(root, pkgPath)
	if err != nil {
		return ""
	}
	return d
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func cmdInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	force := fs.Bool("force", false, "reinstall even if same version installed")
	skipLicense := fs.Bool("licenses-accepted", false, "assume licenses already accepted (CI)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pkgs := fs.Args()
	if len(pkgs) == 0 {
		return fmt.Errorf("usage: apex install <pkg>...")
	}
	root, err := sdkRoot()
	if err != nil {
		return err
	}
	if err := ensureTree(root); err != nil {
		return err
	}
	m, err := loadManifest(root)
	if err != nil {
		return err
	}
	c, err := loadCatalog()
	if err != nil {
		return err
	}
	if !*skipLicense && !licensesAccepted(root) {
		return fmt.Errorf("license not accepted; run: apex licenses")
	}
	for _, pkgPath := range pkgs {
		pkg, err := c.find(pkgPath)
		if err != nil {
			return err
		}
		if cur := m.find(pkgPath); cur != nil && cur.Version == pkg.Version && dirExists(mustDir(root, pkgPath)) && !*force {
			fmt.Printf("skip %s (already installed %s; use --force)\n", pkgPath, cur.Version)
			continue
		}
		dir, err := pkgDir(root, pkgPath)
		if err != nil {
			return err
		}
		tmp := filepath.Join(os.TempDir(), "apex-dl-"+filepath.Base(pkg.URL))
		fmt.Printf("downloading %s (%d bytes)...\n", pkg.URL, pkg.Size)
		if err := downloadToFile(pkg.URL, tmp, pkg.SHA256); err != nil {
			return err
		}
		fmt.Printf("extracting to %s...\n", dir)
		if err := extract(tmp, dir, true); err != nil {
			return err
		}
		os.Remove(tmp)

		// metadata: source.properties + package.xml (keep upstream when present)
		kind := strings.SplitN(pkgPath, ";", 2)[0]
		if err := writeSourceProperties(dir, sourceProps(kind, pkg.Version, hostTag())); err != nil {
			return err
		}
		display := displayNameFor(kind, pkg.Version)
		if err := writePackageXML(dir, pkgPath, pkg.Version, display); err != nil {
			return err
		}
		m.add(newEntry(pkg))
		if err := m.save(root); err != nil {
			return err
		}
		fmt.Printf("installed %s\n", pkgPath)
	}
	return nil
}

func displayNameFor(kind, version string) string {
	v := strings.TrimPrefix(version, "android-")
	switch kind {
	case "platforms":
		return "Android SDK Platform " + androidVersion(v)
	case "build-tools":
		return "Android SDK Build-Tools " + strings.SplitN(v, ".", 2)[0]
	case "platform-tools":
		return "Android SDK Platform-Tools"
	case "cmake":
		return "CMake " + v
	case "ndk":
		return "NDK (Side by side) " + v
	default:
		return kind + " " + v
	}
}

// androidVersion maps an API-level string (e.g. "37.2", "34") to the Android
// OS version used by Google in Pkg.Desc / Platform.Version / display-name.
func androidVersion(api string) string {
	major := strings.SplitN(api, ".", 2)[0]
	switch major {
	case "30":
		return "11"
	case "31":
		return "12"
	case "32":
		return "12"
	case "33":
		return "13"
	case "34":
		return "14"
	case "35":
		return "15"
	case "36":
		return "16"
	case "37":
		return "17"
	}
	return major
}

func cmdUninstall(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: apex uninstall <pkg>...")
	}
	root, err := sdkRoot()
	if err != nil {
		return err
	}
	m, err := loadManifest(root)
	if err != nil {
		return err
	}
	for _, pkgPath := range args {
		dir, err := pkgDir(root, pkgPath)
		if err != nil {
			return err
		}
		if _, err := os.Stat(dir); err != nil {
			fmt.Printf("not installed: %s\n", pkgPath)
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		m.remove(pkgPath)
		if err := m.save(root); err != nil {
			return err
		}
		fmt.Printf("uninstalled %s\n", pkgPath)
	}
	return nil
}

func cmdUpdate(args []string) error {
	root, err := sdkRoot()
	if err != nil {
		return err
	}
	m, err := loadManifest(root)
	if err != nil {
		return err
	}
	c, err := loadCatalog()
	if err != nil {
		return err
	}
	var toInstall []string
	for _, e := range m.Entries {
		var latest string
		for _, p := range c.Packages {
			if p.Path == e.Path && greater(p.Version, e.Version) {
				latest = p.Version
			}
		}
		if latest != "" {
			fmt.Printf("%s: %s -> %s\n", e.Path, e.Version, latest)
			toInstall = append(toInstall, e.Path)
		}
	}
	if len(toInstall) == 0 {
		fmt.Println("everything up to date")
		return nil
	}
	return cmdInstall(toInstall)
}

// greater is a naive dotted-version compare; good enough for x.y.z tokens.
func greater(a, b string) bool {
	as := splitVer(a)
	bs := splitVer(b)
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] != bs[i] {
			return as[i] > bs[i]
		}
	}
	return len(as) > len(bs)
}

func splitVer(s string) []int {
	s = strings.TrimPrefix(s, "android-")
	parts := strings.Split(s, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		fmt.Sscanf(p, "%d", &out[i])
	}
	return out
}

func cmdLicenses(args []string) error {
	fs := flag.NewFlagSet("licenses", flag.ContinueOnError)
	accept := fs.Bool("accept", false, "accept non-interactively")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := sdkRoot()
	if err != nil {
		return err
	}
	fmt.Print(apexLicenseText)
	if licensesAccepted(root) {
		fmt.Println("licenses already accepted at", filepath.Join(root, "licenses"))
		return nil
	}
	if *accept || promptAccept() {
		if err := writeLicenses(root); err != nil {
			return err
		}
		fmt.Println("licenses written to", filepath.Join(root, "licenses"))
		return nil
	}
	return fmt.Errorf("license not accepted")}
