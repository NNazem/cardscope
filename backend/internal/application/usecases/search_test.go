package usecases

import "testing"

func TestClassify(t *testing.T) {
	for _, test := range []struct {
		score float64
		want  string
	}{{0.9, "confirmed"}, {0.6, "possible"}, {0.2, ""}} {
		if got := classify(test.score, 0.8, 0.45); got != test.want {
			t.Fatalf("classify(%f) = %q, want %q", test.score, got, test.want)
		}
	}
}
