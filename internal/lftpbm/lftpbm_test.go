package lftpbm

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const sampleList = "" +
	"agent-pre-server\tftp://S866_699pic:XXXX@192.168.14.239/699pic_ai_agent\n" +
	"ai-drawing-pro  \tftp://user:secret@192.168.15.69/699pic_go_schedule\n" +
	"support-view-pro\tftp://S866_699pic:XXXX@192.168.14.62/699pic_sp\n"

func TestParseListTrimsPadding(t *testing.T) {
	marks := ParseList(sampleList + "\nbroken-without-tab\n\tftp://x\n")
	if len(marks) != 3 {
		t.Fatalf("len = %d", len(marks))
	}
	if marks[1].Name != "ai-drawing-pro" {
		t.Fatalf("name = %q", marks[1].Name)
	}
	if marks[1].Target() != "ftp://192.168.15.69/699pic_go_schedule" {
		t.Fatalf("target = %q", marks[1].Target())
	}
}

func TestScriptFilterHidesPassword(t *testing.T) {
	out := ScriptFilter(ParseList(sampleList), "")
	if len(out.Items) != 3 || !out.Items[0].Valid || out.Items[0].Arg != "agent-pre-server" {
		t.Fatalf("items = %+v", out.Items)
	}
	blob, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), "secret") || strings.Contains(string(blob), "XXXX") {
		t.Fatalf("credential leaked: %s", blob)
	}
	if !strings.Contains(out.Items[1].Subtitle, "ftp://192.168.15.69/699pic_go_schedule · 回车在终端打开") {
		t.Fatalf("subtitle = %q", out.Items[1].Subtitle)
	}

	filtered := ScriptFilter(ParseList(sampleList), "Support 14.62")
	if len(filtered.Items) != 1 || filtered.Items[0].Title != "support-view-pro" {
		t.Fatalf("filtered = %+v", filtered.Items)
	}
	none := ScriptFilter(ParseList(sampleList), "missing")
	if len(none.Items) != 1 || none.Items[0].Valid || none.Items[0].Title != "没有匹配的 bookmark" {
		t.Fatalf("none = %+v", none.Items)
	}
	empty := ScriptFilter(nil, "")
	if empty.Items[0].Valid || empty.Items[0].Title != "没有 lftp bookmark" {
		t.Fatalf("empty = %+v", empty.Items[0])
	}
}

func TestOpenRunsQuotedCommand(t *testing.T) {
	var gotApp, gotCmd string
	msg, err := Open("ai-drawing-pro", Deps{
		Resolve: func() (string, error) { return "/opt/homebrew/bin/lftp", nil },
		List: func() ([]Bookmark, error) {
			return ParseList(sampleList), nil
		},
		Run: func(app, command string) error {
			gotApp, gotCmd = app, command
			return nil
		},
		Terminal: " iTerm2 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg != "已在终端打开 ai-drawing-pro" || gotApp != "iTerm2" {
		t.Fatalf("msg=%q app=%q", msg, gotApp)
	}
	if gotCmd != "'/opt/homebrew/bin/lftp' 'ai-drawing-pro'" {
		t.Fatalf("cmd = %q", gotCmd)
	}
}

func TestOpenRejectsUnknownOrUnsafe(t *testing.T) {
	called := false
	deps := Deps{
		Resolve: func() (string, error) { return "/bin/lftp", nil },
		List: func() ([]Bookmark, error) {
			return ParseList("ok name\tftp://host/path\n"), nil
		},
		Run: func(string, string) error {
			called = true
			return nil
		},
	}
	if _, err := Open("missing", deps); err == nil || !strings.Contains(err.Error(), "没有 bookmark") {
		t.Fatalf("missing err = %v", err)
	}
	if _, err := Open("-rf", deps); err == nil || !strings.Contains(err.Error(), "无效的 bookmark") {
		t.Fatalf("dash err = %v", err)
	}
	msg, err := Open("ok name", deps)
	if err != nil {
		t.Fatal(err)
	}
	if !called || !strings.Contains(msg, "ok name") {
		t.Fatalf("called=%v msg=%q", called, msg)
	}
}

func TestLaunchScriptSplitsByTerminal(t *testing.T) {
	ghostty, ghosttyArgs := launchScript("ghostty", "'/bin/lftp' 'demo'")
	if !strings.Contains(ghostty, `tell application "Ghostty"`) || !strings.Contains(ghostty, "new tab in front window") || strings.Contains(ghostty, "create window") {
		t.Fatalf("ghostty script = %q", ghostty)
	}
	if len(ghosttyArgs) != 1 || ghosttyArgs[0] != "'/bin/lftp' 'demo'" {
		t.Fatalf("ghostty args = %#v", ghosttyArgs)
	}

	term, termArgs := launchScript("Terminal", "lftp demo")
	if strings.Contains(term, "Ghostty") || strings.Contains(term, "iTerm") || strings.Contains(term, "create window") {
		t.Fatalf("terminal script = %q", term)
	}
	if len(termArgs) != 1 || termArgs[0] != "lftp demo" {
		t.Fatalf("terminal args = %#v", termArgs)
	}

	iterm, itermArgs := launchScript("iTerm2", "lftp demo")
	if !strings.Contains(iterm, `tell application "iTerm"`) || strings.Contains(iterm, "Ghostty") {
		t.Fatalf("iterm script = %q", iterm)
	}
	if len(itermArgs) != 1 || itermArgs[0] != "lftp demo" {
		t.Fatalf("iterm args = %#v", itermArgs)
	}
}

func TestLaunchScriptCompiles(t *testing.T) {
	if _, err := exec.LookPath("osacompile"); err != nil {
		t.Skip(err)
	}
	checks := []struct {
		app  string
		need string
	}{
		{app: "Terminal"},
		{app: "ghostty", need: "/Applications/Ghostty.app"},
	}
	for _, check := range checks {
		if check.need != "" {
			if _, err := os.Stat(check.need); err != nil {
				t.Skip(err)
			}
		}
		script, _ := launchScript(check.app, "/bin/echo ok")
		out, err := exec.Command("osacompile", "-o", filepath.Join(t.TempDir(), "launch.scpt"), "-e", script).CombinedOutput()
		if err != nil {
			t.Fatalf("%s compile: %v\n%s\n%s", check.app, err, out, script)
		}
	}
}

func TestResolveBinaryFallback(t *testing.T) {
	path, err := ResolveBinary(func(string) (string, error) {
		return "", osPathError("not found")
	}, func(path string) bool {
		return path == "/usr/local/bin/lftp"
	})
	if err != nil || path != "/usr/local/bin/lftp" {
		t.Fatalf("path=%q err=%v", path, err)
	}
	if _, err := ResolveBinary(func(string) (string, error) {
		return "", osPathError("not found")
	}, func(string) bool { return false }); err == nil {
		t.Fatal("expected missing lftp")
	}
}

type osPathError string

func (e osPathError) Error() string { return string(e) }
