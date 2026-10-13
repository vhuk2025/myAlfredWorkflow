package lftpbm

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Bookmark is one lftp bookmark alias.
// URL is whatever `lftp -c "bookmark list"` prints, which masks passwords.
type Bookmark struct {
	Name string
	URL  string
}

// Item is one Alfred Script Filter result.
type Item struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg,omitempty"`
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

// Deps wires bookmark lookup and the terminal launcher so Open can be tested.
type Deps struct {
	Resolve  func() (string, error)
	List     func() ([]Bookmark, error)
	Run      func(app, command string) error
	Terminal string
}

var userPass = regexp.MustCompile(`(://[^/:@\s]+):[^@/\s]+@`)

var binaryCandidates = []string{
	"/opt/homebrew/bin/lftp",
	"/usr/local/bin/lftp",
	"/usr/bin/lftp",
}

// ParseList reads the tabular output of `lftp -c "bookmark list"`.
func ParseList(text string) []Bookmark {
	var marks []Bookmark
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		name, rawURL, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		rawURL = strings.TrimSpace(rawURL)
		if name == "" {
			continue
		}
		marks = append(marks, Bookmark{Name: name, URL: rawURL})
	}
	return marks
}

// Target is a bookmark location safe to show in Alfred. User and password are removed.
func (b Bookmark) Target() string {
	raw := strings.TrimSpace(b.URL)
	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		scheme := parsed.Scheme
		if scheme == "" {
			scheme = "ftp"
		}
		path := parsed.EscapedPath()
		if path == "" {
			path = "/"
		}
		return scheme + "://" + parsed.Host + path
	}
	return userPass.ReplaceAllString(raw, "$1@")
}

// ScriptFilter lists bookmark aliases, optionally filtered by query.
func ScriptFilter(marks []Bookmark, query string) Output {
	items := make([]Item, 0, len(marks))
	for _, mark := range marks {
		if !matches(mark, query) {
			continue
		}
		subtitle := "回车在终端打开"
		if target := mark.Target(); target != "" {
			subtitle = target + " · " + subtitle
		}
		items = append(items, Item{
			Title:    mark.Name,
			Subtitle: subtitle,
			Arg:      mark.Name,
			Valid:    true,
			Icon:     &Icon{Path: "icon.png"},
		})
	}
	if len(items) > 0 {
		return Output{Items: items}
	}
	title := "没有 lftp bookmark"
	subtitle := "先用 bookmark add 添加别名"
	if strings.TrimSpace(query) != "" {
		title = "没有匹配的 bookmark"
		subtitle = "换个关键字试试"
	}
	return Output{Items: []Item{{
		Title:    title,
		Subtitle: subtitle,
		Valid:    false,
		Icon:     &Icon{Path: "icon.png"},
	}}}
}

// MessageOutput is a single non-actionable Alfred row.
func MessageOutput(title, subtitle string) Output {
	return Output{Items: []Item{{
		Title:    title,
		Subtitle: subtitle,
		Valid:    false,
		Icon:     &Icon{Path: "icon.png"},
	}}}
}

// Open checks the alias, then runs `lftp <alias>` in Terminal or iTerm.
func Open(name string, deps Deps) (string, error) {
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return "", err
	}
	if deps.List == nil || deps.Resolve == nil || deps.Run == nil {
		return "", fmt.Errorf("lftp 未配置")
	}
	marks, err := deps.List()
	if err != nil {
		return "", err
	}
	if !hasName(marks, name) {
		return "", fmt.Errorf("没有 bookmark：%s", name)
	}
	bin, err := deps.Resolve()
	if err != nil {
		return "", err
	}
	if err := deps.Run(TerminalApp(deps.Terminal), shellQuote(bin)+" "+shellQuote(name)); err != nil {
		return "", err
	}
	return "已在终端打开 " + name, nil
}

// List runs `lftp -c "bookmark list"` and parses aliases. It does not read the bookmarks file.
func List() ([]Bookmark, error) {
	bin, err := FindLFTP()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(bin, "-c", "bookmark list")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("无法读取 lftp bookmark：%s", commandError(err))
	}
	return ParseList(string(out)), nil
}

// FindLFTP locates the lftp binary. Alfred's PATH often misses Homebrew.
func FindLFTP() (string, error) {
	return ResolveBinary(exec.LookPath, executableFile)
}

// ResolveBinary prefers PATH, then common install locations.
func ResolveBinary(lookPath func(string) (string, error), exists func(string) bool) (string, error) {
	if lookPath != nil {
		if path, err := lookPath("lftp"); err == nil && path != "" {
			return path, nil
		}
	}
	if exists != nil {
		for _, candidate := range binaryCandidates {
			if exists(candidate) {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("找不到 lftp")
}

// TerminalApp returns the configured terminal, or Terminal when unset.
func TerminalApp(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Terminal"
	}
	return name
}

// RunInTerminal opens a new terminal window and runs command.
// Terminal, iTerm, and Ghostty each get their own script. A combined script
// fails to compile when the unused terminal is not installed.
func RunInTerminal(app, command string) error {
	script, args := launchScript(app, command)
	argv := make([]string, 0, len(args)+2)
	argv = append(argv, "-e", script)
	argv = append(argv, args...)
	cmd := exec.Command("/usr/bin/osascript", argv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("无法打开终端：%s", msg)
	}
	return nil
}

func launchScript(app, command string) (string, []string) {
	switch terminalKind(app) {
	case kindGhostty:
		// Ghostty already runs `bash -c "exec -l <command>"`.
		// A shell: prefix is only for its config file and becomes part of the path.
		return ghosttyLaunchScript, []string{command}
	case kindITerm:
		return itermLaunchScript, []string{command}
	default:
		return terminalLaunchScript, []string{command}
	}
}

type kind int

const (
	kindTerminal kind = iota
	kindITerm
	kindGhostty
)

func terminalKind(app string) kind {
	name := strings.ToLower(strings.TrimSpace(app))
	switch {
	case strings.Contains(name, "ghostty"):
		return kindGhostty
	case strings.Contains(name, "iterm"):
		return kindITerm
	default:
		return kindTerminal
	}
}

const terminalLaunchScript = `on run argv
set cmd to item 1 of argv
tell application "Terminal"
activate
do script cmd
end tell
end run`

const itermLaunchScript = `on run argv
set cmd to item 1 of argv
tell application "iTerm"
activate
set w to (create window with default profile)
tell current session of w to write text cmd
end tell
end run`

const ghosttyLaunchScript = `on run argv
set cmd to item 1 of argv
tell application "Ghostty"
activate
set cfg to new surface configuration
set command of cfg to cmd
set wait after command of cfg to true
if (count of windows) > 0 then
new tab in front window with configuration cfg
else
new window with configuration cfg
end if
end tell
end run`

func matches(mark Bookmark, query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}
	haystack := strings.ToLower(mark.Name + " " + mark.URL + " " + mark.Target())
	for _, part := range strings.Fields(query) {
		if !strings.Contains(haystack, strings.ToLower(part)) {
			return false
		}
	}
	return true
}

func hasName(marks []Bookmark, name string) bool {
	for _, mark := range marks {
		if mark.Name == name {
			return true
		}
	}
	return false
}

func validateName(name string) error {
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\r\n\x00") {
		return fmt.Errorf("无效的 bookmark：%s", name)
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func executableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func commandError(err error) string {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if msg := strings.TrimSpace(string(exitErr.Stderr)); msg != "" {
			return msg
		}
	}
	return strings.TrimSpace(err.Error())
}
