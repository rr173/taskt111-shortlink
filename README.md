# taskt111-shortlink

用 Go 开发的短链 + 访问统计服务。把长 URL 映射为短码，提供跳转、批量创建、分页列表、
点击记录与按天/来源/热门排行的访问统计。数据统一落库到单个 SQLite 文件，重启后自动恢复。

## 技术栈

- Go 1.26.3（`GOTOOLCHAIN=local`，`go.mod` 语言版本 `go 1.26.3`）
- 纯 Go SQLite 驱动 `modernc.org/sqlite`（CGO 关闭，跨平台静态构建）
- 标准库 `net/http` 路由（Go 1.22+ 模式匹配），无第三方 Web 框架

## 快速开始

```bash
GOTOOLCHAIN=local GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn go mod download
CGO_ENABLED=0 GOTOOLCHAIN=local go run . --addr :8080 --db shortlink.db
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
go run . --smoke-test
```

## 目录结构

```
env/
├── main.go                服务入口
├── smoke.go               --smoke-test 自检
├── internal/
│   ├── store/             SQLite 持久化（链接 / 点击 / 统计原始数据）
│   ├── idgen/             短码生成（base62 + 碰撞重试）
│   ├── link/              短链领域：创建 / 解析 / 过期与点击上限 / 批量 / 状态
│   ├── click/             点击采集
│   ├── stat/              统计聚合：按天 / 来源 / 热门 / 总量
│   └── httpapi/           HTTP 路由与处理器
├── Dockerfile             健康基线自检镜像（module mode 构建）
├── benzhi.Dockerfile      benzhi 评测镜像
├── build_benzhi_docker.sh benzhi 镜像构建脚本
└── BENZHI_README.md       评测与命令说明
```
