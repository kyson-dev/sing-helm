package export

import (
	"fmt"
	"strconv"
	"strings"
)

func versionLess(a, b string) (bool, error) {
	av, err := parseVersion(a)
	if err != nil {
		return false, err
	}
	bv, err := parseVersion(b)
	if err != nil {
		return false, err
	}

	for i := 0; i < 3; i++ {
		if av[i] < bv[i] {
			return true, nil
		}
		if av[i] > bv[i] {
			return false, nil
		}
	}
	return false, nil
}

func parseVersion(v string) ([3]int, error) {
	var out [3]int
	trimmed := strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if trimmed == "" {
		return out, fmt.Errorf("invalid version: %q", v)
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) > 3 {
		parts = parts[:3]
	}

	for i := 0; i < 3; i++ {
		if i >= len(parts) {
			out[i] = 0
			continue
		}
		part := strings.TrimSpace(parts[i])
		if part == "" {
			return out, fmt.Errorf("invalid version: %q", v)
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return out, fmt.Errorf("invalid version: %q", v)
		}
		out[i] = value
	}

	return out, nil
}
