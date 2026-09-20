package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const apexLicenseID = "apex-sdk-license"

// AGP hardcoded accepted license hashes (android-sdk-license).
// ponytail: keep in sync with AGP SdkConstants; extend when AGP adds hashes.
var agpLicenseHashes = []string{
	"8933bad161af0178038a4d5b4e07233cd3795a7b",
	"d56f5187479451eabf01fb78af6dfcb131a6481e",
	"24333f8a63b6825ea9c5514f83c2829b004d1fee",
	"84831b9409646a918e30573bab4c9c91346d8abd",
}

// The apex license text; shipped verbatim into licenses/apex-sdk-license.
const apexLicenseText = `APEX TOOLS SDK LICENSE

This software package is part of apex-tools, a custom Android SDK
distribution maintained by Apex-Studio-Dev. Components are derived from the
Android Open Source Project and repackaged for aarch64/arm hosts.

By accepting, you agree to the terms in the LICENSE file shipped with
apex-tools and acknowledge that the underlying components remain subject
to their upstream licenses (Apache-2.0 for AOSP-derived code).

Copyright (c) 2026 Apex-Studio-Dev
`

func licensesAccepted(root string) bool {
	_, err := os.Stat(filepath.Join(root, "licenses", apexLicenseID))
	return err == nil
}

func writeLicenses(root string) error {
	dir := filepath.Join(root, "licenses")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, apexLicenseID), []byte(apexLicenseText), 0o644); err != nil {
		return err
	}
	// AGP compatibility file: known accepted hashes, one per line.
	agpPath := filepath.Join(dir, "android-sdk-license")
	if existing, err := os.ReadFile(agpPath); err == nil {
		// merge: keep lines, append missing hashes
		have := map[string]bool{}
		for _, l := range strings.Split(string(existing), "\n") {
			have[strings.TrimSpace(l)] = true
		}
		var b strings.Builder
		b.Write(existing)
		for _, h := range agpLicenseHashes {
			if !have[h] {
				b.WriteString(h + "\n")
			}
		}
		return os.WriteFile(agpPath, []byte(b.String()), 0o644)
	}
	var b strings.Builder
	for _, h := range agpLicenseHashes {
		b.WriteString(h + "\n")
	}
	return os.WriteFile(agpPath, []byte(b.String()), 0o644)
}

func promptAccept() bool {
	fmt.Print("Accept? [y/N] ")
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}
