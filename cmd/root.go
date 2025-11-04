package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// 全局配置文件路径
	cfgFile string
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "dockship",
	Short: "🚢 Dockship - Docker镜像分发工具",
	Long: `Dockship 是一个轻量级 Docker 镜像分发工具。

用于在没有镜像仓库（registry）的环境下，高效地将本地或远程镜像
传输到多台目标主机，并在远端自动执行 docker load。

支持的功能：
  • 镜像自动获取（本地/远程）
  • 多主机并发分发
  • SSH安全传输
  • 自动加载镜像
  • 失败重试机制`,
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// 定义全局标志
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "配置文件路径")
}

// GetConfigFile 获取配置文件路径
func GetConfigFile() string {
	return cfgFile
}
