package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"myalfredworkflow/internal/timefmt"
)

func main() {
	query := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(timefmt.ScriptFilter(time.Now(), query)); err != nil {
		fmt.Fprintf(os.Stderr, "encode script filter: %v\n", err)
		os.Exit(1)
	}
}
