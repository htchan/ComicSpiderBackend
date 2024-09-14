package websitebatchupdate

import (
	"context"
	"fmt"
	"time"

	"github.com/htchan/goworkers"
	worker "github.com/htchan/goworkers"
	"github.com/htchan/goworkers/stream"
	"github.com/htchan/goworkers/stream/redis"
	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidiscompat"
	"github.com/rs/zerolog/log"
)

type Task struct {
	stream stream.Stream
}

var _ worker.Task = (*Task)(nil)

func NewTask(redisClient rueidis.Client) *Task {
	return &Task{
		stream: redis.NewRedisStream(
			redisClient,
			"web-history/website-batch-update",
			"web-history-group",
			"batch-update-consumer",
			redis.Config{
				BlockDuration: 10 * time.Minute,
				IdleDuration:  24 * time.Hour,
			},
		),
	}
}

func (t *Task) Name() string {
	return "website_batch_update"
}

func (t *Task) Execute(ctx context.Context, params interface{}) error {
	redisParams := params.(rueidiscompat.XMessage)
	log.
		Info().
		Str("redis_task_id", redisParams.ID).
		Interface("values", redisParams.Values).
		Msg("execute website batch update task")

	// TODO: loop all websites from DB and publish update job
	ackErr := t.stream.Acknowledge(ctx, params)
	if ackErr != nil {
		return fmt.Errorf("website batch update acknowledge: %w", ackErr)
	}

	return nil
}

func (t *Task) Publish(ctx context.Context, params interface{}) error {
	err := t.stream.Publish(ctx, params)
	if err != nil {
		return fmt.Errorf("website batch update publish: %w", err)
	}
	return nil
}

func (t *Task) Subscribe(ctx context.Context, ch chan goworkers.Msg) error {
	log.Info().Msg("subscribe website batch update task")

	interfaceCh := make(chan interface{})
	// parse interfaceCh to goworkers.Msg
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-interfaceCh:
				ch <- goworkers.Msg{
					TaskName: t.Name(),
					Params:   msg,
				}
			}
		}
	}()

	err := t.stream.Subscribe(ctx, interfaceCh)
	if err != nil {
		return fmt.Errorf("website batch update subscribe: %w", err)
	}
	log.Info().Msg("finish subscribe website batch update task")

	return nil
}

func (t *Task) Unsubscribe() error {
	return nil
}
