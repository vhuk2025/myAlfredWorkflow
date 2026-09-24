package timefmt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	dateTimePattern = regexp.MustCompile(`^(\d{4})(?:[-/.](\d{1,2})(?:[-/.](\d{1,2})(?: (\d{1,2})(?::(\d{1,2})(?::(\d{1,2}))?)?)?)?)?(Z|[+-]\d{2}:?\d{2})?$`)
	zoneSpace       = regexp.MustCompile(` ([+-]\d{2}:?\d{2}|Z)$`)
)

// Parse reads a datetime. A missing month or day defaults to 1.
// Missing minutes and seconds default to 0.
// Accepted forms include 2025, 2025-01, 2025-02-02, 2025-02-02 05,
// 2025/2/2 5, RFC3339, compact YYYYMMDD[HH[mm[ss]]], and 10/13-digit Unix time.
func Parse(query string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.Local
	}
	raw := strings.TrimSpace(query)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}

	iso := strings.NewReplacer(" ", "T", "：", ":", "－", "-", "—", "-").Replace(raw)
	if t, err := time.Parse(time.RFC3339, iso); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, iso); err == nil {
		return t, nil
	}

	s := strings.NewReplacer("T", " ", "：", ":", "－", "-", "—", "-", "/", "-", ".", "-").Replace(raw)
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimRight(s, " -:")
	s = zoneSpace.ReplaceAllString(s, "$1")
	if t, ok := parseDateTime(s, loc); ok {
		return t, nil
	}
	if t, ok := parseDigits(s, loc); ok {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q", query)
}

func parseDateTime(s string, loc *time.Location) (time.Time, bool) {
	match := dateTimePattern.FindStringSubmatch(s)
	if match == nil {
		return time.Time{}, false
	}
	year, ok1 := atoi(match[1])
	month, ok2 := atoi(match[2])
	day, ok3 := atoi(match[3])
	hour, ok4 := atoi(match[4])
	minute, ok5 := atoi(match[5])
	second, ok6 := atoi(match[6])
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 {
		return time.Time{}, false
	}
	if match[2] == "" {
		month = 1
	}
	if match[3] == "" {
		day = 1
	}
	whenLoc := loc
	if match[7] != "" {
		parsed, ok := parseZone(match[7])
		if !ok {
			return time.Time{}, false
		}
		whenLoc = parsed
	}
	return buildTime(year, month, day, hour, minute, second, whenLoc)
}

func parseDigits(s string, loc *time.Location) (time.Time, bool) {
	if s == "" || strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return time.Time{}, false
	}
	switch len(s) {
	case 10:
		sec, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return time.Time{}, false
		}
		return time.Unix(sec, 0).In(loc), true
	case 13:
		ms, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return time.Time{}, false
		}
		return time.UnixMilli(ms).In(loc), true
	case 4:
		return buildTime(mustAtoi(s), 1, 1, 0, 0, 0, loc)
	case 6:
		return buildTime(mustAtoi(s[0:4]), mustAtoi(s[4:6]), 1, 0, 0, 0, loc)
	case 8:
		return buildTime(mustAtoi(s[0:4]), mustAtoi(s[4:6]), mustAtoi(s[6:8]), 0, 0, 0, loc)
	case 12:
		return buildTime(mustAtoi(s[0:4]), mustAtoi(s[4:6]), mustAtoi(s[6:8]), mustAtoi(s[8:10]), mustAtoi(s[10:12]), 0, loc)
	case 14:
		return buildTime(mustAtoi(s[0:4]), mustAtoi(s[4:6]), mustAtoi(s[6:8]), mustAtoi(s[8:10]), mustAtoi(s[10:12]), mustAtoi(s[12:14]), loc)
	default:
		return time.Time{}, false
	}
}

func parseZone(tz string) (*time.Location, bool) {
	if tz == "Z" {
		return time.UTC, true
	}
	sign := 1
	switch tz[0] {
	case '+':
	case '-':
		sign = -1
	default:
		return nil, false
	}
	digits := strings.ReplaceAll(tz[1:], ":", "")
	if len(digits) != 4 {
		return nil, false
	}
	hour, ok1 := atoi(digits[:2])
	minute, ok2 := atoi(digits[2:])
	if !ok1 || !ok2 || hour > 23 || minute > 59 {
		return nil, false
	}
	return time.FixedZone(tz, sign*(hour*3600+minute*60)), true
}

func buildTime(year, month, day, hour, minute, second int, loc *time.Location) (time.Time, bool) {
	if month < 1 || month > 12 || day < 1 || hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 {
		return time.Time{}, false
	}
	t := time.Date(year, time.Month(month), day, hour, minute, second, 0, loc)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, false
	}
	return t, true
}

func atoi(s string) (int, bool) {
	if s == "" {
		return 0, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
