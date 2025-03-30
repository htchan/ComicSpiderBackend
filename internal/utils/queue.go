package utils

import (
	"github.com/htchan/WebHistory/internal/config"
	"github.com/nats-io/nats.go"
)

func ConnectNatsQueue(conf *config.NatsConfig) (*nats.Conn, error) {
	nc, err := nats.Connect(conf.URL)
	if err != nil {
		return nil, err
	}

	return nc, err
}
