// Package install wires a binary's self-completion into the user's shell.
//
// For bash and zsh that is one `complete -C` line in the startup file, which
// points the shell at the binary itself. Fish has no equivalent, so it gets a
// small function in its completions directory that forwards the command line
// through COMP_LINE.
package install

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// binPath is the absolute path of the running binary, which is what the shell
// must call back into. Resolved once; an empty value means we could not tell,
// and every installer refuses rather than writing a line that does nothing.
var binPath = resolveBinPath()

// Run performs an install or uninstall, prompting first unless yes is set.
func Run(name string, uninstall, yes bool, out io.Writer, in io.Reader) {
	action := "install"
	if uninstall {
		action = "uninstall"
	}
	if !yes {
		fmt.Fprintf(out, "%s completion for %s? [y/N] ", action, name)
		answer, _ := bufio.NewReader(in).ReadString('\n')
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "y", "yes":
		default:
			fmt.Fprintln(out, "cancelled")
			return
		}
	}

	var err error
	if uninstall {
		err = Uninstall(name)
	} else {
		err = Install(name)
	}
	if err != nil {
		fmt.Fprintf(out, "%s failed: %s\n", action, err)
		os.Exit(1)
	}
	fmt.Fprintf(out, "%sed; start a new shell to pick it up\n", action)
}

// Install adds completion for cmd to every shell found on this machine.
func Install(cmd string) error {
	return each(cmd, func(s shell) error { return s.install(cmd) })
}

// Uninstall removes it again.
func Uninstall(cmd string) error {
	return each(cmd, func(s shell) error { return s.uninstall(cmd) })
}

// IsInstalled reports whether any shell already has it.
func IsInstalled(cmd string) bool {
	for _, s := range shells() {
		if s.installed(cmd) {
			return true
		}
	}
	return false
}

// BashCmd is the line bash needs in ~/.bashrc.
func BashCmd(cmd string) (string, error) {
	if binPath == "" {
		return "", errors.New("complete: cannot determine this binary's path")
	}
	// -o default keeps bash's own filename completion as a fallback, so a path
	// argument still completes where we predict nothing.
	return fmt.Sprintf("complete -C %s -o default %s", binPath, cmd), nil
}

// ZshCmd is the lines zsh needs in ~/.zshrc.
func ZshCmd(cmd string) (string, error) {
	if binPath == "" {
		return "", errors.New("complete: cannot determine this binary's path")
	}
	// zsh has no native `complete -C`; bashcompinit provides it.
	return fmt.Sprintf(
		"autoload -U +X bashcompinit && bashcompinit\ncomplete -C %s -o default %s",
		binPath, cmd,
	), nil
}

// FishCmd is the completion file fish needs.
func FishCmd(cmd string) (string, error) {
	if binPath == "" {
		return "", errors.New("complete: cannot determine this binary's path")
	}
	// `commandline -cp` is the line up to the cursor. When the cursor sits on
	// whitespace there is no current token, and the trailing space has to be
	// restored or the binary cannot tell "finished word" from "word being
	// typed".
	return fmt.Sprintf(`function __complete_%[1]s
    set -lx COMP_LINE (commandline -cp)
    test -z (commandline -ct)
    and set COMP_LINE "$COMP_LINE "
    %[2]s
end
complete -f -c %[1]s -a "(__complete_%[1]s)"`, cmd, binPath), nil
}

// each applies fn to every shell present, collecting failures rather than
// stopping: bash succeeding is worth having even if fish does not.
func each(cmd string, fn func(shell) error) error {
	found := shells()
	if len(found) == 0 {
		return errors.New("complete: found no supported shell to install into")
	}
	var errs []error
	for _, s := range found {
		if err := fn(s); err != nil {
			errs = append(errs, err)
		}
	}
	// Every shell refusing for the same reason is a real failure; one of three
	// refusing is usually "already installed" and not worth failing over.
	if len(errs) == len(found) {
		return errors.Join(errs...)
	}
	return nil
}

// shell is one shell's integration.
type shell interface {
	installed(cmd string) bool
	install(cmd string) error
	uninstall(cmd string) error
}

// shells is the integrations that apply on this machine.
func shells() []shell {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var out []shell
	// A startup file is created if missing only when its shell is actually
	// installed, so a machine without zsh does not grow a ~/.zshrc.
	if rc := rcFile(home, ".bashrc", "bash"); rc != "" {
		out = append(out, &rcShell{path: rc, body: BashCmd})
	}
	if rc := rcFile(home, ".zshrc", "zsh"); rc != "" {
		out = append(out, &rcShell{path: rc, body: ZshCmd})
	}
	if dir := fishConfigDir(home); dir != "" {
		out = append(out, &fishShell{dir: dir})
	}
	return out
}

// rcFile is the startup file to edit, or "" when that shell is not here.
func rcFile(home, name, bin string) string {
	path := filepath.Join(home, name)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if _, err := exec.LookPath(bin); err == nil {
		return path
	}
	return ""
}

// fishConfigDir honours XDG, which is where fish actually looks.
func fishConfigDir(home string) string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "fish")
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	if _, err := exec.LookPath("fish"); err == nil {
		return dir
	}
	return ""
}

// rcShell is a shell configured by a line in a startup file.
type rcShell struct {
	path string
	body func(cmd string) (string, error)
}

func (s *rcShell) installed(cmd string) bool {
	want, err := s.body(cmd)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return false
	}
	return containsBlock(string(data), want)
}

func (s *rcShell) install(cmd string) error {
	want, err := s.body(cmd)
	if err != nil {
		return err
	}
	if s.installed(cmd) {
		return fmt.Errorf("already installed in %s", s.path)
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	// A leading newline so appending to a file with no trailing one cannot
	// splice our line onto the end of theirs.
	if _, err := fmt.Fprintf(f, "\n%s\n", want); err != nil {
		return err
	}
	return nil
}

func (s *rcShell) uninstall(cmd string) error {
	want, err := s.body(cmd)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if !containsBlock(string(data), want) {
		return fmt.Errorf("not installed in %s", s.path)
	}
	drop := make(map[string]bool, 2)
	for _, line := range strings.Split(want, "\n") {
		drop[line] = true
	}
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if drop[line] {
			continue
		}
		kept = append(kept, line)
	}
	// Written through a neighbouring file and renamed, so an interrupted
	// uninstall cannot leave someone with a truncated startup file.
	tmp := s.path + ".complete.tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// containsBlock reports whether every line of block appears in body.
func containsBlock(body, block string) bool {
	lines := make(map[string]bool)
	for _, line := range strings.Split(body, "\n") {
		lines[line] = true
	}
	for _, line := range strings.Split(block, "\n") {
		if !lines[line] {
			return false
		}
	}
	return true
}

// fishShell is configured by a file in the completions directory.
type fishShell struct{ dir string }

func (s *fishShell) file(cmd string) string {
	return filepath.Join(s.dir, "completions", cmd+".fish")
}

func (s *fishShell) installed(cmd string) bool {
	_, err := os.Stat(s.file(cmd))
	return err == nil
}

func (s *fishShell) install(cmd string) error {
	if s.installed(cmd) {
		return fmt.Errorf("already installed at %s", s.file(cmd))
	}
	body, err := FishCmd(cmd)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.file(cmd)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.file(cmd), []byte(body+"\n"), 0o644)
}

func (s *fishShell) uninstall(cmd string) error {
	if !s.installed(cmd) {
		return fmt.Errorf("not installed in %s", s.dir)
	}
	return os.Remove(s.file(cmd))
}

// resolveBinPath finds the absolute path of the running binary.
func resolveBinPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	// Resolved through symlinks so a link that later moves does not leave the
	// shell calling a path that no longer exists.
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return ""
	}
	return abs
}
