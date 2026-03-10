# Sub2API

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.25.7-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

**AI API 网关平台 - 订阅配额分发管理**

中文 | [English](README_EN.md)

</div>

---

## 默认说明

当前 GitHub 默认展示中文入口页。

- 完整中文文档：`README_CN.md`
- English documentation: `README_EN.md`
- 部署文档：`deploy/README.md`

## 快速命令

### 一键安装

```bash
curl -sSL https://raw.githubusercontent.com/ZY4869/sub2api/main/deploy/install.sh | sudo bash
```

### 覆盖升级

已安装后，推荐使用升级命令覆盖现有内容：

```bash
curl -sSL https://raw.githubusercontent.com/ZY4869/sub2api/main/deploy/install.sh | sudo bash -s -- upgrade
```

### 指定版本升级

```bash
curl -sSL https://raw.githubusercontent.com/ZY4869/sub2api/main/deploy/install.sh | sudo bash -s -- upgrade -v v0.0.1
```

### 卸载

```bash
curl -sSL https://raw.githubusercontent.com/ZY4869/sub2api/main/deploy/install.sh | sudo bash -s -- uninstall -y
```

## 发布说明

- 当前仓库可直接复用 `.github/workflows/release.yml` 与 `.goreleaser*.yaml`
- 推荐先发布预发布标签，例如 `v0.0.1-rc1`
- 发布与部署细节见 `README_CN.md` 和 `deploy/README.md`
