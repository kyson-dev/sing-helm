package main

import (
  "encoding/json"
  "fmt"

  "github.com/kyson-dev/sing-helm/internal/proxy/config"
  "github.com/kyson-dev/sing-helm/internal/proxy/config/export"
  "github.com/kyson-dev/sing-helm/internal/proxy/config/model"
  "github.com/kyson-dev/sing-helm/internal/sys/paths"
)

func main() {
  _ = paths.Init("")
  runops := model.DefaultRunOptions()
  runops.ProxyMode = model.ProxyModeTUN
  runops.RouteMode = model.RouteModeRule
  runops.APIPort = 9090
  runops.MixedPort = 7890
  opts, err := config.BuildOptions(&runops)
  if err != nil { panic(err) }
  data, err := export.Export(opts, export.Target{Version: "1.11.4", Platform: "ios"})
  if err != nil { panic(err) }

  var root map[string]any
  if err := json.Unmarshal(data, &root); err != nil { panic(err) }

  route, _ := root["route"].(map[string]any)
  fmt.Println("route.final=", route["final"])

  outbounds, _ := root["outbounds"].([]any)
  for _, o := range outbounds {
    ob, _ := o.(map[string]any)
    tag, _ := ob["tag"].(string)
    if tag == "proxy" || tag == "auto" || tag == "direct" || tag == "block" {
      b, _ := json.Marshal(ob)
      fmt.Println(string(b))
    }
  }
}
