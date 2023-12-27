package tt

import (
	"strings"
)

// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241

func match(filter, name string) bool {
	if filter == "#" {
		if name[0] == '$' {
			return false
		}
		return true
	}

	hasNumberSign := strings.HasSuffix(filter, "#")
	hasPlusSign := strings.Contains(filter, "+")

	switch {
	case hasNumberSign && !hasPlusSign:
		// a/b/#  a/b
		prefix := filter[:len(filter)-2]
		return strings.HasPrefix(name, prefix)

	case hasNumberSign && hasPlusSign:
		// a/+/#  a/b/c
		filter = filter[:len(filter)-2]
		filterLevels := strings.Split(filter, "/")
		nameLevels := strings.Split(name, "/")
		for i, f := range filterLevels {
			// a/b/+/#  a/b
			if i == len(nameLevels) {
				return false
			}
			if f == "+" {
				continue
			}
			if f != nameLevels[i] {
				return false
			}
		}
		return true

	case !hasNumberSign && hasPlusSign:
		filterLevels := strings.Split(filter, "/")
		nameLevels := strings.Split(name, "/")
		if len(nameLevels) > len(filterLevels) {
			return false
		}
		for i, f := range filterLevels {
			// a/+/+  a/b
			if i == len(nameLevels) {
				return false
			}
			if f == "+" {
				if i == 0 && nameLevels[i][0] == '$' {
					return false
				}
				continue
			}
			if f != nameLevels[i] {
				return false
			}
		}
		return true
	}
	return filter == name
}
