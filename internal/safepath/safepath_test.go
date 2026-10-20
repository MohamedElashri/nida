package safepath

import "testing"

func TestEscapes(t *testing.T) {
	for rel, want := range map[string]bool{
		"..": true, "../x": true, "a/../..": true, "/abs": true,
		".": false, "a": false, "a/..": false, "..a": false, "a/../b": false,
	} {
		if got := Escapes(rel); got != want {
			t.Errorf("Escapes(%q) = %t, want %t", rel, got, want)
		}
	}
}
