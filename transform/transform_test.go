package transform

import (
	"testing"

	"golang.org/x/text/transform"
)

func TestNormalize(t *testing.T) {
	testCases := []struct {
		in   string
		want string
	}{
		{"hello, world\r\n", "hello, world\n"},
		{"hello, world\r", "hello, world\n"},
		{"hello, world\n", "hello, world\n"},
		{"", ""},
		{"\r\n", "\n"},
		{"hello,\r\nworld", "hello,\nworld"},
		{"hello,\rworld", "hello,\nworld"},
		{"hello,\nworld", "hello,\nworld"},
		{"hello,\n\rworld", "hello,\n\nworld"},
		{"hello,\r\n\r\nworld", "hello,\n\nworld"},

		{"hello,  world", "hello, world"},
		{"hello,    world", "hello, world"},
		{"hello,  \tworld", "hello, world"},
		{"hello,\t\t\tworld", "hello, world"},
		{"\t\thello,\t\t\tworld  ", " hello, world "},
		{"hello,\v\t\vworld", "hello, world"},
	}

	n := &normalize{}
	for _, c := range testCases {
		got, _, err := transform.String(n, c.in)
		if err != nil {
			t.Errorf("transform error %q: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("transform %q: got %q, want %q", c.in, got, c.want)
		}
	}
}
