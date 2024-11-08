FROM debian:bullseye

RUN apt-get update && \
    apt-get install -y wget build-essential g++ cmake libopencv-dev

ARG GO_BINARY_NAME="go1.23.3.linux-arm64.tar.gz"

WORKDIR /build

RUN wget "https://go.dev/dl/$GO_BINARY_NAME" && rm -rf /usr/local/go && sudo tar -C /usr/local -xzf "$GO_BINARY_NAME"

WORKDIR /build/openkvm

COPY . .

RUN sed -i 's/gocv\.io\/x\/gocv v\S+/gocv.io\/x\/gocv opencv-4.5.1/g' go.mod && \
    /usr/local/go/bin/go mod tidy

#RUN /usr/local/go/bin/go build -o openkvm .

CMD ["/usr/local/go/bin/go", "build", "-o", "../openkvm", "."]

# export docker_http_proxy=http://
# docker build --build-arg http_proxy=$docker_http_proxy --build-arg https_proxy=$docker_http_proxy --progress plain --platform linux/arm64 -t allape/openkvm:debian11-arm64-latest -f builder.debian11.arm64.Dockerfile .
# docker run -v $(pwd)/openkvm:/build/openkvm/openkvm --rm allape/openkvm:debian11-arm64-latest
# -v $(pwd):/build/openkvm
