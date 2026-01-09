package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kysonzou/sing-helm/internal/logger"
	"github.com/kysonzou/sing-helm/internal/client"
	"github.com/spf13/cobra"
)

var apiAddr string // 复用 flag

func newNodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage proxy nodes",
		// 注意：不要定义 PersistentPreRun，否则会覆盖 root 的 PersistentPreRun
		// root 的 PersistentPreRun 会调用 env.Init() 和 logger.Setup()
	}

	// 注册子命令
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newUseCommand())

	// 定义 PersistentFlag，让子命令都能用到
	cmd.PersistentFlags().StringVar(&apiAddr, "api", "", "API address")

	return cmd
}

// 1. 实现 List 命令
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all proxy groups and nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload := map[string]any{}
			if apiAddr != "" {
				payload["api"] = apiAddr
			}
			resp, err := dispatchToDaemon(cmd.Context(), "node.list", payload)
			if err != nil {
				return fmt.Errorf("failed to list proxies: %w (tip: is sing-helm running?)", err)
			}

			proxies, err := decodeProxies(resp.Data["proxies"])
			if err != nil {
				return fmt.Errorf("failed to decode proxies: %w", err)
			}

			// 简单的美化输出
			fmt.Printf("%-20s %-15s %s\n", "GROUP", "TYPE", "CURRENT / NODES")
			fmt.Println(strings.Repeat("-", 60))

			// 为了输出稳定，我们需要对 Map Key 排序
			var keys []string
			for k := range proxies {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, name := range keys {
				p := proxies[name]
				// 我们只关心 Selector 类型的组，因为它们可以切换
				if p.Type == "Selector" {
					fmt.Printf("%-20s %-15s \033[32m%s\033[0m\n", name, p.Type, p.Now)
					// 可选：打印该组下所有可选节点 (缩进显示)
					for _, node := range p.All {
						mark := " "
						if node == p.Now {
							mark = "*"
						}
						fmt.Printf("  %s %s\n", mark, node)
					}
				}
			}
			return nil
		},
	}
}

// 2. 实现 Use 命令
func newUseCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "use [group] [node]",
		Short:   "Switch node for a selector group",
		Example: "  sing-helm node use Proxy 'HongKong 01'",
		Args:    cobra.ExactArgs(2), // 必须传 2 个参数
		RunE: func(cmd *cobra.Command, args []string) error {
			group := args[0]
			node := args[1]

			payload := map[string]any{
				"group": group,
				"node":  node,
			}
			if apiAddr != "" {
				payload["api"] = apiAddr
			}
			if _, err := dispatchToDaemon(cmd.Context(), "node.use", payload); err != nil {
				return fmt.Errorf("failed to switch node: %w", err)
			}

			logger.Info("Switched successfully", "group", group, "node", node)
			return nil
		},
	}
}

func decodeProxies(raw any) (map[string]client.ProxyData, error) {
	if raw == nil {
		return nil, fmt.Errorf("missing proxies data")
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var proxies map[string]client.ProxyData
	if err := json.Unmarshal(data, &proxies); err != nil {
		return nil, err
	}
	return proxies, nil
}
