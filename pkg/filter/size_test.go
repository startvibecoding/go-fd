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

func TestParseSizeOverflow(t *testing.T) {
	// Values that parse fine as uint64 but overflow when multiplied by the unit.
	// max uint64 = 18446744073709551615.
	// 18446744073710 * 1T (=10^12) = 18446744073710000000000 > max uint64.
	if _, err := ParseSize("+18446744073710t"); err == nil {
		t.Error("expected overflow error for 18446744073710t")
	}
	// 18446744074 * 1G (=10^9) also overflows.
	if _, err := ParseSize("+18446744074g"); err == nil {
		t.Error("expected overflow error for 18446744074g")
	}
}
