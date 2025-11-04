package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"golang.org/x/crypto/ssh"
)

// Client SSH客户端
type Client struct {
	host       string
	port       int
	user       string
	password   string
	keyFile    string
	timeout    time.Duration
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	progress   *mpb.Progress // 多进度条容器
}

// NewClient 创建SSH客户端
func NewClient(host string, port int, user, password, keyFile string, timeout int, progress *mpb.Progress) *Client {
	return &Client{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		keyFile:  keyFile,
		timeout:  time.Duration(timeout) * time.Second,
		progress: progress,
	}
}

// Connect 连接到SSH服务器
func (c *Client) Connect() error {
	// 构建SSH配置
	config := &ssh.ClientConfig{
		User:            c.user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证主机密钥
		Timeout:         c.timeout,
	}

	// 添加认证方式
	if c.keyFile != "" {
		// 使用密钥认证
		key, err := os.ReadFile(c.keyFile)
		if err != nil {
			return fmt.Errorf("读取SSH密钥文件失败: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("解析SSH密钥失败: %w", err)
		}

		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if c.password != "" {
		// 使用密码认证
		config.Auth = []ssh.AuthMethod{ssh.Password(c.password)}
	}

	// 连接SSH服务器
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("连接SSH服务器失败 [%s]: %w", addr, err)
	}

	c.sshClient = sshClient

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}

	c.sftpClient = sftpClient
	return nil
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.sftpClient != nil {
		c.sftpClient.Close()
	}
	if c.sshClient != nil {
		return c.sshClient.Close()
	}
	return nil
}

// UploadFile 上传文件到远程服务器
func (c *Client) UploadFile(localPath, remotePath string) error {
	// 打开本地文件
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer localFile.Close()

	// 获取文件信息
	fileInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 确保远程目录存在
	remoteDir := filepath.Dir(remotePath)
	if err := c.sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("创建远程目录失败: %w", err)
	}

	// 创建远程文件
	remoteFile, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	// 创建进度条
	bar := c.progress.AddBar(fileInfo.Size(),
		mpb.BarRemoveOnComplete(),
		mpb.PrependDecorators(
			decor.Name(fmt.Sprintf("📤 [%s]", c.host), decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.CountersKibiByte("%.1f / %.1f"),
			decor.NewPercentage("%d"),
			decor.AverageSpeed(decor.SizeB1024(0), " %.1f/s"),
		),
	)

	// 使用缓冲区分块传输并更新进度
	buffer := make([]byte, 32*1024) // 32KB 缓冲区
	var written int64

	for {
		nr, errRead := localFile.Read(buffer)
		if nr > 0 {
			nw, errWrite := remoteFile.Write(buffer[0:nr])
			if nw > 0 {
				written += int64(nw)
				bar.SetCurrent(written) // 手动更新进度
			}
			if errWrite != nil {
				bar.Abort(false)
				return fmt.Errorf("写入远程文件失败: %w", errWrite)
			}
			if nr != nw {
				bar.Abort(false)
				return fmt.Errorf("写入数据不完整")
			}
		}
		if errRead == io.EOF {
			break
		}
		if errRead != nil {
			bar.Abort(false)
			return fmt.Errorf("读取本地文件失败: %w", errRead)
		}
	}

	if written != fileInfo.Size() {
		bar.Abort(false)
		return fmt.Errorf("文件上传不完整: 期望 %d 字节，实际 %d 字节", fileInfo.Size(), written)
	}

	// 标记进度条完成并清除
	bar.SetCurrent(fileInfo.Size())
	bar.EnableTriggerComplete()

	return nil
}

// ExecuteCommand 执行远程命令
func (c *Client) ExecuteCommand(command string) (string, error) {
	// 创建会话
	session, err := c.sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 执行命令
	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("执行远程命令失败: %w", err)
	}

	return string(output), nil
}

// LoadDockerImage 在远程主机上加载Docker镜像
func (c *Client) LoadDockerImage(remoteTarPath string) error {
	command := fmt.Sprintf("docker load -i %s", remoteTarPath)
	output, err := c.ExecuteCommand(command)
	if err != nil {
		return fmt.Errorf("加载Docker镜像失败: %w\n输出: %s", err, output)
	}
	return nil
}

// RemoveRemoteFile 删除远程文件
func (c *Client) RemoveRemoteFile(remotePath string) error {
	if err := c.sftpClient.Remove(remotePath); err != nil {
		return fmt.Errorf("删除远程文件失败: %w", err)
	}
	return nil
}

// CheckDockerAvailable 检查远程主机的Docker是否可用
func (c *Client) CheckDockerAvailable() error {
	_, err := c.ExecuteCommand("docker version")
	if err != nil {
		return fmt.Errorf("远程主机Docker不可用: %w", err)
	}
	return nil
}

// ExecuteHooks 执行hooks命令列表
// stage: 执行阶段名称（pre_load/post_load），用于日志输出
// commands: 要执行的命令列表
// 返回：是否有命令执行失败
func (c *Client) ExecuteHooks(stage string, commands []string) bool {
	if len(commands) == 0 {
		return true // 没有命令，视为成功
	}

	fmt.Printf("  🔧 [%s] 执行 %s hooks...\n", c.host, stage)
	hasError := false

	for i, command := range commands {
		fmt.Printf("    [%s][%d/%d] 执行: %s\n", c.host, i+1, len(commands), command)

		output, err := c.ExecuteCommand(command)
		if err != nil {
			hasError = true
			fmt.Printf("    [%s] ❌ 失败: %v\n", c.host, err)
			if output != "" {
				fmt.Printf("    [%s] 输出: %s\n", c.host, output)
			}
			// 继续执行下一个命令
			continue
		}

		fmt.Printf("    [%s] ✅ 成功\n", c.host)
		// 显示命令输出
		if output != "" {
			fmt.Printf("    [%s] 输出: %s\n", c.host, output)
		}
	}

	if hasError {
		fmt.Printf("  [%s] ⚠️  %s hooks 执行完成（部分失败）\n", c.host, stage)
	} else {
		fmt.Printf("  [%s] ✅ %s hooks 执行成功\n", c.host, stage)
	}

	return !hasError
}
