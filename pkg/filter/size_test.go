package filter

import "testing"

func TestParseSize(t *testing.T) {
	cases := []struct {
		in   string
		kind SizeKind
		lim  uint64
	}{
		{"+1b", SizeMin, 1},
		{"-1k", SizeMax, 1000},
		{"+1ki", SizeMin, 1024},
		{"+1m", SizeMin, 1000000},
		{"+1mi", SizeMin, 1048576},
		{"100k", SizeEquals, 100000},
	}
	for _, c := range cases {
		f, err := ParseSize(c.in)
		if err != nil {
			t.Fatalf("ParseSize(%q): %v", c.in, err)
		}
		if f.Kind != c.kind || f.Limit != c.lim {
			t.Errorf("ParseSize(%q) = %+v, want kind=%v lim=%d", c.in, f, c.kind, c.lim)
		}
	}
}

func TestParseSizeInvalid(t *testing.T) {
	for _, in := range []string{"+g", "+18", "badval", "9999", "+1bib"} {
		if _, err := ParseSize(in); err == nil {
			t.Errorf("ParseSize(%q) expected error", in)
		}
	}
}

func TestSizeIsWithin(t *testing.T) {
	f, _ := ParseSize("-1k")
	if !f.IsWithin(999) || !f.IsWithin(1000) || f.IsWithin(1001) {
		t.Error("max filter boundary incorrect")
	}
	f, _ = ParseSize("+1k")
	if f.IsWithin(999) || !f.IsWithin(1000) || !f.IsWithin(1001) {
		t.Error("min filter boundary incorrect")
	}
}
