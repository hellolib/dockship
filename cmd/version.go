package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// 版本信息（可以通过 ldflags 在编译时注入）
var (
	Version   = "v1.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GoVersion = runtime.Version()
)

// versionCmd 版本命令
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  "显示 Dockship 的版本、构建时间、Git commit 等信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🚢 Dockship %s\n", Version)
		fmt.Println("=====================================")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Go Version: %s\n", GoVersion)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	// 将版本命令添加到根命令
	rootCmd.AddCommand(versionCmd)
}
