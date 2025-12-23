package main

import "github.com/kyson/minibox/internal/adapter/cli"

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		panic(err)
	}
}
