package transfer

import (
	"dockship/internal/config"
	"dockship/internal/docker"
	"dockship/internal/ssh"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v8"
)

// Manager 传输管理器
type Manager struct {
	cfg          *config.Config
	dockerClient *docker.Client
}

// NewManager 创建传输管理器
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:          cfg,
		dockerClient: docker.NewClient(cfg.LocalStorage.TempDir),
	}
}

// TransferResult 传输结果
type TransferResult struct {
	Host    string // 目标主机
	Image   string // 镜像名称
	Success bool   // 是否成功
	Error   error  // 错误信息
}

type preparedImage struct {
	ImageCfg config.ImageConfig
	TarFile  string
	Err      error
}

type imagePipelineResult struct {
	Image string
	Err   error
}

// Start 启动传输任务
func (m *Manager) Start() error {
	fmt.Println("🚀 Dockship 开始执行镜像传输任务")
	fmt.Println(strings.Repeat("=", 60))

	// 检查本地Docker是否可用
	if err := docker.CheckDockerAvailable(); err != nil {
		return err
	}

	startTime := time.Now()

	imageCount := len(m.cfg.Images)
	if imageCount == 0 {
		return nil
	}

	concurrency := m.cfg.Transfer.Concurrent
	if concurrency <= 0 {
		concurrency = 1
	}

	imageCh := make(chan config.ImageConfig)
	preparedCh := make(chan preparedImage, imageCount)
	resultCh := make(chan imagePipelineResult, imageCount)

	var prepareWg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		prepareWg.Add(1)
		go func() {
			defer prepareWg.Done()
			for imageCfg := range imageCh {
				tarFile, err := m.dockerClient.PrepareImage(imageCfg.Name)
				preparedCh <- preparedImage{
					ImageCfg: imageCfg,
					TarFile:  tarFile,
					Err:      err,
				}
			}
		}()
	}

	go func() {
		prepareWg.Wait()
		close(preparedCh)
	}()

	go func() {
		for _, imageCfg := range m.cfg.Images {
			imageCh <- imageCfg
		}
		close(imageCh)
	}()

	var transferWg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		transferWg.Add(1)
		go func() {
			defer transferWg.Done()
			for prepared := range preparedCh {
				err := m.handlePreparedImage(prepared)
				resultCh <- imagePipelineResult{
					Image: prepared.ImageCfg.Name,
					Err:   err,
				}
			}
		}()
	}

	go func() {
		transferWg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		if res.Err != nil {
			fmt.Printf("❌ 镜像 %s 处理失败: %v\n", res.Image, res.Err)
		}
	}

	elapsed := time.Since(startTime)
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("✅ 所有任务完成，总耗时: %.2f 秒\n", elapsed.Seconds())

	return nil
}

func (m *Manager) handlePreparedImage(prepared preparedImage) error {
	if prepared.Err != nil {
		return prepared.Err
	}

	if prepared.TarFile == "" {
		return fmt.Errorf("镜像 %s 的 tar 文件不存在", prepared.ImageCfg.Name)
	}

	if m.cfg.LocalStorage.AutoCleanup {
		defer func(tarPath string) {
			if err := m.dockerClient.CleanupTarFile(tarPath); err != nil {
				fmt.Printf("⚠️  清理tar文件失败: %v\n", err)
			}
		}(prepared.TarFile)
	}

	return m.transferPreparedImage(prepared.ImageCfg, prepared.TarFile)
}

func (m *Manager) transferPreparedImage(imageCfg config.ImageConfig, tarFile string) error {
	fmt.Printf("\n📦 处理镜像: %s\n", imageCfg.Name)
	fmt.Println(strings.Repeat("-", 60))

	progress := mpb.New(
		mpb.WithRefreshRate(120 * time.Millisecond),
	)

	results := m.transferToHosts(imageCfg, tarFile, progress)

	progress.Wait()

	fmt.Println()
	for _, result := range results {
		if result.Success {
			fmt.Printf("  ✅ [%s] 镜像传输完成\n", result.Host)
		} else {
			fmt.Printf("  ❌ [%s] 失败: %v\n", result.Host, result.Error)
		}
	}

	success := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			success++
		} else {
			failed++
		}
	}

	fmt.Printf("\n📊 镜像 %s 传输统计: 成功 %d 台，失败 %d 台\n", imageCfg.Name, success, failed)
	return nil
}

// transferToHosts 并发传输到多个主机
func (m *Manager) transferToHosts(imageCfg config.ImageConfig, tarFile string, progress *mpb.Progress) []TransferResult {
	var wg sync.WaitGroup
	results := make([]TransferResult, len(m.cfg.TargetHosts))

	// 创建信号量控制并发数
	semaphore := make(chan struct{}, m.cfg.Transfer.Concurrent)

	for i, host := range m.cfg.TargetHosts {
		wg.Add(1)

		go func(index int, targetHost string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 执行传输
			result := m.transferToHost(targetHost, imageCfg, tarFile, progress)
			results[index] = result
		}(i, host)
	}

	wg.Wait()
	return results
}

// transferToHost 传输镜像到单个主机（带重试）
func (m *Manager) transferToHost(host string, imageCfg config.ImageConfig, tarFile string, progress *mpb.Progress) TransferResult {
	var lastErr error
	maxRetries := m.cfg.Transfer.Retry

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := m.doTransfer(host, imageCfg, tarFile, progress)
		if err == nil {
			return TransferResult{
				Host:    host,
				Image:   imageCfg.Name,
				Success: true,
			}
		}

		lastErr = err
		if attempt < maxRetries {
			time.Sleep(2 * time.Second) // 重试前等待
		}
	}

	return TransferResult{
		Host:    host,
		Image:   imageCfg.Name,
		Success: false,
		Error:   lastErr,
	}
}

// doTransfer 执行实际的传输操作
func (m *Manager) doTransfer(host string, imageCfg config.ImageConfig, tarFile string, progress *mpb.Progress) error {
	// 1. 创建SSH客户端
	sshClient := ssh.NewClient(
		host,
		m.cfg.SSH.Port,
		m.cfg.SSH.User,
		m.cfg.SSH.Password,
		m.cfg.SSH.KeyFile,
		m.cfg.SSH.Timeout,
		progress,
	)

	// 2. 连接SSH
	if err := sshClient.Connect(); err != nil {
		return err
	}
	defer sshClient.Close()

	// 3. 检查远程Docker是否可用
	if err := sshClient.CheckDockerAvailable(); err != nil {
		return err
	}

	// 4. 上传tar文件到远程临时目录
	remoteTarPath := filepath.Join(m.cfg.RemoteStorage.TempDir, filepath.Base(tarFile))
	if err := sshClient.UploadFile(tarFile, remoteTarPath); err != nil {
		return err
	}

	// 5. 执行hooks（全局 + 镜像级）
	vars := map[string]string{"image": imageCfg.Name}

	// 5a. 执行全局 pre_load hooks
	if len(m.cfg.Hooks.PreLoad) > 0 {
		sshClient.ExecuteHooks("pre_load", m.cfg.Hooks.PreLoad, vars)
	}
	// 5b. 执行镜像级 pre_load hooks
	if len(imageCfg.Hooks.PreLoad) > 0 {
		sshClient.ExecuteHooks("pre_load", imageCfg.Hooks.PreLoad, vars)
	}

	// 6. 根据配置决定是否加载Docker镜像
	if m.cfg.Transfer.AutoLoad {
		if err := sshClient.LoadDockerImage(remoteTarPath); err != nil {
			return err
		}

		// 7. 执行post_load hooks（全局 + 镜像级）
		// 7a. 执行全局 post_load hooks
		if len(m.cfg.Hooks.PostLoad) > 0 {
			sshClient.ExecuteHooks("post_load", m.cfg.Hooks.PostLoad, vars)
		}
		// 7b. 执行镜像级 post_load hooks
		if len(imageCfg.Hooks.PostLoad) > 0 {
			sshClient.ExecuteHooks("post_load", imageCfg.Hooks.PostLoad, vars)
		}
	}

	// 8. 根据配置决定是否清理远程tar文件
	if m.cfg.RemoteStorage.AutoCleanup {
		sshClient.RemoveRemoteFile(remoteTarPath)
	}

	return nil
}
