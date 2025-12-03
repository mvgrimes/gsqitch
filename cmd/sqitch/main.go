package main

import (
	"fmt"
	"os"

	"github.com/sqitchers/sqitch-go/internal/command"
)

func main() {
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
