package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 0}

// progressReader prints a realtime download counter on one line.
type progressReader struct {
	r     io.Reader
	total int64
	done  int64
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.done += int64(n)
	if p.done >= 1<<20 {
		if p.total > 0 {
			fmt.Fprintf(os.Stdout, "\r  %.1f/%.1f MB (%d%%)", float64(p.done)/1e6, float64(p.total)/1e6, p.done*100/p.total)
		} else {
			fmt.Fprintf(os.Stdout, "\r  %.1f MB", float64(p.done)/1e6)
		}
	}
	return n, err
}

func httpGet(url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// downloadToFile streams url to dest with sha256 verification, realtime
// progress and 2 retries. expected is the catalog-reported size (fallback
// when the server sends no Content-Length).
func downloadToFile(url, dest, wantSHA string, expected int64) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(3 * time.Second)
		}
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
			continue
		}
		f, err := os.Create(dest)
		if err != nil {
			resp.Body.Close()
			return err
		}
		h := sha256.New()
		total := resp.ContentLength
		if total <= 0 {
			total = expected
		}
		n, err := io.Copy(io.MultiWriter(f, h), &progressReader{r: resp.Body, total: total})
		fmt.Fprintln(os.Stdout)
		resp.Body.Close()
		f.Close()
		if err != nil {
			lastErr = err
			continue
		}
		got := hex.EncodeToString(h.Sum(nil))
		if wantSHA != "" && got != strings.ToLower(wantSHA) {
			lastErr = fmt.Errorf("checksum mismatch: got %s want %s", got[:12], wantSHA[:12])
			continue
		}
		_ = n
		return nil
	}
	os.Remove(dest)
	return lastErr
}

// extract unpacks .tar.xz / .tar.gz / .zip into dest, stripping first path
// component when stripTop is true, with zip-slip protection.
func extract(archive, dest string, stripTop bool) error {
	tmp := dest + ".tmp"
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	var err error
	switch {
	case strings.HasSuffix(archive, ".zip"):
		err = extractZip(archive, tmp, stripTop)
	case strings.HasSuffix(archive, ".tar.xz"):
		err = extractTar(archive, tmp, "xz", stripTop)
	case strings.HasSuffix(archive, ".tar.gz"), strings.HasSuffix(archive, ".tgz"):
		err = extractTar(archive, tmp, "gz", stripTop)
	case strings.HasSuffix(archive, ".tar"):
		err = extractTar(archive, tmp, "", stripTop)
	default:
		err = fmt.Errorf("unsupported archive type: %s", archive)
	}
	if err != nil {
		os.RemoveAll(tmp)
		return err
	}
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	// atomic-ish install
	return os.Rename(tmp, dest)
}

func sanitize(dest, name string) (string, error) {
	clean := filepath.Clean(name)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("unsafe archive entry: %s", name)
	}
	return filepath.Join(dest, clean), nil
}

func stripFirst(name string) (string, bool) {
	for strings.HasPrefix(name, "./") {
		name = name[2:]
	}
	if i := strings.IndexByte(name, '/'); i >= 0 {
		return name[i+1:], true
	}
	return "", false
}

func extractTar(archive, dest, comp string, strip bool) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	switch comp {
	case "xz":
		xr, err := newXZReader(f)
		if err != nil {
			return err
		}
		defer xr.Close()
		r = xr
	case "gz":
		gr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gr.Close()
		r = gr
	}
	tr := tar.NewReader(r)
	var count int
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			if count > 0 {
				fmt.Println()
			}
			return nil
		}
		if err != nil {
			return err
		}
		name := hdr.Name
		if strip {
			var ok bool
			name, ok = stripFirst(name)
			if !ok || name == "" {
				continue
			}
		}
		target, err := sanitize(dest, name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)&0o777|0o700); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)&0o777|0o600)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			count++
			fmt.Printf("\r  extracting [%d] %s", count, filepath.Base(target))
		}
	}
}

func extractZip(archive, dest string, strip bool) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	zipCount := 0
	for _, zf := range zr.File {
		name := zf.Name
		if strip {
			var ok bool
			name, ok = stripFirst(name)
			if !ok || name == "" {
				continue
			}
		}
		target, err := sanitize(dest, name)
		if err != nil {
			return err
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, zf.Mode()&0o777|0o600)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
		zipCount++
		fmt.Printf("\r  extracting [%d] %s", zipCount, filepath.Base(target))
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
