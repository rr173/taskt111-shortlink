# 官方 Go 镜像（daocloud 镜像源，版本与本地 Go 1.26.3 完全一致），自带完整工具链。
FROM docker.m.daocloud.io/library/golang:1.26.3-bookworm

WORKDIR /app

# 先复制依赖文件并下载依赖（benzhi 构建阶段允许联网下载），利用缓存。
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 预编译一次，把编译缓存留在镜像里；不影响源码修改。
RUN go build ./...

# 容器启动后进入 shell，方便操作。
CMD ["bash"]
