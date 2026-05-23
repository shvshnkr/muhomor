package mihomo

import (
	"os"
	"testing"
)

func TestStartupActivity(t *testing.T) {
	cases := []struct {
		line     string
		wantText string
		geoReady bool
	}{
		{
			line:     `time="2026-05-23T05:26:50.547190300+03:00" level=info msg="Can't find MMDB, start download"`,
			wantText: "Скачивание geo-базы (первый запуск)…",
		},
		{
			line:     `time="2026-05-23T05:27:08.873500100+03:00" level=info msg="Load MMDB file: C:\\data\\geoip.metadb"`,
			wantText: "Geo-база готова",
			geoReady: true,
		},
		{
			line:     `time="2026-05-23T05:26:50.542598600+03:00" level=info msg="Start initial configuration in progress"`,
			wantText: "Инициализация mihomo…",
		},
		{
			line:     `time="2026-05-23T05:26:51.790357700+03:00" level=info msg="Initial configuration complete, total time: 1247ms"`,
			wantText: "Конфигурация mihomo готова…",
		},
		{
			line:     `time="2026-05-23T05:26:52.000000000+03:00" level=error msg="can't initial GeoIP: can't download MMDB: context deadline exceeded"`,
			wantText: "Ошибка geo-базы: can't initial GeoIP: can't download MMDB: context deadline exceeded",
		},
	}
	for _, tc := range cases {
		text, ready := StartupActivity(tc.line)
		if text != tc.wantText {
			t.Errorf("StartupActivity(%q) text=%q want %q", tc.line, text, tc.wantText)
		}
		if ready != tc.geoReady {
			t.Errorf("StartupActivity(%q) geoReady=%v want %v", tc.line, ready, tc.geoReady)
		}
	}
}

func TestExtractLogMsg(t *testing.T) {
	line := `time="2026-05-23T05:26:51.793548900+03:00" level=info msg="Mixed(http+socks) proxy listening at: 127.0.0.1:2181"`
	if got := extractLogMsg(line); got != `Mixed(http+socks) proxy listening at: 127.0.0.1:2181` {
		t.Fatalf("extractLogMsg()=%q", got)
	}
}

func TestGeoDatabaseCached(t *testing.T) {
	dir := t.TempDir()
	if GeoDatabaseCached(dir) {
		t.Fatal("expected false for empty dir")
	}
	path := dir + "/geoip.metadb"
	if err := os.WriteFile(path, make([]byte, geoDBMinBytes), 0o600); err != nil {
		t.Fatal(err)
	}
	if !GeoDatabaseCached(dir) {
		t.Fatal("expected true after writing geo file")
	}
}

func TestTailReadPartialLine(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/mihomo-subprocess.log"
	if err := os.WriteFile(path, []byte(`line1`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lines, off, err := tailRead(path, 0)
	if err != nil || len(lines) != 1 || off != 6 {
		t.Fatalf("tailRead() lines=%v off=%d err=%v", lines, off, err)
	}
}
