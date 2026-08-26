package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sqitchers/sqitch-go/internal/command"
)

var version = "1.0.1"

func main() {
	versionInfo := fmt.Sprintf("gsqitch (golang) v%s", strings.TrimPrefix(version, "v"))
	if err := command.Execute(versionInfo); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
