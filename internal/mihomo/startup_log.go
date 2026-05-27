package mihomo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const geoDBMinBytes = 64 << 10 // ignore tiny/partial files

var geoDBFilenames = []string{"geoip.metadb", "Country.mmdb"}

// SubprocessLogPath returns the mihomo child log path for a config directory.
func SubprocessLogPath(cfgDir string) string {
	return filepath.Join(cfgDir, "mihomo-subprocess.log")
}

// LogFileSize returns current log size or 0 if missing.
func LogFileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

// GeoDatabaseCached reports whether a geo database file is already on disk.
func GeoDatabaseCached(cfgDir string) bool {
	for _, name := range geoDBFilenames {
		st, err := os.Stat(filepath.Join(cfgDir, name))
		if err == nil && st.Size() >= geoDBMinBytes {
			return true
		}
	}
	return false
}

// StartupActivity maps a mihomo subprocess log line to user-facing status text.
// The second return value is true when geo download has finished successfully.
func StartupActivity(line string) (text string, geoReady bool) {
	msg := extractLogMsg(line)
	if msg == "" {
		return "", false
	}
	switch {
	case strings.Contains(msg, "Load MMDB file"):
		return "Geo-база готова", true
	case strings.Contains(msg, "Can't find MMDB, start download"):
		return "Скачивание geo-базы (первый запуск)…", false
	case strings.Contains(msg, "can't download MMDB"), strings.Contains(msg, "can't initial GeoIP"):
		short := msg
		if len(short) > 72 {
			short = short[:69] + "…"
		}
		return "Ошибка geo-базы: " + short, false
	case strings.Contains(msg, "Start initial configuration in progress"):
		return "Инициализация mihomo…", false
	case strings.Contains(msg, "Initial configuration complete"):
		return "Конфигурация mihomo готова…", false
	default:
		return "", false
	}
}

func extractLogMsg(line string) string {
	const key = `msg="`
	i := strings.Index(line, key)
	if i < 0 {
		return ""
	}
	rest := line[i+len(key):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func tailRead(path string, offset int64) ([]string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, offset, nil
		}
		return nil, offset, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, offset, err
	}
	size := st.Size()
	if size < offset {
		offset = 0
	}
	if size <= offset {
		return nil, offset, nil
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, offset, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, offset, err
	}
	newOffset := offset + int64(len(data))
	if len(data) == 0 {
		return nil, newOffset, nil
	}
	text := string(data)
	lines := strings.Split(text, "\n")
	trimmed := lines[:0]
	for _, ln := range lines {
		if ln != "" {
			trimmed = append(trimmed, ln)
		}
	}
	lines = trimmed
	if len(lines) > 0 && lines[len(lines)-1] != "" && !strings.HasSuffix(text, "\n") {
		// last fragment is incomplete; keep offset before it
		partial := lines[len(lines)-1]
		newOffset = offset + int64(len(text)-len(partial))
		lines = lines[:len(lines)-1]
	}
	return lines, newOffset, nil
}

// RunGeoStartupWatcher tails mihomo logs for geo progress without blocking callers.
// Geo errors are not reported via onStatus (avoid sticky UI); use logs instead.
func RunGeoStartupWatcher(ctx context.Context, cfgDir, logPath string, logOffset int64, onStatus func(string)) {
	go func() {
		_ = WaitGeoReady(ctx, cfgDir, logPath, logOffset, func(text string) {
			if text == "" || strings.HasPrefix(text, "Ошибка geo-базы") {
				return
			}
			if onStatus != nil {
				onStatus(text)
			}
		}, 120*time.Second)
	}()
}

// WaitGeoReady blocks until mihomo finishes loading geo data or timeout elapses.
// onStatus receives human-readable progress for UI activity text.
func WaitGeoReady(ctx context.Context, cfgDir, logPath string, logOffset int64, onStatus func(string), timeout time.Duration) error {
	if GeoDatabaseCached(cfgDir) {
		return nil
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	deadline := time.Now().Add(timeout)
	downloadStart := time.Time{}
	offset := logOffset
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if GeoDatabaseCached(cfgDir) {
				if onStatus != nil {
					onStatus("Geo-база готова")
				}
				return nil
			}
			lines, newOffset, err := tailRead(logPath, offset)
			if err == nil {
				offset = newOffset
				for _, line := range lines {
					text, geoReady := StartupActivity(line)
					if text != "" && onStatus != nil {
						onStatus(text)
					}
					if geoReady {
						return nil
					}
					msg := extractLogMsg(line)
					if strings.Contains(msg, "Can't find MMDB, start download") {
						downloadStart = time.Now()
					}
					if strings.Contains(msg, "can't download MMDB") || strings.Contains(msg, "can't initial GeoIP") {
						return fmt.Errorf("%s", msg)
					}
				}
			}
			if !downloadStart.IsZero() && onStatus != nil {
				sec := int(time.Since(downloadStart).Seconds())
				if sec > 0 {
					onStatus(fmt.Sprintf("Скачивание geo-базы… %d с", sec))
				}
			}
			if time.Now().After(deadline) {
				return nil
			}
		}
	}
}
