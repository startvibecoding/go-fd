package glob

import "testing"

func TestToRegexMatch(t *testing.T) {
	cases := []struct {
		pattern string
		input   string
		match   bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.rs", false},
		{"*.go", "sub/main.go", false}, // literal separator
		{"**/*.go", "sub/main.go", true},
		{"**/*.go", "main.go", true},
		{"foo?", "fool", true},
		{"foo?", "foo", false},
		{"[abc].txt", "a.txt", true},
		{"[abc].txt", "d.txt", false},
		{"[!abc].txt", "d.txt", true},
		{"{a,b}.txt", "a.txt", true},
		{"{a,b}.txt", "c.txt", false},
		{"src/*.go", "src/x.go", true},
	}
	for _, c := range cases {
		re, err := Compile(c.pattern, Options{LiteralSeparator: true})
		if err != nil {
			t.Fatalf("compile %q: %v", c.pattern, err)
		}
		if got := re.MatchString(c.input); got != c.match {
			t.Errorf("glob %q vs %q = %v, want %v", c.pattern, c.input, got, c.match)
		}
	}
}

func TestCaseInsensitive(t *testing.T) {
	re, err := Compile("*.GO", Options{LiteralSeparator: true, CaseInsensitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("main.go") {
		t.Error("expected case-insensitive match")
	}
}
