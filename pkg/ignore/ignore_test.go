package ignore

import "testing"

func TestGitignoreMatching(t *testing.T) {
	g := NewFromLines("", []string{
		"node_modules",
		"*.log",
		"/build",
		"!important.log",
		"docs/",
	})
	cases := []struct {
		path  string
		isDir bool
		want  Match
	}{
		{"node_modules", true, Ignore},
		{"sub/node_modules", true, Ignore},
		{"app.log", false, Ignore},
		{"important.log", false, Whitelist},
		{"build", true, Ignore},
		{"sub/build", true, None}, // anchored to root
		{"docs", true, Ignore},
		{"docs", false, None}, // dir-only
		{"src/main.go", false, None},
	}
	for _, c := range cases {
		if got := g.Matches(c.path, c.isDir); got != c.want {
			t.Errorf("Matches(%q, dir=%v) = %v, want %v", c.path, c.isDir, got, c.want)
		}
	}
}

func TestComments(t *testing.T) {
	g := NewFromLines("", []string{"# comment", "", "  ", "foo"})
	if got := g.Matches("foo", false); got != Ignore {
		t.Errorf("expected foo ignored, got %v", got)
	}
	if got := g.Matches("comment", false); got != None {
		t.Errorf("comment should not be a pattern, got %v", got)
	}
}
