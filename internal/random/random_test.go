package random

import (
	"strings"
	"testing"
)

func TestLotteryValidation(t *testing.T) {
	got := Lottery(nil, nil, 1)
	if !strings.Contains(got, "抽選対象") {
		t.Fatalf("Lottery() = %q, want no candidates warning", got)
	}

	got = Lottery([]string{"a"}, []string{"1"}, 2)
	if !strings.Contains(got, "候補者数") {
		t.Fatalf("Lottery() = %q, want count warning", got)
	}
}

func TestLotteryDoesNotMutateCandidates(t *testing.T) {
	candidates := []string{"1", "2", "3"}
	_ = Lottery([]string{"a", "b", "c"}, candidates, 2)

	want := []string{"1", "2", "3"}
	for i := range want {
		if candidates[i] != want[i] {
			t.Fatalf("candidates mutated: got %#v, want %#v", candidates, want)
		}
	}
}
