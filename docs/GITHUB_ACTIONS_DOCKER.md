# GitHub Actions 发布 Docker 镜像

这个方案由 GitHub Actions 构建镜像并推送到国内容器镜像仓库。Windows 本机不需要安装 Docker，国内服务器也不需要连接 GitHub。

## 1. 创建阿里云 ACR 仓库

在阿里云“容器镜像服务 ACR”中创建命名空间和镜像仓库。仓库类型可选私有，代码源不需要绑定 GitHub。

## 2. 配置 GitHub Secrets

进入 GitHub 仓库的 `Settings -> Secrets and variables -> Actions`，创建：

```text
DOCKER_REGISTRY=registry.cn-shanghai.aliyuncs.com
DOCKER_IMAGE=你的命名空间/backend-go
DOCKER_USERNAME=阿里云镜像仓库登录用户名
DOCKER_PASSWORD=阿里云镜像仓库访问凭证
```

`DOCKER_REGISTRY` 不要带 `https://`，`DOCKER_IMAGE` 不要带标签。

推送到 `main` 后，工作流会发布：

```text
registry.cn-shanghai.aliyuncs.com/你的命名空间/backend-go:latest
registry.cn-shanghai.aliyuncs.com/你的命名空间/backend-go:sha-提交号
```

也可以在 GitHub 的 `Actions -> Publish Docker image` 中手动运行。

## 3. 国内服务器部署

服务器只需要保存 `docker-compose.yml`、`.env` 和数据目录。创建 `.env`：

```env
BACKEND_IMAGE=registry.cn-shanghai.aliyuncs.com/你的命名空间/backend-go
VERSION=latest
HTTP_PUBLIC_API_BASE=https://api.example.com
HTTP_CORS_ORIGINS=https://admin.example.com
```

首次启动：

```bash
cd /opt/backend-go
mkdir -p config databases storage
chown -R 10001:10001 config databases storage
docker login registry.cn-shanghai.aliyuncs.com
docker compose pull
docker compose up -d --no-build
docker compose logs -f backend-go
```

后续更新：

```bash
cd /opt/backend-go
docker compose pull
docker compose up -d --no-build
docker image prune -f
```

建议生产环境把 `VERSION` 固定为 `sha-提交号`，确认新版本正常后再修改该值，可以方便地回滚。
