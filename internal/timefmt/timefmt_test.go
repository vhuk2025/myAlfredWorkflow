package timefmt

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

func fixedNow() time.Time {
	loc := time.FixedZone("CST", 8*3600)
	return time.Date(2026, 3, 22, 15, 4, 5, 0, loc)
}

func TestScriptFilterAll(t *testing.T) {
	now := fixedNow()
	out := ScriptFilter(now, "")
	if len(out.Items) != 6 {
		t.Fatalf("len = %d, want 6", len(out.Items))
	}
	if out.Items[0].Title != "2026-03-22 15:04:05" {
		t.Fatalf("datetime = %q", out.Items[0].Title)
	}
	if out.Items[0].Arg != out.Items[0].Title || !out.Items[0].Valid {
		t.Fatalf("datetime item not copyable: %+v", out.Items[0])
	}
	if !strings.HasPrefix(out.Items[0].Subtitle, weekdays[now.Weekday()]) {
		t.Fatalf("subtitle = %q", out.Items[0].Subtitle)
	}
	if out.Items[3].Title != "2026-03-22T15:04:05+08:00" {
		t.Fatalf("iso = %q", out.Items[3].Title)
	}
	if out.Items[4].Title != strconv.FormatInt(now.Unix(), 10) {
		t.Fatalf("unix = %q", out.Items[4].Title)
	}
	if out.Items[5].Title != strconv.FormatInt(now.UnixMilli(), 10) {
		t.Fatalf("unixms = %q", out.Items[5].Title)
	}
}

func TestScriptFilterInput(t *testing.T) {
	now := fixedNow()
	out := ScriptFilter(now, "2025-02-02 05")
	if len(out.Items) != 6 {
		t.Fatalf("len = %d, want 6", len(out.Items))
	}
	want := time.Date(2025, 2, 2, 5, 0, 0, 0, now.Location())
	if out.Items[0].UID != "unix" || out.Items[0].Title != strconv.FormatInt(want.Unix(), 10) || !out.Items[0].Valid {
		t.Fatalf("unix first = %+v", out.Items[0])
	}
	if out.Items[1].UID != "unixms" || out.Items[2].Title != "2025-02-02 05:00:00" || out.Items[3].Title != "2025-02-02" {
		t.Fatalf("date order = %+v", out.Items)
	}

	month := ScriptFilter(now, "2025-01")
	if month.Items[0].UID != "unix" || month.Items[2].Title != "2025-01-01 00:00:00" {
		t.Fatalf("year-month = %q %q", month.Items[0].UID, month.Items[2].Title)
	}

	slash := ScriptFilter(now, "2025/2/2 5:04")
	if slash.Items[0].UID != "unix" || slash.Items[2].Title != "2025-02-02 05:04:00" {
		t.Fatalf("slash = %+v", slash.Items[2])
	}

	fromUnix := ScriptFilter(now, strconv.FormatInt(want.Unix(), 10))
	if fromUnix.Items[0].UID != "date" || fromUnix.Items[0].Title != "2025-02-02" {
		t.Fatalf("from unix = %+v", fromUnix.Items[0])
	}
	if fromUnix.Items[1].Title != "2025-02-02 05:00:00" {
		t.Fatalf("from unix datetime = %q", fromUnix.Items[1].Title)
	}

	none := ScriptFilter(now, "nope")
	if len(none.Items) != 1 || none.Items[0].Valid {
		t.Fatalf("invalid query = %+v", none.Items)
	}

	raw, err := json.Marshal(fromUnix)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "\"uid\"") {
		t.Fatalf("uid leaked to Alfred JSON: %s", raw)
	}
}
