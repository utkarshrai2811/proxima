package proxy

import (
	"fmt"
	"strconv"
	"strings"
)

// bodySizeUnits maps a size suffix to its multiplier in bytes. Ordered longest
// suffix first so "KB"/"MB"/"GB" are matched before the bare "B".
var bodySizeUnits = []struct {
	suffix string
	mult   int64
}{
	{"KB", 1024},
	{"MB", 1024 * 1024},
	{"GB", 1024 * 1024 * 1024},
	{"B", 1},
}

// ParseBodySize parses a human-readable size string like "10MB" into bytes.
// Supported suffixes (case-insensitive): B, KB, MB, GB. A bare number is
// interpreted as bytes.
func ParseBodySize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))

	for _, u := range bodySizeUnits {
		if strings.HasSuffix(s, u.suffix) {
			num := strings.TrimSpace(strings.TrimSuffix(s, u.suffix))

			n, err := strconv.ParseInt(num, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size %q: %w", s, err)
			}

			if n < 0 {
				return 0, fmt.Errorf("invalid size %q: must not be negative", s)
			}

			return n * u.mult, nil
		}
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q (use B, KB, MB, or GB suffix)", s)
	}

	return n, nil
}
