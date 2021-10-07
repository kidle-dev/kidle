package main

import (
	"encoding/json"
	"fmt"

	"github.com/kidle-dev/kidle/pkg/version"
)

// cmdVersion executes the kidlectl version command
func cmdVersion() {
	b, err := json.Marshal(version.GetVersionInfos())
	if err != nil {
		fmt.Println(version.Version)
		return
	}
	fmt.Println(string(b))
}
