# 健康基线 Docker 自检镜像：与本地一致的 Go 1.26.3（daocloud 镜像源），
# 使用 Go module 下载依赖，二进制支持 --smoke-test 自检后自行退出。
FROM docker.m.daocloud.io/library/golang:1.26.3-bookworm

ENV CGO_ENABLED=0 \
    GOTOOLCHAIN=local \
    GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /usr/local/bin/shortlink .

ENTRYPOINT ["/usr/local/bin/shortlink"]
