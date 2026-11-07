ARG GO_VERSION

FROM golang:${GO_VERSION}-alpine AS builder

ARG SERVICE

WORKDIR /go/src/github.com/htchan/WebHistory
RUN go env -w GOMODCACHE=/root/.cache/go-build \
    && go env -w CGO_ENABLED=0

RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download -x
# Dynamically extract the exact version from go.mod and install the driver
RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=cache,target=/root/.cache/go-build \
    PW_VERSION=$(go list -m -f '{{.Version}}' github.com/mxschmitt/playwright-go) && \
    go run github.com/mxschmitt/playwright-go/cmd/playwright@${PW_VERSION} install --skip-browsers

RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=internal,target=internal \
    --mount=type=bind,source=docs,target=docs \
    --mount=type=bind,source=cmd,target=cmd \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -v ./cmd/$SERVICE


FROM alpine

ARG SERVICE
ENV SERVICE=$SERVICE

WORKDIR /usr/src/app

COPY --from=builder --chmod=755 /go/src/github.com/htchan/WebHistory/${SERVICE} ./app

COPY --from=builder /root/.cache/ms-playwright /root/.cache/ms-playwright

ENTRYPOINT ["/usr/src/app/app"]
