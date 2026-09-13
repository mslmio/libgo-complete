package complete

import (
	"strings"
	"testing"
)

func set(opts ...string) Predictor {
	return PredictFunc(func(string) []string { return opts })
}

var tree = &Command{
	Sub: map[string]*Command{
		"lookup": {
			Flags: map[string]Predictor{
				"--json":  nil,
				"--field": set("ip", "is_vpn", "vpn.provider"),
			},
		},
		"database": {
			Sub: map[string]*Command{
				"list":     {},
				"download": {Flags: map[string]Predictor{"--format": set("csvgz", "mmdb")}},
			},
		},
		"logout": {},
	},
	Flags: map[string]Predictor{"--help": nil, "--version": nil},
}

func TestOptions(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		// Nothing typed yet: every subcommand and top-level flag.
		{"tool ", "--help --version database logout lookup"},
		{"tool l", "logout lookup"},
		{"tool lo", "logout lookup"},
		{"tool loo", "lookup"},
		// A leading dash means flags only; offering subcommands there buries
		// what the user asked for.
		{"tool -", "--help --version"},
		{"tool --h", "--help"},

		// Descending resets the level, so the parent's flags are gone.
		{"tool lookup ", "--field --json"},
		{"tool lookup --", "--field --json"},
		{"tool database ", "download list"},
		{"tool database d", "download"},
		{"tool database download ", "--format"},

		// A flag taking a value predicts that value, not the level's options.
		{"tool lookup --field ", "ip is_vpn vpn.provider"},
		{"tool lookup --field is", "is_vpn"},
		{"tool database download --format ", "csvgz mmdb"},
		// A boolean flag predicts nothing, so the level's options come back.
		{"tool lookup --json ", "--field --json"},

		// An unknown subcommand is not a level, so the top level still answers.
		{"tool nope ", "--help --version database logout lookup"},
		// A word already completed is not re-offered as a prefix.
		{"tool database list ", ""},
	}
	for _, c := range cases {
		got := strings.Join(tree.options(c.line), " ")
		if got != c.want {
			t.Errorf("options(%q) =\n  %q\nwant\n  %q", c.line, got, c.want)
		}
	}
}

// Candidates come out of maps, whose iteration order is deliberately random.
// Unsorted output makes the shell's column layout reshuffle between presses of
// the same key.
func TestOptionsStable(t *testing.T) {
	first := strings.Join(tree.options("tool "), " ")
	for i := 0; i < 50; i++ {
		if got := strings.Join(tree.options("tool "), " "); got != first {
			t.Fatalf("unstable: %q then %q", first, got)
		}
	}
}

func TestOptionsDeduplicates(t *testing.T) {
	c := &Command{
		Sub:   map[string]*Command{"go": {}},
		Args:  set("go", "stop"),
		Flags: map[string]Predictor{},
	}
	got := strings.Join(c.options("tool "), " ")
	if got != "go stop" {
		t.Errorf("got %q, want %q", got, "go stop")
	}
}

// A nil subcommand is a declared leaf we have nothing to say about; it must not
// panic.
func TestOptionsNilSub(t *testing.T) {
	c := &Command{Sub: map[string]*Command{"x": nil}}
	if got := c.options("tool x "); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := strings.Join(c.options("tool "), " "); got != "x" {
		t.Errorf("got %q, want %q", got, "x")
	}
}

func TestWriteLines(t *testing.T) {
	var b strings.Builder
	tree.write(&b, "tool l")
	if b.String() != "logout\nlookup\n" {
		t.Errorf("got %q", b.String())
	}
}
