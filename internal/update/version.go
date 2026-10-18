package update

import (
	"strconv"
	"strings"
)

// IsNewerRelease rejects malformed or overflowing version components rather than treating them as zero.
func IsNewerRelease(latest, current string) bool {
	parse := func(v string) ([3]uint64, bool) {
		var out [3]uint64
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		parts := strings.Split(v, ".")
		if len(parts) != 3 {
			return out, false
		}
		for i, part := range parts {
			if part == "" {
				return out, false
			}
			for _, c := range part {
				if c < '0' || c > '9' {
					return out, false
				}
			}
			n, err := strconv.ParseUint(part, 10, 64)
			if err != nil {
				return out, false
			}
			out[i] = n
		}
		return out, true
	}
	l, okL := parse(latest)
	c, okC := parse(current)
	if !okL || !okC {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}
