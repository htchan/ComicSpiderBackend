FROM golang:1.24.2-alpine AS builder

ARG SERVICE

WORKDIR /go/src/github.com/htchan/WebHistory
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


FROM alpine

ARG SERVICE
ENV SERVICE=$SERVICE

WORKDIR /usr/src/app

COPY --from=builder --chmod=755 /go/src/github.com/htchan/WebHistory/${SERVICE} ./app

ENTRYPOINT ["/usr/src/app/app"]
