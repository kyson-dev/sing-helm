package main

import (
	"os"

	"github.com/kyson/minibox/internal/cli"
	_ "github.com/sagernet/sing-box/include"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
