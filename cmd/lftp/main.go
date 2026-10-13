package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"myalfredworkflow/internal/lftpbm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: lftp filter|open [query]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "filter":
		if err := filter(strings.TrimSpace(strings.Join(os.Args[2:], " "))); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "open":
		fmt.Println(openMessage(strings.TrimSpace(strings.Join(os.Args[2:], " "))))
	default:
		fmt.Fprintf(os.Stderr, "unknown command %s\n", os.Args[1])
		os.Exit(2)
	}
}

func filter(query string) error {
	marks, err := lftpbm.List()
	out := lftpbm.ScriptFilter(marks, query)
	if err != nil {
		out = lftpbm.MessageOutput("无法读取 bookmark", err.Error())
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}

func openMessage(name string) string {
	msg, err := lftpbm.Open(name, lftpbm.Deps{
		Resolve:  lftpbm.FindLFTP,
		List:     lftpbm.List,
		Run:      lftpbm.RunInTerminal,
		Terminal: os.Getenv("lftp_terminal"),
	})
	if err != nil {
		return err.Error()
	}
	return msg
}
