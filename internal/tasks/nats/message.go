package nats

import "github.com/nats-io/nats.go/jetstream"

//go:generate go tool mockgen -destination=../../mock/nats/msg.go -package=mocknats . NatsMsg
type NatsMsg jetstream.Msg
