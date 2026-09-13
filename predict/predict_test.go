package predict

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetAndNothing(t *testing.T) {
	if got := Nothing.Predict(""); got != nil {
		t.Errorf("Nothing predicted %v", got)
	}
	got := Set([]string{"a", "b"}).Predict("")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("Set = %v", got)
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.mmdb", "b.mmdb", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := Files("*.mmdb").Predict(dir + string(filepath.Separator))
	joined := strings.Join(got, " ")
	for _, want := range []string{"a.mmdb", "b.mmdb"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	if strings.Contains(joined, "c.txt") {
		t.Errorf("pattern not applied: %v", got)
	}
	// A directory is always offered, whatever the pattern: the path into it has
	// to be typed before anything inside can match.
	if !strings.Contains(joined, "sub") {
		t.Errorf("directory not offered: %v", got)
	}

	// Dirs offers only directories.
	dirs := strings.Join(Dirs("*").Predict(dir+string(filepath.Separator)), " ")
	if strings.Contains(dirs, ".mmdb") || !strings.Contains(dirs, "sub") {
		t.Errorf("Dirs = %v", dirs)
	}
}

// A prefix naming a partial entry completes against its parent directory.
func TestFilesPartialPrefix(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "country.mmdb"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	got := Files("*.mmdb").Predict(filepath.Join(dir, "coun"))
	if len(got) != 1 || !strings.HasSuffix(got[0], "country.mmdb") {
		t.Errorf("got %v", got)
	}
}

func TestFilesMissingDir(t *testing.T) {
	if got := Files("*").Predict("/nonexistent-dir-xyz/"); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}
