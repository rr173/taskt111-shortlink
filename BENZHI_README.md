# taskt111-shortlink 评测说明（BENZHI）

## 项目用途

`taskt111-shortlink` 是一个用 Go 开发的**短链 + 访问统计服务**：把长 URL 映射为短码，
提供跳转、批量创建、分页列表、点击记录与按天/来源/热门排行的访问统计。所有数据统一落库到
单个 SQLite 文件，服务重启后通过重新打开同一文件恢复全部状态（链接、点击记录、统计所需原始数据），
具备保存与重启恢复路径。小型数据使用嵌入式 SQLite，不依赖任何外部服务。

## 标准构建 / 运行 / 测试命令

以下命令均假定在仓库根目录（即含 `go.mod` 的 `env/`）执行，且使用 PATH 中的 `go`
（已设置 `GOTOOLCHAIN=local`，Go 版本 `go1.26.3`）。

```bash
# 1) 下载 Go module 依赖（构建阶段使用国内代理）
GOTOOLCHAIN=local GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn \
  go mod download

# 2) 编译（CGO 关闭，双架构 linux/amd64 与 linux/arm64）
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...

# 3) 运行（HTTP 服务，默认 :8080，数据库文件 shortlink.db）
CGO_ENABLED=0 GOTOOLCHAIN=local go run . --addr :8080 --db shortlink.db

# 4) 测试
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...

# 5) 自检（不依赖外部服务，执行后自行退出，exit 0 表示通过）
go run . --smoke-test
```

## Docker 容器交付

```bash
# 健康基线自检镜像（module mode 构建，支持 --smoke-test）
docker buildx build --platform linux/amd64 --load -t go-task-check:amd64 .
docker run --rm go-task-check:amd64 --smoke-test
docker buildx build --platform linux/arm64 --load -t go-task-check:arm64 .
docker run --rm go-task-check:arm64 --smoke-test

# benzhi 评测镜像（允许在构建阶段联网下载依赖）
bash ./build_benzhi_docker.sh go-task-benzhi:amd64 linux/amd64
docker run --rm go-task-benzhi:amd64 go version
bash ./build_benzhi_docker.sh go-task-benzhi:arm64 linux/arm64
docker run --rm go-task-benzhi:arm64 go version
```

镜像基础版本固定为 `golang:1.26.3`（daocloud 镜像源
`docker.m.daocloud.io/library/golang:1.26.3-bookworm`），构建 `CGO_ENABLED=0`，
目标平台 `linux/amd64` 与 `linux/arm64` 双架构。

## 主要 HTTP API（节选）

- `POST /api/links` 创建短链（可指定 custom_code）
- `POST /api/links/bulk` 批量创建
- `GET /api/links?limit=&offset=&owner=` 分页列表
- `GET /api/links/{code}/info`、`/status`、`/stats`、`/stats/daily`、`/stats/referers`
- `PUT /api/links/{code}` 更新；`DELETE /api/links/{code}` 删除
- `POST /api/links/{code}/click` 记录点击；`GET /api/links/{code}/reset` 重置点击
- `GET /api/stats/top`、`/api/stats/recent`
- `GET /api/health`、`/api/health/ready`、`/api/metrics`
- `GET /{code}` 公开 302 跳转
