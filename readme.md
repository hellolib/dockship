# 🚢 dockship

> **dockship** — 一个轻量级 Docker 镜像分发工具，用于在没有镜像仓库（registry）的环境下，高效地将本地或远程镜像传输到多台目标主机，并在远端自动执行 `docker load`。

---

## 🎯 使用场景

1. **无外网环境部署**：内网环境无法访问 Docker Hub 或私有仓库
2. **快速批量分发**：需要将镜像快速分发到多台主机

---

## ✨ 功能特性

- ✅ **镜像自动获取**：支持从本地或远程拉取镜像
- ✅ **镜像打包**：自动执行 `docker save` 保存为 `.tar` 文件
- ✅ **文件传输**：通过 SSH/SFTP 安全传输镜像包至目标主机
- ✅ **实时进度条**：多主机并发传输时显示实时上传进度
- ✅ **远程加载**：自动在目标主机执行 `docker load -i`
- ✅ **Hooks 机制**：支持镜像加载前后执行自定义命令
- ✅ **多主机并发**：支持并行传输到多台主机，可配置并发数
- ✅ **失败重试**：支持配置失败重试次数
- ✅ **自动清理**：支持本地和远程临时文件自动清理
- ✅ **离线可用**：无需依赖 Docker Registry
- ✅ **跨平台编译**：单可执行文件运行，无需额外依赖

---

## 📦 安装

### 编译安装

**方式 1：使用 Makefile（推荐）**

```bash
git clone https://github.com/hellolib/dockship.git
cd dockship
make build
```

Makefile 会自动注入版本信息（Git commit、构建时间等）。

**方式 2：直接使用 go build**

```bash
git clone https://github.com/hellolib/dockship.git
cd dockship
go build -o dockship
```

### Makefile 命令

```bash
make build    # 编译项目
make clean    # 清理编译产物
make install  # 安装到 $GOPATH/bin
make version  # 显示版本信息
make run      # 编译并运行
make help     # 显示帮助信息
```

### 查看版本

```bash
# 查看详细版本信息
./dockship version

# 或使用标志
./dockship --version
./dockship -v
```

---

## ⚙️ 使用方式

### 1️⃣ 配置文件

在项目根目录创建 `config.yaml`：

```yaml
# Docker镜像列表
images:
  - nginx:1.25
  - redis:7.0

# 目标主机列表
target_hosts:
  - 192.168.1.10
  - 192.168.1.11
  - 192.168.1.12

# SSH连接配置
ssh:
  user: root
  pwd: your_password             # SSH密码（不推荐，建议使用密钥）
  # key_file: ~/.ssh/id_rsa      # SSH密钥文件路径（推荐）
  port: 22                        # SSH端口
  timeout: 30                     # 连接超时时间（秒）

# 本地存储配置
local_storage:
  temp_dir: /tmp/dockship         # 本地临时文件目录
  auto_cleanup: true              # 传输完成后是否自动清理本地临时文件

# 远程存储配置
remote_storage:
  temp_dir: /tmp/dockship         # 远程主机临时文件目录
  auto_cleanup: true              # 镜像加载完成后是否自动清理远程临时文件

# 传输配置
transfer:
  concurrent: 5                   # 并发传输主机数量
  retry: 3                        # 失败重试次数

# Hooks配置（对所有镜像和主机生效）
hooks:
  # 镜像加载前执行（在docker load之前）
  pre_load:
    - echo "准备加载镜像..."
    - docker service ls

  # 镜像加载后执行（在docker load之后）
  post_load:
    - echo "镜像加载完成"
    - docker images | tail -5
    # 示例：更新Swarm服务
    # - docker service update --image <镜像名> <服务名>
```

### 2️⃣ 执行传输

```bash
# 使用默认配置文件 config.yaml
./dockship transfer
# 或使用简短别名
./dockship go

# 使用自定义配置文件
./dockship transfer -c custom.yaml
# 或
./dockship go -c custom.yaml
```

### 3️⃣ 运行示例

```
📝 加载配置文件: config.yaml

📋 配置信息：
  镜像数量: 2
    1. nginx:1.25
    2. redis:7.0
  目标主机: 3 台
    1. 192.168.1.10
    2. 192.168.1.11
    3. 192.168.1.12
  并发数: 5
  重试次数: 3
  SSH用户: root
  SSH端口: 22
  认证方式: 密钥 (~/.ssh/id_rsa)

⚠️  确认要继续执行吗? [y/N]: y

🚀 Dockship 开始执行镜像传输任务
============================================================

📦 处理镜像: nginx:1.25
------------------------------------------------------------
✅ 镜像已存在: nginx:1.25
✅ 镜像保存成功: /tmp/dockship/nginx_1.25.tar

📤 [192.168.1.10] ████████████████████ 125.5MB/125.5MB 100% 15.2MB/s
📤 [192.168.1.11] ████████████████████ 125.5MB/125.5MB 100% 14.8MB/s
📤 [192.168.1.12] ████████████████████ 125.5MB/125.5MB 100% 16.1MB/s

  🔧 [192.168.1.10] 执行 pre_load hooks...
    [192.168.1.10][1/2] 执行: echo "准备加载镜像..."
    [192.168.1.10] ✅ 成功
    [192.168.1.10] 输出: 准备加载镜像...

  🔧 [192.168.1.10] 执行 post_load hooks...
    [192.168.1.10][1/2] 执行: docker images | tail -5
    [192.168.1.10] ✅ 成功
    [192.168.1.10] 输出: nginx  1.25  ...

  ✅ [192.168.1.10] 镜像传输完成
  ✅ [192.168.1.11] 镜像传输完成
  ✅ [192.168.1.12] 镜像传输完成

📊 镜像 nginx:1.25 传输统计: 成功 3 台，失败 0 台

============================================================
✅ 所有任务完成，总耗时: 32.15 秒
```

---

## 🔧 Hooks 机制

Hooks 允许你在镜像加载的不同阶段执行自定义命令，这对于自动化部署非常有用。

### 使用场景

#### 1. Docker Swarm 服务更新

```yaml
hooks:
  post_load:
    - docker service update --image nginx:1.25 my-web-service
    - docker service update --image redis:7.0 my-cache-service
```

#### 2. 验证镜像

```yaml
hooks:
  pre_load:
    - docker images | grep nginx  # 检查是否存在旧版本
  post_load:
    - docker images | grep nginx  # 验证新版本已加载
    - docker inspect nginx:1.25   # 检查镜像详情
```

#### 3. 通知和日志

```yaml
hooks:
  pre_load:
    - echo "$(date): 开始更新镜像" >> /var/log/dockship.log
  post_load:
    - echo "$(date): 镜像更新完成" >> /var/log/dockship.log
    - curl -X POST https://your-webhook.com/notify -d "镜像更新成功"
```

### Hooks 特性

- ✅ 支持 `pre_load` 和 `post_load` 两个阶段
- ✅ 命令按顺序依次执行
- ✅ 失败不中断（continue-on-error），不影响主流程
- ✅ 显示每条命令的执行结果和输出
- ✅ 自动标注主机信息，便于多主机并发时区分

---

## 📝 配置说明

### SSH 认证方式

**推荐使用密钥认证**（更安全）：

```yaml
ssh:
  user: root
  key_file: ~/.ssh/id_rsa
  port: 22
  timeout: 30
```

密码认证（不推荐）：

```yaml
ssh:
  user: root
  pwd: your_password
  port: 22
  timeout: 30
```

### 并发控制

`concurrent` 参数控制同时传输的主机数量：

```yaml
transfer:
  concurrent: 5    # 同时向5台主机传输
  retry: 3         # 失败重试3次
```

`concurrent` 同时决定不同镜像作业的并发度以及单个镜像向多主机传输的并发度，可根据本地磁盘与网络能力调节。

### 自动清理

```yaml
local_storage:
  auto_cleanup: true   # 传输完成后自动删除本地 tar 文件

remote_storage:
  auto_cleanup: true   # 镜像加载后自动删除远程 tar 文件
```

---

## 🗒 TODO

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

## 📄 License

MIT License
