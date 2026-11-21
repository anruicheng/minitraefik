# ---- 阶段 1：编译 ----
FROM golang:1.27-alpine AS build
WORKDIR /src

# 先只复制 go.mod/go.sum，单独下载依赖，利用 Docker 层缓存
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# 只复制编译需要的目录，避免无关文件改动打穿层缓存
COPY proxy/ ./proxy/
RUN CGO_ENABLED=0 go build -o /proxy ./proxy

# ---- 阶段 2：运行 ----
FROM alpine:3.19
COPY --from=build /proxy /proxy
ENTRYPOINT ["/proxy"]