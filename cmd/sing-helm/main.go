package main

import (
	"os"

	"github.com/kysonzou/sing-helm/internal/cli"
	_ "github.com/sagernet/sing-box/include"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
