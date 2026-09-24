package main

import (
	"fmt"
	"os"

	"myalfredworkflow/internal/jsonfmt"
)

func main() {
	msg, err := run()
	if err != nil {
		if alertErr := jsonfmt.Alert(err.Error()); alertErr != nil {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println(msg)
}

func run() (string, error) {
	raw, ok := os.LookupEnv("clipboard_json")
	if !ok {
		return "", fmt.Errorf("没有拿到剪贴板内容")
	}
	res, err := jsonfmt.Run([]byte(raw), jsonfmt.Config{Editor: os.Getenv("json_editor")})
	if err != nil {
		return "", err
	}
	return res.Message, nil
}
