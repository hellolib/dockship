package cmd

import (
	"fmt"
	"os"

	"dockship/internal/config"
	"dockship/internal/transfer"

	"github.com/spf13/cobra"
)

var skipConfirm bool

// transferCmd 传输命令
var transferCmd = &cobra.Command{
	Use:     "transfer",
	Aliases: []string{"go"},
	Short:   "传输Docker镜像到目标主机",
	Long: `读取配置文件中的镜像列表和目标主机列表，
自动执行以下操作：
  1. 检查本地镜像，不存在则从远程拉取
  2. 使用 docker save 保存镜像为 tar 文件
  3. 通过 SSH 传输到目标主机
  4. 在远程主机执行 docker load 加载镜像
  5. 清理临时文件

示例：
  dockship transfer                    # 使用默认配置文件 config.yaml
  dockship go                          # 等同于 transfer，更简短的别名
  dockship transfer -c custom.yaml     # 使用自定义配置文件
  dockship transfer -y                 # 跳过二次确认
  dockship go -c custom.yaml     # 使用自定义配置文件`,
	RunE: runTransfer,
}

func init() {
	// 将传输命令添加到根命令
	rootCmd.AddCommand(transferCmd)
	transferCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "跳过二次确认，直接执行")
}

// runTransfer 执行传输任务
func runTransfer(cmd *cobra.Command, args []string) error {
	// 1. 加载配置文件
	fmt.Printf("📝 加载配置文件: %s\n", GetConfigFile())
	cfg, err := config.LoadConfig(GetConfigFile())
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 2. 显示配置信息
	printConfigInfo(cfg, skipConfirm)

	// 3. 创建传输管理器
	manager := transfer.NewManager(cfg)

	// 4. 开始传输
	if err := manager.Start(); err != nil {
		return fmt.Errorf("传输任务失败: %w", err)
	}

	return nil
}

// printConfigInfo 打印配置信息
func printConfigInfo(cfg *config.Config, skipConfirm bool) {
	fmt.Println("\n📋 配置信息：")
	fmt.Printf("  镜像数量: %d\n", len(cfg.Images))
	for i, imageCfg := range cfg.Images {
		fmt.Printf("    %d. %s\n", i+1, imageCfg.Name)
	}
	fmt.Printf("  目标主机: %d 台\n", len(cfg.TargetHosts))
	for i, target := range cfg.TargetHosts {
		fmt.Printf("    %d. %s\n", i+1, target)
	}
	fmt.Printf("  并发数: %d\n", cfg.Transfer.Concurrent)
	fmt.Printf("  重试次数: %d\n", cfg.Transfer.Retry)
	fmt.Printf("  自动加载镜像: %v\n", cfg.Transfer.AutoLoad)
	fmt.Printf("  SSH用户: %s\n", cfg.SSH.User)
	fmt.Printf("  SSH端口: %d\n", cfg.SSH.Port)

	// 检查是否使用密钥认证
	authMethod := "密码"
	if cfg.SSH.KeyFile != "" {
		authMethod = fmt.Sprintf("密钥 (%s)", cfg.SSH.KeyFile)
	}
	fmt.Printf("  认证方式: %s\n", authMethod)

	// 确认执行（配置开启确认且未指定 -y 时才询问）
	if cfg.Transfer.Confirm && !skipConfirm {
		fmt.Print("\n⚠️  确认要继续执行吗? [y/N]: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("❌ 任务已取消")
			os.Exit(0)
		}
	}
	fmt.Println()
}
