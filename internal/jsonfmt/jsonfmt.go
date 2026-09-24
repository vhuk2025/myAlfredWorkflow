package jsonfmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Config controls where the formatted file is written and which editor opens it.
type Config struct {
	Editor string
	Dir    string
	Now    time.Time
	Format func([]byte) ([]byte, error)
	Open   func(app, path string) error
	Exists func(app string) bool
}

// Result is a formatted file opened in an editor.
type Result struct {
	Path    string
	Editor  string
	Message string
}

var editorCandidates = []string{
	"VSCodium",
	"Cursor",
	"Visual Studio Code",
	"Zed",
	"Sublime Text",
	"GoLand",
	"PhpStorm",
	"Nova",
	"BBEdit",
	"TextEdit",
}

// Run formats JSON and opens it in an editor.
func Run(input []byte, cfg Config) (Result, error) {
	if cfg.Format == nil {
		cfg.Format = Format
	}
	if cfg.Open == nil {
		cfg.Open = OpenApp
	}
	if cfg.Exists == nil {
		cfg.Exists = AppExists
	}
	if cfg.Now.IsZero() {
		cfg.Now = time.Now()
	}
	if cfg.Dir == "" {
		cfg.Dir = filepath.Join(os.TempDir(), "alfred-json")
	}

	formatted, err := cfg.Format(input)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return Result{}, fmt.Errorf("无法创建临时目录: %w", err)
	}
	path := filepath.Join(cfg.Dir, "clipboard-"+cfg.Now.Format("20060102-150405")+".json")
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return Result{}, fmt.Errorf("无法写入文件: %w", err)
	}
	editor := SelectEditor(cfg.Editor, cfg.Exists)
	if err := cfg.Open(editor, path); err != nil {
		return Result{}, err
	}
	return Result{
		Path:    path,
		Editor:  editor,
		Message: "已用 " + editor + " 打开",
	}, nil
}

// Format pretty-prints JSON without reordering keys.
func Format(input []byte) ([]byte, error) {
	raw := bytes.TrimSpace(input)
	if len(raw) == 0 {
		return nil, fmt.Errorf("剪贴板是空的")
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		line, _, _ := strings.Cut(err.Error(), "\n")
		return nil, fmt.Errorf("剪贴板内容不是合法 JSON: %s", line)
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// SelectEditor returns an explicit app name, otherwise the first installed candidate.
func SelectEditor(override string, exists func(string) bool) string {
	if app := strings.TrimSpace(override); app != "" {
		return app
	}
	if exists == nil {
		return "TextEdit"
	}
	for _, name := range editorCandidates {
		if exists(name) {
			return name
		}
	}
	return "TextEdit"
}

// AppExists reports whether an app bundle is installed for the current user.
func AppExists(name string) bool {
	dirs := []string{"/Applications", "/System/Applications"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Applications"))
	}
	for _, dir := range dirs {
		if _, err := os.Stat(filepath.Join(dir, name+".app")); err == nil {
			return true
		}
	}
	return false
}

// OpenApp opens path with a macOS application.
func OpenApp(app, path string) error {
	cmd := exec.Command("/usr/bin/open", "-a", app, path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("无法用 %s 打开: %s", app, msg)
	}
	return nil
}

// Alert shows a modal warning. Used when the clipboard cannot be formatted.
func Alert(message string) error {
	script := `on run argv
display alert "无法格式化 JSON" message (item 1 of argv) as warning
end run`
	cmd := exec.Command("/usr/bin/osascript", "-e", script, message)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}
