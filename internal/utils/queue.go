package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func ConnectNatsQueue(conf *config.NatsConfig) (*nats.Conn, error) {
	nc, err := nats.Connect(fmt.Sprintf("nats://%s:%s@%s", conf.User, conf.Password, conf.URL))
	if err != nil {
		return nil, err
	}

	return nc, err
}

func Subscribe(ctx context.Context, nc *nats.Conn, subject string, handler jetstream.MessageHandler) (jetstream.ConsumeContext, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     subject,
		Subjects: []string{subject},
		MaxAge:   time.Hour * 24 * 7,
	})
	if err != nil {
		return nil, err
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:      subject,
		Durable:   subject,
		AckPolicy: jetstream.AckExplicitPolicy,
		AckWait:   time.Minute * 10,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(handler)
}
