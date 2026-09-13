package complete

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		line string
		want []Arg
	}{
		{"", nil},
		{"cmd", []Arg{{"cmd", false}}},
		// The trailing space is the only thing telling a finished word from one
		// still being typed, so it has to produce an extra, empty argument.
		{"cmd ", []Arg{{"cmd", true}, {"", false}}},
		{"cmd sub", []Arg{{"cmd", true}, {"sub", false}}},
		{"cmd sub ", []Arg{{"cmd", true}, {"sub", true}, {"", false}}},
		{"cmd   sub", []Arg{{"cmd", true}, {"sub", false}}},
		{"cmd\tsub", []Arg{{"cmd", true}, {"sub", false}}},

		// Quoting.
		{`cmd "two words"`, []Arg{{"cmd", true}, {"two words", false}}},
		{`cmd 'two words'`, []Arg{{"cmd", true}, {"two words", false}}},
		{`cmd "two words" x`, []Arg{{"cmd", true}, {"two words", true}, {"x", false}}},
		{`cmd a"b"c`, []Arg{{"cmd", true}, {"abc", false}}},
		// An unterminated quote is a word still being typed.
		{`cmd "half`, []Arg{{"cmd", true}, {"half", false}}},
		// An empty quoted string is a real, empty argument.
		{`cmd "" `, []Arg{{"cmd", true}, {"", true}, {"", false}}},

		// Escapes.
		{`cmd a\ b`, []Arg{{"cmd", true}, {"a b", false}}},
		{`cmd a\"b`, []Arg{{"cmd", true}, {`a"b`, false}}},
		// Inside single quotes a backslash is a literal backslash.
		{`cmd 'a\b'`, []Arg{{"cmd", true}, {`a\b`, false}}},
		{`cmd \`, []Arg{{"cmd", true}, {"", false}}},

		// Flags.
		{"cmd --json", []Arg{{"cmd", true}, {"--json", false}}},
		{"cmd -f a,b", []Arg{{"cmd", true}, {"-f", true}, {"a,b", false}}},
	}
	for _, c := range cases {
		got := Parse(c.line)
		if len(got) != len(c.want) {
			t.Errorf("Parse(%q) = %v, want %v", c.line, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("Parse(%q)[%d] = %+v, want %+v", c.line, i, got[i], c.want[i])
			}
		}
	}
}
