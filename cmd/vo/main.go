package main

import (
	"os"

	"voyage/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
