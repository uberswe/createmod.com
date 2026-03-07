package pages

import "testing"

func Test_Trivial(t *testing.T) {
	if 1+1 != 2 {
		t.Fatalf("basic arithmetic failed")
	}
}
