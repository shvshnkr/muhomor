package mihomo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DefaultGeoMMDBURL is the mihomo default geoip.metadb mirror (MetaCubeX meta-rules-dat).
var DefaultGeoMMDBURL = "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb"

const geoFilename = "geoip.metadb"

// GeoDatabasePath returns the primary MMDB path under mihomo config dir (-d).
func GeoDatabasePath(cfgDir string) string {
	return filepath.Join(cfgDir, geoFilename)
}

// DownloadGeoDatabase fetches geoip.metadb into cfgDir (atomic replace).
func DownloadGeoDatabase(ctx context.Context, cfgDir string) error {
	if cfgDir == "" {
		return fmt.Errorf("geodata: empty config dir")
	}
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return err
	}
	dest := GeoDatabasePath(cfgDir)
	tmp := dest + ".download"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DefaultGeoMMDBURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("geodata: HTTP %d", resp.StatusCode)
	}
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, 64<<20))
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n < geoDBMinBytes {
		_ = os.Remove(tmp)
		return fmt.Errorf("geodata: file too small (%d bytes)", n)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// UpdateGeoDatabaseIfStale downloads when missing or older than maxAge. maxAge <= 0 always refreshes.
func UpdateGeoDatabaseIfStale(ctx context.Context, cfgDir string, maxAge time.Duration) error {
	path := GeoDatabasePath(cfgDir)
	if st, err := os.Stat(path); err == nil && st.Size() >= geoDBMinBytes {
		if maxAge <= 0 || time.Since(st.ModTime()) < maxAge {
			return nil
		}
	}
	return DownloadGeoDatabase(ctx, cfgDir)
}

// GeoActivityForUI returns true if activity text is geo download progress (not a hard failure).
func GeoActivityForUI(text string) bool {
	return text == "Geo-база готова" ||
		text == "Скачивание geo-базы (первый запуск)…" ||
		(len(text) >= len("Скачивание geo-базы…") && text[:len("Скачивание geo-базы…")] == "Скачивание geo-базы…")
}
