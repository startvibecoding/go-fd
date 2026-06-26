package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TimeKind distinguishes before/after comparisons.
type TimeKind int

const (
	// TimeBefore matches files modified before the limit.
	TimeBefore TimeKind = iota
	// TimeAfter matches files modified after the limit.
	TimeAfter
)

// TimeFilter constrains results based on modification time.
type TimeFilter struct {
	Kind  TimeKind
	Limit time.Time
}

// durationRe matches simple duration spans like "10h", "1d", "2weeks", "35min".
var durationRe = regexp.MustCompile(`(?i)^\s*((?:\d+\s*[a-z]+\s*)+)$`)

var unitRe = regexp.MustCompile(`(?i)(\d+)\s*([a-z]+)`)

// now is overridable for testing.
var now = time.Now

func parseDuration(s string) (time.Duration, bool) {
	if !durationRe.MatchString(s) {
		return 0, false
	}
	matches := unitRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0, false
	}
	var total time.Duration
	for _, m := range matches {
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return 0, false
		}
		unit := strings.ToLower(m[2])
		var d time.Duration
		switch unit {
		case "s", "sec", "secs", "second", "seconds":
			d = time.Second
		case "m", "min", "mins", "minute", "minutes":
			d = time.Minute
		case "h", "hr", "hrs", "hour", "hours":
			d = time.Hour
		case "d", "day", "days":
			d = 24 * time.Hour
		case "w", "week", "weeks":
			d = 7 * 24 * time.Hour
		case "mo", "month", "months":
			d = 30 * 24 * time.Hour
		case "y", "year", "years":
			d = 365 * 24 * time.Hour
		default:
			return 0, false
		}
		total += time.Duration(n) * d
	}
	return total, true
}

func parseTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	// Duration relative to now.
	if d, ok := parseDuration(s); ok {
		return now().Add(-d), true
	}
	// @unix timestamp
	if strings.HasPrefix(s, "@") {
		secs, err := strconv.ParseInt(s[1:], 10, 64)
		if err != nil {
			return time.Time{}, false
		}
		return time.Unix(secs, 0), true
	}
	// RFC3339 / common datetime formats.
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// After returns a filter matching files modified after the given time spec.
func After(s string) (TimeFilter, error) {
	t, ok := parseTime(s)
	if !ok {
		return TimeFilter{}, fmt.Errorf("'%s' is not a valid date or duration. See 'fd --help'.", s)
	}
	return TimeFilter{Kind: TimeAfter, Limit: t}, nil
}

// Before returns a filter matching files modified before the given time spec.
func Before(s string) (TimeFilter, error) {
	t, ok := parseTime(s)
	if !ok {
		return TimeFilter{}, fmt.Errorf("'%s' is not a valid date or duration. See 'fd --help'.", s)
	}
	return TimeFilter{Kind: TimeBefore, Limit: t}, nil
}

// AppliesTo reports whether the modification time satisfies the filter.
func (f TimeFilter) AppliesTo(t time.Time) bool {
	switch f.Kind {
	case TimeBefore:
		return t.Before(f.Limit)
	case TimeAfter:
		return t.After(f.Limit)
	}
	return false
}
