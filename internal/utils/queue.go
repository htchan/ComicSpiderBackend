package utils

import (
	"fmt"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/nats-io/nats.go"
)

func ConnectNatsQueue(conf *config.NatsConfig) (*nats.Conn, error) {
	nc, err := nats.Connect(fmt.Sprintf("nats://%s:%s@%s", conf.User, conf.Password, conf.URL))
	if err != nil {
		return nil, err
	}

	return nc, err
}
