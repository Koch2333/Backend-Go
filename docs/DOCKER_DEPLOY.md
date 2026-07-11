# Docker 部署

这个项目推荐用 Docker Compose 跑后端容器，再用 nginx / Caddy 在宿主机做 HTTPS 反向代理。

## 目录

服务器上建议放在：

```text
/opt/backend-go/
  Dockerfile
  docker-compose.yml
  config/
  databases/
  storage/
```

`config/`、`databases/`、`storage/` 是持久数据，必须备份。镜像里不打包这些目录，容器运行时通过 volume 挂载。

## 首次启动

```bash
cd /opt/backend-go
mkdir -p config databases storage
sudo chown -R 10001:10001 config databases storage
docker compose up -d --build
docker compose logs -f backend-go
```

首次运行会自动生成各模块的默认配置：

```text
config/<module>/.env
databases/
storage/
```

生成后先停容器，修改生产配置，再重新启动：

```bash
docker compose down
vim config/roundnfc/.env
vim config/redirect/.env
docker compose up -d
```

至少要改：

- 后台默认密码，不要保留 `admin / admin`
- JWT / HMAC 等密钥保持私密
- WebAuthn 的 `*_WEBAUTHN_RPID` 和 `*_WEBAUTHN_ORIGINS`
- 如需 Turnstile / COS / 邮件，填对应模块的配置

## 环境变量

生产建议在同目录创建 `.env`：

```env
VERSION=prod
COMMIT=manual
BUILD=2026-07-11T00:00:00Z

HTTP_PUBLIC_API_BASE=https://api.example.com
HTTP_CORS_ORIGINS=https://admin.example.com
```

Compose 默认只把服务端口绑定到 `127.0.0.1`：

```yaml
ports:
  - "127.0.0.1:8080:8080"
  - "127.0.0.1:8081:8081"
```

公网不要直接开放 8080 / 8081，交给 nginx / Caddy 暴露 443。

## nginx 示例

```nginx
server {
    listen 443 ssl http2;
    server_name api.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}

server {
    listen 443 ssl http2;
    server_name admin.example.com;

    location / {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

对应配置：

```env
HTTP_PUBLIC_API_BASE=https://api.example.com
HTTP_CORS_ORIGINS=https://admin.example.com
```

如果使用 Passkey / WebAuthn：

```env
ROUNDNFC_WEBAUTHN_RPID=admin.example.com
ROUNDNFC_WEBAUTHN_ORIGINS=https://admin.example.com
REDIRECT_WEBAUTHN_RPID=admin.example.com
REDIRECT_WEBAUTHN_ORIGINS=https://admin.example.com
```

## 更新

```bash
cd /opt/backend-go
git pull
docker compose up -d --build
docker compose logs -f backend-go
```

## 备份

至少备份：

```text
config/
databases/
storage/
```

恢复时把这三个目录放回同级目录，然后 `docker compose up -d`。
