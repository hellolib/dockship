# 🚢 dockship

> **dockship** — 一个轻量级 Docker 镜像分发工具，用于在没有镜像仓库（registry）的环境下，高效地将本地或远程镜像传输到多台目标主机，并在远端自动执行 `docker load`。

---

## ✨ 功能特性

- ✅ **镜像自动获取**：支持从本地或远程拉取镜像  
- ✅ **镜像打包**：自动执行 `docker save` 保存为 `.tar` 文件  
- ✅ **文件传输**：通过 SSH 安全传输镜像包至目标主机  
- ✅ **远程加载**：自动在目标主机执行 `docker load -i`  
- ✅ **多主机分发**：支持并行发送至多个主机  
- ✅ **离线可用**：无需依赖 Docker Registry  
- ✅ **跨平台编译**：单可执行文件运行，无需额外依赖

---

## ⚙️ 使用方式

### 1️⃣ 配置文件

在项目根目录创建 `config.yaml`：

```yaml
# config.yaml
images:
  - nginx:1.25
  - redis:7.0
targets:
  - 192.168.1.10
  - 192.168.1.11

ssh:
  user: root
  key: ~/.ssh/id_rsa
  port: 22
