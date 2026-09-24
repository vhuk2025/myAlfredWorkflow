package jsonfmt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	out, err := Format([]byte(" \n{\"z\":1,\"a\":[1,2]}  "))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "{\n  \"z\": 1,\n  \"a\": [\n    1,\n    2\n  ]\n}\n" {
		t.Fatalf("formatted = %q", out)
	}
	if _, err := Format([]byte("   ")); err == nil || !strings.Contains(err.Error(), "剪贴板是空的") {
		t.Fatalf("empty err = %v", err)
	}
	if _, err := Format([]byte("not-json")); err == nil || !strings.Contains(err.Error(), "不是合法 JSON") {
		t.Fatalf("invalid err = %v", err)
	}
}

func TestRunOpensEditor(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 24, 11, 30, 0, 0, time.UTC)
	var gotApp, gotPath string
	res, err := Run([]byte(`{"a":1}`), Config{
		Editor: "VSCodium",
		Dir:    dir,
		Now:    now,
		Format: func(input []byte) ([]byte, error) {
			if string(input) != `{"a":1}` {
				t.Fatalf("input = %s", input)
			}
			return []byte("{\n  \"a\": 1\n}\n"), nil
		},
		Open: func(app, path string) error {
			gotApp, gotPath = app, path
			return nil
		},
		Exists: func(string) bool { return false },
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "clipboard-20260924-113000.json")
	if res.Editor != "VSCodium" || res.Path != want || gotApp != "VSCodium" || gotPath != want {
		t.Fatalf("result=%+v opened %s %s", res, gotApp, gotPath)
	}
	body, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "{\n  \"a\": 1\n}\n" {
		t.Fatalf("file = %q", body)
	}
}

func TestSelectEditor(t *testing.T) {
	if got := SelectEditor("  GoLand ", nil); got != "GoLand" {
		t.Fatalf("override = %q", got)
	}
	got := SelectEditor("", func(name string) bool { return name == "Zed" || name == "TextEdit" })
	if got != "Zed" {
		t.Fatalf("selected = %q", got)
	}
	if got := SelectEditor("", func(string) bool { return false }); got != "TextEdit" {
		t.Fatalf("fallback = %q", got)
	}
}
