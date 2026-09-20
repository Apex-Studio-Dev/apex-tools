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

// writePackageXML emits an sdkmanager-recognizable localPackage descriptor
// mirroring Google's repository.xml layout: namespaced repository root,
// embedded license text, genericDetailsType and uses-license ref.
func writePackageXML(dir, pkgPath, version, displayName string) error {
	major, minor, micro := pkgRev(version)

	namespaces := []string{
		`xmlns:ns2="http://schemas.android.com/repository/android/common/02"`,
		`xmlns:ns3="http://schemas.android.com/repository/android/common/01"`,
		`xmlns:ns4="http://schemas.android.com/repository/android/generic/01"`,
		`xmlns:ns5="http://schemas.android.com/repository/android/generic/02"`,
		`xmlns:ns6="http://schemas.android.com/sdk/android/repo/addon2/01"`,
		`xmlns:ns7="http://schemas.android.com/sdk/android/repo/addon2/02"`,
		`xmlns:ns8="http://schemas.android.com/sdk/android/repo/addon2/03"`,
		`xmlns:ns9="http://schemas.android.com/sdk/android/repo/repository2/01"`,
		`xmlns:ns10="http://schemas.android.com/sdk/android/repo/repository2/02"`,
		`xmlns:ns11="http://schemas.android.com/sdk/android/repo/repository2/03"`,
		`xmlns:ns12="http://schemas.android.com/sdk/android/repo/sys-img2/04"`,
		`xmlns:ns13="http://schemas.android.com/sdk/android/repo/sys-img2/03"`,
		`xmlns:ns14="http://schemas.android.com/sdk/android/repo/sys-img2/02"`,
		`xmlns:ns15="http://schemas.android.com/sdk/android/repo/sys-img2/01"`,
	}

	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString("<ns2:repository ")
	b.WriteString(strings.Join(namespaces, " "))
	b.WriteString(">\n")
	b.WriteString(`<license id="`)
	b.WriteString(apexLicenseID)
	b.WriteString(`" type="text">`)
	if err := xml.EscapeText(&b, []byte(apexLicenseText)); err != nil {
		return err
	}
	b.WriteString("</license>\n")
	b.WriteString(fmt.Sprintf(`<localPackage path="%s" obsolete="false">`+"\n", pkgPath))
	b.WriteString(`    <type-details xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="ns5:genericDetailsType"/>` + "\n")
	b.WriteString(fmt.Sprintf("    <revision><major>%d</major><minor>%d</minor><micro>%d</micro></revision>\n", major, minor, micro))
	b.WriteString("    <display-name>")
	if err := xml.EscapeText(&b, []byte(displayName)); err != nil {
		return err
	}
	b.WriteString("</display-name>\n")
	b.WriteString(fmt.Sprintf(`    <uses-license ref="%s"/>`+"\n", apexLicenseID))
	b.WriteString("</localPackage>\n")
	b.WriteString("</ns2:repository>\n")
	return os.WriteFile(filepath.Join(dir, "package.xml"), []byte(b.String()), 0o644)
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
