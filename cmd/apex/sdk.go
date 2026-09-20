package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// newXZReader shells out to xz; pure-Go xz is a dependency we avoid.
// ponytail: requires xz binary (Termux/standard linux both ship it); swap in
// github.com/ulikunitz/xz when vendoring becomes acceptable.
// pw is closed only after cmd.Wait(): closing early kills xz mid-stream and
// yields silent partial extraction.
func newXZReader(r io.Reader) (io.ReadCloser, error) {
	path, err := exec.LookPath("xz")
	if err != nil {
		return nil, fmt.Errorf("xz binary not found in PATH (required for .tar.xz extraction)")
	}
	cmd := exec.Command(path, "-d", "-T0")
	cmd.Stdin = r
	var out bytes.Buffer
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		cmd.Wait()
		pw.Close()
	}()
	return readCloserPipe{pr, cmd}, nil
}

type readCloserPipe struct {
	*io.PipeReader
	cmd *exec.Cmd
}

func (c readCloserPipe) Close() error {
	err := c.PipeReader.Close()
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	return err
}

// sdkRoot resolves ANDROID_HOME (fallback ANDROID_SDK_ROOT).
func sdkRoot() (string, error) {
	root := os.Getenv("ANDROID_HOME")
	if root == "" {
		root = os.Getenv("ANDROID_SDK_ROOT")
	}
	if root == "" {
		return "", fmt.Errorf("ANDROID_HOME not set (set it to your SDK root)")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(abs, "/"), nil
}

// pkgDir maps "platforms;android-34" -> $SDK/platforms/android-34.
func pkgDir(root, pkgPath string) (string, error) {
	parts := strings.SplitN(pkgPath, ";", 2)
	switch {
	case len(parts) == 2:
		return filepath.Join(root, parts[0], parts[1]), nil
	case pkgPath == "platform-tools":
		return filepath.Join(root, "platform-tools"), nil
	default:
		return "", fmt.Errorf("invalid package path %q (expected \"prefix;version\" or platform-tools)", pkgPath)
	}
}

// ensureTree creates the standard SDK skeleton directories.
func ensureTree(root string) error {
	for _, d := range []string{"platforms", "build-tools", "platform-tools", "ndk", "cmake", "licenses", ".apex"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return err
		}
	}
	return nil
}
