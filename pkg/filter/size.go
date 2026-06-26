// Package filter implements the result filters used by fd: size, modification
// time and (on unix) ownership constraints.
package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SI prefixes (powers of 10) and binary prefixes (powers of 2).
const (
	kilo = 1000
	mega = kilo * 1000
	giga = mega * 1000
	tera = giga * 1000

	kibi = 1024
	mebi = kibi * 1024
	gibi = mebi * 1024
	tebi = gibi * 1024
)

// SizeKind distinguishes the comparison used by a SizeFilter.
type SizeKind int

const (
	// SizeMin requires file size >= Limit.
	SizeMin SizeKind = iota
	// SizeMax requires file size <= Limit.
	SizeMax
	// SizeEquals requires file size == Limit.
	SizeEquals
)

// SizeFilter constrains results based on file size.
type SizeFilter struct {
	Kind  SizeKind
	Limit uint64
}

var sizeRe = regexp.MustCompile(`(?i)^([+-]?)(\d+)(b|[kmgt]i?b?)$`)

// ParseSize parses a size constraint string like "+1m", "-500k", "10ki".
func ParseSize(s string) (SizeFilter, error) {
	m := sizeRe.FindStringSubmatch(s)
	if m == nil {
		return SizeFilter{}, fmt.Errorf("'%s' is not a valid size constraint. See 'fd --help'.", s)
	}
	limitKind := m[1]
	quantity, err := strconv.ParseUint(m[2], 10, 64)
	if err != nil {
		return SizeFilter{}, fmt.Errorf("'%s' is not a valid size constraint. See 'fd --help'.", s)
	}
	unit := strings.ToLower(m[3])
	var mult uint64
	switch {
	case strings.HasPrefix(unit, "ki"):
		mult = kibi
	case strings.HasPrefix(unit, "k"):
		mult = kilo
	case strings.HasPrefix(unit, "mi"):
		mult = mebi
	case strings.HasPrefix(unit, "m"):
		mult = mega
	case strings.HasPrefix(unit, "gi"):
		mult = gibi
	case strings.HasPrefix(unit, "g"):
		mult = giga
	case strings.HasPrefix(unit, "ti"):
		mult = tebi
	case strings.HasPrefix(unit, "t"):
		mult = tera
	case unit == "b":
		mult = 1
	default:
		return SizeFilter{}, fmt.Errorf("'%s' is not a valid size constraint. See 'fd --help'.", s)
	}
	size := quantity * mult
	switch limitKind {
	case "+":
		return SizeFilter{Kind: SizeMin, Limit: size}, nil
	case "-":
		return SizeFilter{Kind: SizeMax, Limit: size}, nil
	case "":
		return SizeFilter{Kind: SizeEquals, Limit: size}, nil
	default:
		return SizeFilter{}, fmt.Errorf("'%s' is not a valid size constraint. See 'fd --help'.", s)
	}
}

// IsWithin reports whether the given size satisfies the filter.
func (f SizeFilter) IsWithin(size uint64) bool {
	switch f.Kind {
	case SizeMax:
		return size <= f.Limit
	case SizeMin:
		return size >= f.Limit
	case SizeEquals:
		return size == f.Limit
	}
	return false
}
