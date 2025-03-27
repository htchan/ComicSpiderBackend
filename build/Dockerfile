FROM golang:1.24.1-alpine AS builder

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


FROM scratch

ARG SERVICE
ENV SERVICE=$SERVICE

WORKDIR /usr/src/app

COPY --from=builder /go/src/github.com/htchan/WebHistory/$SERVICE .

ENTRYPOINT ["/usr/src/app/$SERVICE"]
