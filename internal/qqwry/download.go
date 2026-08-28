package qqwry

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Sources are the download URLs tried in order until one succeeds. The
// first is the canonical mirror (updated weekly); the second is served
// from a China-friendly CDN.
var Sources = []string{
	"https://github.com/metowolf/qqwry.dat/releases/latest/download/qqwry.dat",
	"https://cdn.1008.site/gh/nmgliangwei/qqwry@main/qqwry.dat",
	"https://github.com/nmgliangwei/qqwry/releases/latest/download/qqwry.dat",
}

// DefaultPath returns the default database location under the platform
// cache directory:
//
//	macOS:   ~/Library/Caches/ling-box/qqwry.dat
//	Linux:   ~/.cache/ling-box/qqwry.dat (or $XDG_CACHE_HOME/ling-box)
//	Windows: %LocalAppData%\ling-box\qqwry.dat
//
// If the platform cache directory is unavailable, the config directory
// (e.g. ~/.config, ~/Library/Application Support, %AppData%) is used,
// falling back to a .ling-box directory in the user's home.
func DefaultPath() (string, error) {
	dir, err := defaultDir(os.UserCacheDir, os.UserConfigDir, os.UserHomeDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ling-box", "qqwry.dat"), nil
}

// defaultDir picks the directory for cached databases, trying the platform
// cache directory, then the config directory, then the user's home.
func defaultDir(cacheDir, configDir, homeDir func() (string, error)) (string, error) {
	for _, dir := range []func() (string, error){cacheDir, configDir} {
		if d, err := dir(); err == nil {
			return d, nil
		}
	}
	return homeDir()
}

// Ensure makes sure a database exists at path, downloading it from Sources
// if it is missing or force is true. The download is atomic: the file is
// written to a temporary name, validated, then renamed into place. If
// progress is non-nil, a notice is reported before and after the download;
// with interactive set, percentage steps are reported in between.
func Ensure(path string, force bool, client *http.Client, progress io.Writer, interactive bool) (bool, error) {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return false, nil
		}
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create cache directory: %w", err)
	}
	errs := make([]error, 0, len(Sources))
	for _, url := range Sources {
		if err := download(client, url, path, progress, interactive); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", url, err))
			continue
		}
		return true, nil
	}
	return false, fmt.Errorf("download qqwry.dat failed: %w", errors.Join(errs...))
}

func download(client *http.Client, url, path string, progress io.Writer, interactive bool) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ling-box")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %s", resp.Status)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".qqwry-*.dat")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()
	if progress != nil {
		fmt.Fprintf(progress, "downloading qqwry.dat from %s\n", url)
	}
	w := io.Writer(tmp)
	if progress != nil && interactive {
		w = io.MultiWriter(tmp, &progressWriter{out: progress, total: resp.ContentLength})
	}
	written, err := io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if progress != nil {
		if interactive {
			fmt.Fprintln(progress)
		}
		fmt.Fprintf(progress, "downloaded qqwry.dat (%s) to %s\n", humanSize(written), path)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// validate before the file replaces an existing database
	if _, err := Open(tmpName); err != nil {
		return fmt.Errorf("invalid database: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return nil
}

// humanSize renders a byte count in human-readable units (e.g. "26.4 MiB").
func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// progressWriter reports download progress to out in 10% steps.
type progressWriter struct {
	out      io.Writer
	total    int64
	done     int64
	reported int64
}

func (w *progressWriter) Write(p []byte) (int, error) {
	w.done += int64(len(p))
	if w.total > 0 {
		pct := w.done * 100 / w.total
		if pct >= w.reported+10 {
			w.reported = pct - pct%10
			fmt.Fprintf(w.out, "...%d%%", w.reported)
		}
	}
	return len(p), nil
}
