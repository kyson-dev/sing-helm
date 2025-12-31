package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kyson/minibox/internal/env"
	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Open configuration file in editor",
		Long: `Open the user configuration file (profile.json) in your default editor.

The editor is determined by (in order):
  1. VISUAL environment variable
  2. EDITOR environment variable  
  3. vi (fallback)

The configuration file contains your proxy nodes and settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := env.Get().ConfigFile

			// 如果配置文件不存在，提示用户
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				fmt.Printf("Configuration file not found: %s\n", configPath)
				fmt.Println("It will be created when you save in the editor.")
			}

			// 获取编辑器
			editor := os.Getenv("VISUAL")
			if editor == "" {
				editor = os.Getenv("EDITOR")
			}
			if editor == "" {
				editor = "vi" // 兜底
			}

			fmt.Printf("Opening: %s\n", configPath)
			fmt.Printf("Editor:  %s\n\n", editor)

			// 打开编辑器
			editorCmd := exec.Command(editor, configPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}

			return nil
		},
	}
}
