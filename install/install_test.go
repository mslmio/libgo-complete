package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellCmds(t *testing.T) {
	binPath = "/usr/local/bin/mytool"

	bash, err := BashCmd("mytool")
	if err != nil {
		t.Fatal(err)
	}
	if bash != "complete -C /usr/local/bin/mytool -o default mytool" {
		t.Errorf("bash: %q", bash)
	}

	zsh, err := ZshCmd("mytool")
	if err != nil {
		t.Fatal(err)
	}
	// bashcompinit must come first, or `complete` is not a command yet.
	if !strings.HasPrefix(zsh, "autoload -U +X bashcompinit && bashcompinit\n") {
		t.Errorf("zsh: %q", zsh)
	}
	if !strings.Contains(zsh, "complete -C /usr/local/bin/mytool -o default mytool") {
		t.Errorf("zsh: %q", zsh)
	}

	fish, err := FishCmd("mytool")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"function __complete_mytool",
		"set -lx COMP_LINE (commandline -cp)",
		"/usr/local/bin/mytool",
		`complete -f -c mytool -a "(__complete_mytool)"`,
	} {
		if !strings.Contains(fish, want) {
			t.Errorf("fish missing %q in:\n%s", want, fish)
		}
	}
}

// Without a resolvable binary path there is nothing useful to write, so every
// generator must refuse rather than emit a line pointing at nothing.
func TestShellCmdsNeedBinPath(t *testing.T) {
	saved := binPath
	t.Cleanup(func() { binPath = saved })
	binPath = ""

	for name, fn := range map[string]func(string) (string, error){
		"bash": BashCmd, "zsh": ZshCmd, "fish": FishCmd,
	} {
		if _, err := fn("mytool"); err == nil {
			t.Errorf("%s should refuse without a binary path", name)
		}
	}
}

func TestRcShellRoundTrip(t *testing.T) {
	binPath = "/usr/local/bin/mytool"
	dir := t.TempDir()
	rc := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rc, []byte("export PATH=$PATH:/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &rcShell{path: rc, body: BashCmd}

	if s.installed("mytool") {
		t.Fatal("reported installed before installing")
	}
	if err := s.install("mytool"); err != nil {
		t.Fatal(err)
	}
	if !s.installed("mytool") {
		t.Fatal("not reported installed after installing")
	}
	// Installing twice must not duplicate the line.
	if err := s.install("mytool"); err == nil {
		t.Error("second install should report it is already there")
	}

	body, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(body), "complete -C") != 1 {
		t.Errorf("expected one completion line, got:\n%s", body)
	}
	// The user's own content survives.
	if !strings.Contains(string(body), "export PATH=$PATH:/x") {
		t.Error("clobbered existing rc content")
	}

	if err := s.uninstall("mytool"); err != nil {
		t.Fatal(err)
	}
	if s.installed("mytool") {
		t.Fatal("still installed after uninstall")
	}
	body, err = os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "complete -C") {
		t.Errorf("line survived uninstall:\n%s", body)
	}
	if !strings.Contains(string(body), "export PATH=$PATH:/x") {
		t.Error("uninstall removed the user's own content")
	}
}

// Appending to a file with no trailing newline must not splice our line onto
// the end of theirs.
func TestRcShellNoTrailingNewline(t *testing.T) {
	binPath = "/usr/local/bin/mytool"
	dir := t.TempDir()
	rc := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rc, []byte("alias ll='ls -l'"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &rcShell{path: rc, body: BashCmd}
	if err := s.install("mytool"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "ls -l'complete") {
		t.Errorf("spliced onto the existing line:\n%s", body)
	}
	if !s.installed("mytool") {
		t.Error("not detected after appending")
	}
}

func TestFishShellRoundTrip(t *testing.T) {
	binPath = "/usr/local/bin/mytool"
	dir := t.TempDir()
	s := &fishShell{dir: dir}

	if s.installed("mytool") {
		t.Fatal("reported installed before installing")
	}
	if err := s.install("mytool"); err != nil {
		t.Fatal(err)
	}
	if !s.installed("mytool") {
		t.Fatal("not reported installed after installing")
	}
	if _, err := os.Stat(filepath.Join(dir, "completions", "mytool.fish")); err != nil {
		t.Errorf("completion file not where fish looks: %v", err)
	}
	if err := s.uninstall("mytool"); err != nil {
		t.Fatal(err)
	}
	if s.installed("mytool") {
		t.Error("still installed after uninstall")
	}
}
