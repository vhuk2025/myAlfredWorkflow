package timefmt

import (
	"strconv"
	"strings"
	"time"
)

// Item is one Alfred Script Filter result.
// UID is kept for tests only. Alfred reorders items that have a uid
// based on which result was chosen before, which would float the timestamp back to the top.
type Item struct {
	UID      string `json:"-"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg"`
	Valid    bool   `json:"valid"`
	Icon     *Icon  `json:"icon,omitempty"`
}

// Icon points at a file inside the workflow bundle.
type Icon struct {
	Path string `json:"path"`
}

// Output is an Alfred Script Filter response.
type Output struct {
	Items []Item `json:"items"`
}

type spec struct {
	uid      string
	subtitle string
	render   func(time.Time) string
}

var weekdays = [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

var formats = []spec{
	{
		uid:      "datetime",
		subtitle: "日期时间",
		render: func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
	},
	{
		uid:      "date",
		subtitle: "日期",
		render: func(t time.Time) string {
			return t.Format("2006-01-02")
		},
	},
	{
		uid:      "time",
		subtitle: "时间",
		render: func(t time.Time) string {
			return t.Format("15:04:05")
		},
	},
	{
		uid:      "iso",
		subtitle: "ISO 8601",
		render: func(t time.Time) string {
			return t.Format(time.RFC3339)
		},
	},
	{
		uid:      "unix",
		subtitle: "Unix 时间戳（秒）",
		render: func(t time.Time) string {
			return strconv.FormatInt(t.Unix(), 10)
		},
	},
	{
		uid:      "unixms",
		subtitle: "Unix 时间戳（毫秒）",
		render: func(t time.Time) string {
			return strconv.FormatInt(t.UnixMilli(), 10)
		},
	},
}

// ScriptFilter builds the Alfred result list.
// An empty query uses now. Any other query is parsed as a datetime and listed in every format.
// A calendar input lists Unix timestamps first. A Unix timestamp lists the date first.
func ScriptFilter(now time.Time, query string) Output {
	when := now
	if strings.TrimSpace(query) != "" {
		parsed, err := Parse(query, now.Location())
		if err != nil {
			return Output{Items: []Item{{
				UID:      "invalid",
				Title:    "无法解析时间",
				Subtitle: "例如 2025-02-02 05",
				Valid:    false,
			}}}
		}
		when = parsed
	}
	items := make([]Item, 0, len(formats))
	for _, format := range orderedFormats(query) {
		value := format.render(when)
		subtitle := format.subtitle + " · 回车复制"
		if format.uid == "datetime" {
			subtitle = weekdays[when.Weekday()] + " · " + subtitle
		}
		items = append(items, Item{
			UID:      format.uid,
			Title:    value,
			Subtitle: subtitle,
			Arg:      value,
			Valid:    true,
			Icon:     &Icon{Path: "icon.png"},
		})
	}
	return Output{Items: items}
}

func orderedFormats(query string) []spec {
	byID := make(map[string]spec, len(formats))
	for _, format := range formats {
		byID[format.uid] = format
	}
	ordered := make([]spec, 0, len(formats))
	for _, id := range formatOrder(query) {
		ordered = append(ordered, byID[id])
	}
	return ordered
}

func formatOrder(query string) []string {
	switch inputKind(strings.TrimSpace(query)) {
	case kindUnix:
		return []string{"date", "datetime", "time", "iso", "unix", "unixms"}
	case kindDate:
		return []string{"unix", "unixms", "datetime", "date", "time", "iso"}
	default:
		return []string{"datetime", "date", "time", "iso", "unix", "unixms"}
	}
}

type kind int

const (
	kindNow kind = iota
	kindDate
	kindUnix
)

func inputKind(query string) kind {
	if query == "" {
		return kindNow
	}
	if isUnixDigits(query) {
		return kindUnix
	}
	return kindDate
}

func isUnixDigits(s string) bool {
	if len(s) != 10 && len(s) != 13 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
