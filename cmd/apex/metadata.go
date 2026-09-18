package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// pkgRev converts "33.0.1" -> (33, 0, 1). Handles "37.2" -> (37, 2, 0) too.
func pkgRev(version string) (major, minor, micro int) {
	parts := strings.SplitN(version, ".", 3)
	vals := [3]int{}
	for i, p := range parts {
		fmt.Sscanf(p, "%d", &vals[i])
	}
	return vals[0], vals[1], vals[2]
}

type xmlLicenseRef struct {
	XMLName xml.Name `xml:"uses-license"`
	Ref     string   `xml:"ref,attr"`
}

// writePackageXML emits an sdkmanager-recognizable localPackage descriptor.
// Mirrors the schema used by android-ndk-custom's package-generator.c.
func writePackageXML(dir, pkgPath, version, displayName string) error {
	major, minor, micro := pkgRev(version)
	type revision struct {
		Major int `xml:"major"`
		Minor int `xml:"minor"`
		Micro int `xml:"micro"`
	}
	type pkg struct {
		XMLName     xml.Name       `xml:"localPackage"`
		Path        string         `xml:"path,attr"`
		Obsolete    bool           `xml:"obsolete,attr"`
		Revision    revision       `xml:"revision"`
		DisplayName string         `xml:"display-name"`
		UsesLicense xmlLicenseRef  `xml:"uses-license"`
	}
	data, err := xml.MarshalIndent(pkg{
		Path:     pkgPath,
		Obsolete: false,
		Revision: revision{major, minor, micro},
		DisplayName: displayName,
		UsesLicense: xmlLicenseRef{Ref: apexLicenseID},
	}, "", "    ")
	if err != nil {
		return err
	}
	out := []byte(xml.Header)
	out = append(out, data...)
	out = append(out, '\n')
	return os.WriteFile(filepath.Join(dir, "package.xml"), out, 0o644)
}

// writeSourceProperties ensures a package dir has source.properties.
// Existing files (from upstream artifacts) are preserved.
func writeSourceProperties(dir string, props string) error {
	p := filepath.Join(dir, "source.properties")
	if _, err := os.Stat(p); err == nil {
		return nil // keep upstream
	}
	return os.WriteFile(p, []byte(props), 0o644)
}

func sourceProps(kind, version, hostTag string) string {
	var b strings.Builder
	b.WriteString("Pkg.Desc=Android SDK " + strings.Title(kind) + "\n")
	b.WriteString("Pkg.Revision=" + version + "\n")
	b.WriteString("Pkg.License=" + apexLicenseID + "\n")
	switch kind {
	case "platform":
		b.WriteString("AndroidVersion.ApiLevel=" + strings.TrimPrefix(version, "android-") + "\n")
	case "build-tools":
		b.WriteString("Pkg.BuildTools.Version=" + version + "\n")
	case "cmake":
		b.WriteString("Pkg.CMAKE.Version=" + version + "\n")
	}
	_ = hostTag
	return b.String()
}
