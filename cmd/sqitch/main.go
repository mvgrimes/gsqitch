package main

import (
	"fmt"
	"os"

	"github.com/sqitchers/sqitch-go/internal/command"
)

var version = "0.1.2"

func main() {
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
