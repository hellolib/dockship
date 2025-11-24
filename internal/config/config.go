package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Images        []string       `mapstructure:"images"`         // 需要传输的镜像列表
	TargetHosts   []string       `mapstructure:"target_hosts"`   // 目标主机列表
	SSH           SSHConfig      `mapstructure:"ssh"`            // SSH连接配置
	LocalStorage  StorageConfig  `mapstructure:"local_storage"`  // 本地存储配置
	RemoteStorage StorageConfig  `mapstructure:"remote_storage"` // 远程存储配置
	Transfer      TransferConfig `mapstructure:"transfer"`       // 传输配置
	Hooks         HooksConfig    `mapstructure:"hooks"`          // Hooks配置
}

// SSHConfig SSH连接配置
type SSHConfig struct {
	User     string `mapstructure:"user"`     // SSH用户名
	Password string `mapstructure:"pwd"`      // SSH密码（不推荐）
	KeyFile  string `mapstructure:"key_file"` // SSH私钥文件路径（推荐）
	Port     int    `mapstructure:"port"`     // SSH端口
	Timeout  int    `mapstructure:"timeout"`  // 连接超时时间（秒）
}

// StorageConfig 本地存储配置
type StorageConfig struct {
	TempDir     string `mapstructure:"temp_dir"`     // 本地临时文件目录
	AutoCleanup bool   `mapstructure:"auto_cleanup"` // 传输完成后是否自动清理本地临时文件
}

// TransferConfig 传输配置
type TransferConfig struct {
	Concurrent int `mapstructure:"concurrent"` // 并发传输主机数量，也用于镜像并发
	Retry      int `mapstructure:"retry"`      // 失败重试次数
}

// HooksConfig Hooks配置
type HooksConfig struct {
	PreLoad  []string `mapstructure:"pre_load"`  // 镜像加载前执行的命令列表
	PostLoad []string `mapstructure:"post_load"` // 镜像加载后执行的命令列表
}

var globalConfig *Config

// LoadConfig 从配置文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	globalConfig = &cfg
	return &cfg, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	viper.SetDefault("ssh.port", 22)
	viper.SetDefault("ssh.timeout", 30)
	viper.SetDefault("local_storage.temp_dir", "/tmp/dockship")
	viper.SetDefault("local_storage.auto_cleanup", true)
	viper.SetDefault("remote_storage.temp_dir", "/tmp")
	viper.SetDefault("remote_storage.auto_cleanup", true)
	viper.SetDefault("transfer.concurrent", 5)
	viper.SetDefault("transfer.retry", 3)
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	if len(c.Images) == 0 {
		return fmt.Errorf("镜像列表不能为空")
	}

	if len(c.TargetHosts) == 0 {
		return fmt.Errorf("目标主机列表不能为空")
	}

	if c.SSH.User == "" {
		return fmt.Errorf("SSH用户名不能为空")
	}

	// 必须提供密码或密钥文件之一
	if c.SSH.Password == "" && c.SSH.KeyFile == "" {
		return fmt.Errorf("必须提供SSH密码或密钥文件")
	}

	// 如果指定了密钥文件，检查文件是否存在
	if c.SSH.KeyFile != "" {
		if _, err := os.Stat(c.SSH.KeyFile); err != nil {
			return fmt.Errorf("SSH密钥文件不存在: %s", c.SSH.KeyFile)
		}
	}

	if c.SSH.Port <= 0 || c.SSH.Port > 65535 {
		return fmt.Errorf("SSH端口无效: %d", c.SSH.Port)
	}

	if c.Transfer.Concurrent <= 0 {
		c.Transfer.Concurrent = 1
	}

	return nil
}

// GetConfig 获取全局配置实例
func GetConfig() *Config {
	return globalConfig
}
