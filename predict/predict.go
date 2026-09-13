// Package predict holds the completion predictors a command tree is built from.
package predict

import (
	"os"
	"path/filepath"
	"strings"

	complete "github.com/mslmio/libgo-complete"
)

// Nothing predicts no completions, which is what a boolean flag wants.
var Nothing complete.Predictor = complete.PredictFunc(nil)

// Something predicts nothing but marks a flag as taking a value, so the shell
// does not fall back to offering subcommands where an argument belongs.
var Something complete.Predictor = complete.PredictFunc(
	func(string) []string { return nil },
)

// Set predicts one of a fixed list.
func Set(options []string) complete.Predictor {
	return complete.PredictFunc(func(string) []string { return options })
}

// Func adapts a function to a Predictor.
func Func(f func(prefix string) []string) complete.Predictor {
	return complete.PredictFunc(f)
}

// Files predicts paths matching a glob pattern, as "*.mmdb". Use "*" for any
// file.
//
// Directories are always offered whatever the pattern, since a path into a
// directory has to be typed before the file inside it can match.
func Files(pattern string) complete.Predictor {
	return complete.PredictFunc(func(prefix string) []string {
		return paths(prefix, pattern, false)
	})
}

// Dirs predicts directories only.
func Dirs(pattern string) complete.Predictor {
	return complete.PredictFunc(func(prefix string) []string {
		return paths(prefix, pattern, true)
	})
}

// paths lists candidates in the directory the prefix names.
func paths(prefix, pattern string, dirsOnly bool) []string {
	dir := prefix
	// A prefix that is not itself a directory names a partial entry inside its
	// parent, so complete against the parent.
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		if st, err := os.Stat(prefix); err != nil || !st.IsDir() {
			dir = filepath.Dir(prefix)
		}
	}
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() {
			if dirsOnly {
				continue
			}
			if ok, err := filepath.Match(pattern, name); err != nil || !ok {
				continue
			}
		}
		p := filepath.Join(dir, name)
		// filepath.Join drops a leading "./", which would stop the candidate
		// matching a prefix the user typed that way.
		if strings.HasPrefix(prefix, "."+string(filepath.Separator)) {
			p = "." + string(filepath.Separator) + p
		}
		if e.IsDir() {
			p += string(filepath.Separator)
		}
		out = append(out, p)
	}
	return out
}
