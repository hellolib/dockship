package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Client Docker客户端
type Client struct {
	tempDir string // 临时文件目录
}

// NewClient 创建Docker客户端
func NewClient(tempDir string) *Client {
	return &Client{
		tempDir: tempDir,
	}
}

// CheckImageExists 检查镜像是否存在于本地
func (c *Client) CheckImageExists(image string) (bool, error) {
	cmd := exec.Command("docker", "images", "-q", image)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("检查镜像失败: %w", err)
	}

	// 如果输出不为空，说明镜像存在
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// PullImage 从远程仓库拉取镜像
func (c *Client) PullImage(image string) error {
	fmt.Printf("📥 正在拉取镜像: %s\n", image)

	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("拉取镜像失败: %w", err)
	}

	fmt.Printf("✅ 镜像拉取成功: %s\n", image)
	return nil
}

// EnsureImageExists 确保镜像存在，不存在则拉取
func (c *Client) EnsureImageExists(image string) error {
	exists, err := c.CheckImageExists(image)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("✅ 镜像已存在于本地: %s\n", image)
		return nil
	}

	fmt.Printf("⚠️  镜像不存在于本地，开始拉取: %s\n", image)
	return c.PullImage(image)
}

// SaveImage 将镜像保存为tar文件
func (c *Client) SaveImage(image string) (string, error) {
	// 确保临时目录存在
	if err := os.MkdirAll(c.tempDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 生成tar文件名（将镜像名中的特殊字符替换）
	imageName := strings.ReplaceAll(image, "/", "_")
	imageName = strings.ReplaceAll(imageName, ":", "_")
	tarFile := filepath.Join(c.tempDir, imageName+".tar")

	fmt.Printf("📦 正在保存镜像: %s -> %s\n", image, tarFile)

	cmd := exec.Command("docker", "save", "-o", tarFile, image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("保存镜像失败: %w", err)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(tarFile)
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	fmt.Printf("✅ 镜像保存成功: %s (%.2f MB)\n", tarFile, float64(fileInfo.Size())/1024/1024)
	return tarFile, nil
}

// PrepareImage 准备镜像（确保存在 + 保存为tar）
func (c *Client) PrepareImage(image string) (string, error) {
	// 1. 确保镜像存在
	if err := c.EnsureImageExists(image); err != nil {
		return "", err
	}

	// 2. 保存镜像为tar文件
	return c.SaveImage(image)
}

// CleanupTarFile 清理tar文件
func (c *Client) CleanupTarFile(tarFile string) error {
	if err := os.Remove(tarFile); err != nil {
		return fmt.Errorf("清理tar文件失败: %w", err)
	}
	fmt.Printf("🧹 已清理临时文件: %s\n", tarFile)
	return nil
}

// CheckDockerAvailable 检查Docker是否可用
func CheckDockerAvailable() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker不可用，请确保Docker已安装并正在运行: %w", err)
	}
	return nil
}
