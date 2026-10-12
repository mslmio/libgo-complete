package complete_test

import (
	"strings"
	"testing"

	complete "github.com/mslmio/libgo-complete"
	"github.com/mslmio/libgo-complete/predict"
)

// Built from the predictors a caller uses rather than bare nils: predict.Nothing
// holds a nil PredictFunc, which is not a nil Predictor.
func TestFlagPredictors(t *testing.T) {
	tree := &complete.Command{
		Sub: map[string]*complete.Command{
			"lookup": {
				Flags: map[string]complete.Predictor{
					"--json":  predict.Nothing,
					"--key":   predict.Something,
					"--field": predict.Set([]string{"ip", "is_vpn"}),
				},
			},
		},
		Flags: map[string]complete.Predictor{"--help": predict.Nothing},
	}
	cases := []struct {
		line string
		want string
	}{
		// A boolean flag takes no value, so the level's options come back.
		{"tool --help ", "--help lookup"},
		{"tool lookup --json ", "--field --json --key"},
		{"tool lookup --json --", "--field --json --key"},
		// A flag taking a value predicts it, or nothing when nothing can.
		{"tool lookup --field ", "ip is_vpn"},
		{"tool lookup --field i", "ip is_vpn"},
		{"tool lookup --key ", ""},
	}
	for _, c := range cases {
		got := strings.Join(complete.Options(tree, c.line), " ")
		if got != c.want {
			t.Errorf("options(%q) =\n  %q\nwant\n  %q", c.line, got, c.want)
		}
	}
}
