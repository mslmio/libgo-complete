// Package complete gives a Go program shell auto-completion of its own command
// line, with no completion script to write or maintain.
//
// The mechanism is the one bash exposes through `complete -C`: the shell runs
// the binary itself with COMP_LINE set to what the user has typed so far, and
// reads candidate completions from its standard output. So the program that
// knows its own commands is the one answering, and the shell integration is a
// single line in a startup file.
//
// Describe the command line as a tree and hand it to Command.Complete before
// parsing flags:
//
//	var completions = &complete.Command{
//	    Sub: map[string]*complete.Command{"lookup": completionsLookup},
//	    Flags: map[string]complete.Predictor{"--json": predict.Nothing},
//	}
//
//	func main() {
//	    completions.Complete("mytool")
//	    // ... normal argument handling ...
//	}
//
// Complete returns immediately when the environment is not a completion
// request, so the call costs one getenv on a normal run.
//
// Prior art: posener/complete, which established this approach in Go.
package complete

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/mslmio/libgo-complete/install"
)

// Predictor offers candidate completions for a partly-typed word.
type Predictor interface {
	// Predict returns candidates for prefix. Returning candidates that do not
	// share the prefix is allowed; they are filtered out before they reach the
	// shell, so a predictor may ignore the prefix entirely.
	Predict(prefix string) []string
}

// PredictFunc adapts a function to Predictor.
type PredictFunc func(prefix string) []string

// Predict implements Predictor.
func (f PredictFunc) Predict(prefix string) []string {
	if f == nil {
		return nil
	}
	return f(prefix)
}

// Command describes one level of a command line.
//
// A nil *Command is a valid leaf that predicts nothing, so a tree may name a
// subcommand it has nothing to say about yet.
type Command struct {
	// Sub are the subcommands available at this level, keyed by name.
	Sub map[string]*Command

	// Flags are the flags accepted at this level, keyed by the flag as typed
	// including its dashes. The value predicts that flag's VALUE; use
	// predict.Nothing for a boolean flag.
	Flags map[string]Predictor

	// Args predicts positional arguments at this level.
	Args Predictor
}

// Complete answers a shell completion request and exits.
//
// It does nothing and returns when COMP_LINE is unset, which is every ordinary
// run. name is the command as the user types it, used only by installation.
func (c *Command) Complete(name string) {
	line, ok := os.LookupEnv("COMP_LINE")
	if !ok {
		if os.Getenv("COMP_INSTALL") == "1" {
			install.Run(name, false, os.Getenv("COMP_YES") == "1", os.Stdout, os.Stdin)
			os.Exit(0)
		}
		if os.Getenv("COMP_UNINSTALL") == "1" {
			install.Run(name, true, os.Getenv("COMP_YES") == "1", os.Stdout, os.Stdin)
			os.Exit(0)
		}
		return
	}
	// COMP_POINT is where the cursor is. Completing from the cursor rather than
	// from the end of the line is what makes editing mid-line work.
	if p, err := strconv.Atoi(os.Getenv("COMP_POINT")); err == nil && p >= 0 && p < len(line) {
		line = line[:p]
	}
	c.write(os.Stdout, line)
	os.Exit(0)
}

// write resolves one completion request. Split out from Complete so it is
// testable without an exit.
func (c *Command) write(w io.Writer, line string) {
	for _, opt := range c.options(line) {
		fmt.Fprintln(w, opt)
	}
}

// options is the candidate list for a command line, filtered and sorted.
//
// Sorted because map iteration is random, and an unsorted list makes the
// shell's own column layout reshuffle between presses of the same key.
func (c *Command) options(line string) []string {
	args := Parse(line)
	// The first word is the program itself.
	if len(args) > 0 {
		args = args[1:]
	}

	// Split what is finished from the word under the cursor.
	var prefix string
	if n := len(args); n > 0 && !args[n-1].Completed {
		prefix = args[n-1].Text
		args = args[:n-1]
	}

	cmd, done := c.descend(args)
	if cmd == nil {
		return nil
	}

	// A flag immediately before the cursor is being given its value, unless it
	// is a flag this level does not know or one that takes none.
	if len(done) > 0 {
		if p, ok := cmd.Flags[done[len(done)-1]]; ok && p != nil {
			return filter(p.Predict(prefix), prefix)
		}
	}

	var out []string
	// A leading dash means flags, and only flags: offering subcommands there
	// buries the flags the user has already said they want.
	if strings.HasPrefix(prefix, "-") {
		for flag := range cmd.Flags {
			out = append(out, flag)
		}
		return filter(out, prefix)
	}
	for sub := range cmd.Sub {
		out = append(out, sub)
	}
	for flag := range cmd.Flags {
		out = append(out, flag)
	}
	if cmd.Args != nil {
		out = append(out, cmd.Args.Predict(prefix)...)
	}
	return filter(out, prefix)
}

// descend walks the finished words into the deepest matching subcommand,
// returning it and the words that were not consumed as subcommand names.
func (c *Command) descend(args []Arg) (*Command, []string) {
	cmd := c
	var rest []string
	for _, arg := range args {
		if sub, ok := cmd.Sub[arg.Text]; ok {
			if sub == nil {
				return nil, nil
			}
			cmd = sub
			// A subcommand starts a new level, so nothing before it can be the
			// flag whose value is now being typed.
			rest = nil
			continue
		}
		rest = append(rest, arg.Text)
	}
	return cmd, rest
}

// filter keeps only candidates matching the prefix, de-duplicated and sorted.
func filter(in []string, prefix string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !strings.HasPrefix(s, prefix) {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
