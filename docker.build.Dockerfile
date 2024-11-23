FROM alpine:3.20

RUN apk update && apk add wget build-base

ARG ARCHITECTURE="arm64"
ARG GO_BINARY_NAME="go1.23.3.linux-$ARCHITECTURE.tar.gz"

WORKDIR /build

RUN wget "https://go.dev/dl/$GO_BINARY_NAME" && rm -rf /usr/local/go && tar -C /usr/local -xzf "$GO_BINARY_NAME"

WORKDIR /build/openkvm

COPY . .

RUN /usr/local/go/bin/go build -o openkvm .

# export docker_http_proxy=http://host.docker.internal:1080
# export docker_architecture=arm64
# docker build --build-arg http_proxy=$docker_http_proxy --build-arg https_proxy=$docker_http_proxy --build-arg ARCHITECTURE=$docker_architecture --platform "linux/$docker_architecture" -t allape/openkvm:latest -f docker.build.Dockerfile .
# docker create --name openkvm allape/openkvm:latest
# docker cp openkvm:/build/openkvm/openkvm ./openkvm
# docker rm openkvm
