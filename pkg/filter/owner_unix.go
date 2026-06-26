//go:build unix

package filter

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

type checkKind int

const (
	checkIgnore checkKind = iota
	checkEqual
	checkNotEq
)

type check struct {
	kind checkKind
	val  uint32
}

func (c check) matches(v uint32) bool {
	switch c.kind {
	case checkEqual:
		return v == c.val
	case checkNotEq:
		return v != c.val
	default:
		return true
	}
}

// OwnerFilter constrains results by owning user and/or group.
type OwnerFilter struct {
	uid check
	gid check
}

// ParseOwner parses an owner constraint of the form [(user|uid)][:(group|gid)].
// Either side may be prefixed with '!' to negate. Returns ok=false when the
// constraint is a no-op (e.g. "" or ":").
func ParseOwner(input string) (OwnerFilter, bool, error) {
	parts := strings.Split(input, ":")
	if len(parts) > 2 {
		return OwnerFilter{}, false, fmt.Errorf("more than one ':' present in owner string '%s'. See 'fd --help'.", input)
	}
	var fst, snd string
	fst = parts[0]
	if len(parts) == 2 {
		snd = parts[1]
	}

	uid, err := parseCheck(fst, func(s string) (uint32, error) {
		if n, err := strconv.ParseUint(s, 10, 32); err == nil {
			return uint32(n), nil
		}
		u, err := user.Lookup(s)
		if err != nil {
			return 0, fmt.Errorf("'%s' is not a recognized user name", s)
		}
		n, _ := strconv.ParseUint(u.Uid, 10, 32)
		return uint32(n), nil
	})
	if err != nil {
		return OwnerFilter{}, false, err
	}
	gid, err := parseCheck(snd, func(s string) (uint32, error) {
		if n, err := strconv.ParseUint(s, 10, 32); err == nil {
			return uint32(n), nil
		}
		g, err := user.LookupGroup(s)
		if err != nil {
			return 0, fmt.Errorf("'%s' is not a recognized group name", s)
		}
		n, _ := strconv.ParseUint(g.Gid, 10, 32)
		return uint32(n), nil
	})
	if err != nil {
		return OwnerFilter{}, false, err
	}

	f := OwnerFilter{uid: uid, gid: gid}
	if uid.kind == checkIgnore && gid.kind == checkIgnore {
		return f, false, nil
	}
	return f, true, nil
}

func parseCheck(s string, resolve func(string) (uint32, error)) (check, error) {
	if s == "" {
		return check{kind: checkIgnore}, nil
	}
	negate := false
	if strings.HasPrefix(s, "!") {
		negate = true
		s = s[1:]
	}
	v, err := resolve(s)
	if err != nil {
		return check{}, err
	}
	if negate {
		return check{kind: checkNotEq, val: v}, nil
	}
	return check{kind: checkEqual, val: v}, nil
}

// Matches reports whether the file described by info satisfies the constraint.
func (f OwnerFilter) Matches(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return true
	}
	return f.uid.matches(st.Uid) && f.gid.matches(st.Gid)
}
