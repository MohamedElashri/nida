package main

import (
	"os"

	"github.com/MohamedElashri/nida/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
