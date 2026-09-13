package complete

import "strings"

// Arg is one word of a command line as the shell would split it.
type Arg struct {
	// Text is the word with quoting and escapes resolved.
	Text string
	// Completed is false for the word the cursor sits in, which is the one
	// being completed rather than one already given.
	Completed bool
}

// Parse splits a command line the way a POSIX shell would, honouring single
// quotes, double quotes and backslash escapes.
//
// The final word is Completed only when the line ends in unquoted whitespace.
// That distinction is the whole point: "cmd data" is completing the word
// "data", while "cmd data " is starting a fresh one, and the shell tells the
// difference only through that trailing space.
func Parse(line string) []Arg {
	var (
		args    []Arg
		cur     strings.Builder
		have    bool // a word is open, even if empty so far ("" is a word)
		quote   byte // 0, '\'' or '"'
		escaped bool
	)
	flush := func() {
		if have {
			args = append(args, Arg{Text: cur.String(), Completed: true})
			cur.Reset()
			have = false
		}
	}
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case escaped:
			cur.WriteByte(ch)
			have = true
			escaped = false
		case ch == '\\' && quote != '\'':
			// Inside single quotes a backslash is literal; everywhere else it
			// escapes the next byte.
			escaped = true
			have = true
		case quote != 0:
			if ch == quote {
				quote = 0
			} else {
				cur.WriteByte(ch)
			}
			have = true
		case ch == '\'' || ch == '"':
			quote = ch
			have = true
		case ch == ' ' || ch == '\t':
			flush()
		default:
			cur.WriteByte(ch)
			have = true
		}
	}
	// Whatever is left is the word under the cursor. An unterminated quote or a
	// trailing backslash still leaves a word being typed.
	//
	// A line ending in whitespace has an EMPTY word under the cursor, and that
	// empty word has to be emitted: it is what distinguishes "complete the word
	// `sub`" from "`sub` is settled, offer me what comes after it".
	if have || escaped || quote != 0 {
		args = append(args, Arg{Text: cur.String(), Completed: false})
	} else if line != "" {
		args = append(args, Arg{Completed: false})
	}
	return args
}
