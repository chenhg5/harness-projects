package main

import (
	"os"

	"github.com/chenhg5/harness-projects/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
