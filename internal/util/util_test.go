package util

import "testing"

func TestSinglequote(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{
			input: `hoge's comment`,
			want:  `'hoge\'s comment'`,
		},
		{
			input: `'\`,
			want:  `'\'\\'`,
		},
	}
	for _, tt := range tests {
		got := Singlequote(tt.input)
		if got != tt.want {
			t.Errorf("want %q; got %q", tt.want, got)
		}
	}
}
