//go:build !unix

package filter

import "os"

// OwnerFilter is a no-op on non-unix platforms.
type OwnerFilter struct{}

// ParseOwner is unsupported on non-unix platforms.
func ParseOwner(input string) (OwnerFilter, bool, error) {
	return OwnerFilter{}, false, nil
}

// Matches always returns true on non-unix platforms.
func (f OwnerFilter) Matches(info os.FileInfo) bool { return true }
