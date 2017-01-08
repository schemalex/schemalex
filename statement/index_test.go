package statement

import "testing"

func TestIndex(t *testing.T) {
	// Check that index fulfills Index
	var stmt Index = &index{}
	_ = stmt
}
