package filter

import (
	"testing"
	"time"
)

func TestParseDurationFilter(t *testing.T) {
	fixed := time.Date(2024, 2, 12, 7, 36, 52, 0, time.UTC)
	now = func() time.Time { return fixed }
	defer func() { now = time.Now }()

	f, err := After("1min")
	if err != nil {
		t.Fatal(err)
	}
	// 30s ago is after (now-1min); 2min ago is before.
	if !f.AppliesTo(fixed.Add(-30 * time.Second)) {
		t.Error("30s ago should be after the 1min limit")
	}
	if f.AppliesTo(fixed.Add(-2 * time.Minute)) {
		t.Error("2min ago should not be after the 1min limit")
	}

	b, err := Before("1min")
	if err != nil {
		t.Fatal(err)
	}
	if !b.AppliesTo(fixed.Add(-2 * time.Minute)) {
		t.Error("2min ago should be before the 1min limit")
	}
}

func TestParseTimestamp(t *testing.T) {
	f, err := After("@1707723412")
	if err != nil {
		t.Fatal(err)
	}
	if !f.AppliesTo(time.Unix(1707723413, 0)) {
		t.Error("expected after timestamp")
	}
}
