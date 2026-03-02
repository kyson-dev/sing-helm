package main

import (
  "context"
  "fmt"
  "github.com/kyson-dev/sing-helm/internal/proxy/config"
)

func main() {
  _, err := config.LoadOptionsWithContext(context.Background(), "bin/singbox-config-1.11.4-ios.json")
  if err != nil {
    panic(err)
  }
  fmt.Println("ok")
}
