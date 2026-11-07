ARG GO_VERSION

FROM golang:${GO_VERSION}-bookworm AS builder

ARG SERVICE

WORKDIR /go/src/github.com/example/example
RUN go env -w GOMODCACHE=/root/.cache/go-build \
    && go env -w CGO_ENABLED=0

RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download -x

RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=internal,target=internal \
    --mount=type=bind,source=docs,target=docs \
    --mount=type=bind,source=cmd,target=cmd \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -v ./cmd/$SERVICE

RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=cache,target=/root/.cache/go-build \
    PW_VERSION=$(go list -m -f '{{.Version}}' github.com/mxschmitt/playwright-go) && \
    mkdir -p /tmp/pw-installer && cd /tmp/pw-installer && \
    go mod init temp_install && \
    go get github.com/mxschmitt/playwright-go@${PW_VERSION} && \
    printf 'package main\nimport "github.com/mxschmitt/playwright-go"\nfunc main() {\n\t_ = playwright.Install(&playwright.RunOptions{\n\t\tSkipInstallBrowsers: true,\n\t})\n}' > main.go && \
    go run main.go && \
    rm -rf /tmp/pw-installer

FROM debian:bookworm

ARG SERVICE
ENV SERVICE=$SERVICE

# Install Node.js (strictly required to run the playwright-go driver bridge)
RUN apt-get update && apt-get install -y \
    ca-certificates \
    nodejs \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/src/app

COPY --from=builder --chmod=755 /go/src/github.com/example/example/${SERVICE} ./app
COPY --from=builder /root/.cache/ms-playwright-go /root/.cache/ms-playwright-go

ENTRYPOINT ["/usr/src/app/app"]
