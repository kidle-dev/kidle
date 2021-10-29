package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/kidle-dev/kidle/pkg/version"
)

// VersionCommandOptions are the options of the version command
type VersionCommandOptions struct {
}

// Version executes the kidlectl version command
func Version() {
	b, err := json.Marshal(version.GetVersionInfos())
	if err != nil {
		fmt.Println(version.Version)
		return
	}
	fmt.Println(string(b))
}
