package store

import (
	"strings"
	"time"
)

const (
	dateLayout = "2006-01-02"
	// dateEnd is where the day ends in a value that carries a time of day.
	dateEnd = len(dateLayout)
)

// DateOnly returns the YYYY-MM-DD day at the start of s, dropping any time of
// day that follows it ("2026-10-30T14:05:09Z" gives "2026-10-30"). It
// reports false when s does not start with a real calendar day, or when
// the day is followed by anything but a time separated by "T" or a space.
func DateOnly(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) < dateEnd {
		return "", false
	}
	if rest := s[dateEnd:]; rest != "" && rest[0] != 'T' && rest[0] != ' ' {
		return "", false
	}
	if _, err := time.Parse(dateLayout, s[:dateEnd]); err != nil {
		return "", false
	}
	return s[:dateEnd], true
}
