# Apple Container 部署

Apple Container 是可选的 macOS 本地运行方式。默认本地部署仍以 WSL + Docker Compose 为主；本页只用于已经安装 Apple Container CLI，且 PostgreSQL/Redis 已经可从容器访问的场景。

## 默认值

- 默认镜像：`ghcr.io/zy4869/sub2api:latest`
- 默认仓库命名空间：`ZY4869/sub2api`
- 默认容器名：`sub2api`
- 默认访问地址：`http://127.0.0.1:8080`
- 默认数据目录：`./deploy/apple-container-data`

脚本不会内置数据库密码、管理员密码、JWT secret 或第三方 OAuth secret。需要固定密钥时请通过环境变量传入。

## 准备外部依赖

Apple Container 不读取本仓库的 Docker Compose 文件。请先准备可访问的 PostgreSQL 和 Redis，例如本机服务、远程服务，或你自己管理的容器服务。

常用环境变量：

```sh
export DATABASE_HOST=host.docker.internal
export DATABASE_PORT=5432
export DATABASE_USER=sub2api
export DATABASE_PASSWORD='change-me'
export DATABASE_DBNAME=sub2api

export REDIS_HOST=host.docker.internal
export REDIS_PORT=6379
export REDIS_PASSWORD=''
```

如果你的 Apple Container 环境不能解析 `host.docker.internal`，请改成宿主机在容器网络中可访问的 IP 或 DNS 名称。

## 启动

```sh
chmod +x deploy/apple-container.sh
SERVER_PORT=8080 \
POSTGRES_PASSWORD='change-me' \
JWT_SECRET='replace-with-a-fixed-secret' \
TOTP_ENCRYPTION_KEY='replace-with-a-fixed-secret' \
./deploy/apple-container.sh up
```

查看状态和日志：

```sh
./deploy/apple-container.sh status
./deploy/apple-container.sh logs
```

停止并移除：

```sh
./deploy/apple-container.sh stop
./deploy/apple-container.sh rm
```

## 更新镜像

```sh
SUB2API_IMAGE=ghcr.io/zy4869/sub2api:latest ./deploy/apple-container.sh restart
```

脚本会重新拉取镜像、删除旧应用容器并创建新容器；数据目录不会被删除。

## 安全提示

- 生产或长期本地使用时设置固定 `JWT_SECRET` 和 `TOTP_ENCRYPTION_KEY`。
- 不要把真实密钥写入脚本；使用 shell 环境变量、密钥管理器或本地安全的 env 注入方式。
- Apple Container 方式不替代 Docker Compose 的 PostgreSQL/Redis 编排能力。
