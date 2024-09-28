package websitebatchupdate

import (
	"context"
	"fmt"
	"time"

	"github.com/htchan/WebHistory/internal/repository"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/website_update"
	"github.com/htchan/goworkers"
	"github.com/htchan/goworkers/stream"
	"github.com/htchan/goworkers/stream/redis"
	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidiscompat"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Task struct {
	stream      stream.Stream
	vendorTasks websiteupdate.Tasks
	rpo         repository.Repostory
}

var _ goworkers.Task = (*Task)(nil)

func NewTask(redisClient rueidis.Client, vendorTasks websiteupdate.Tasks, rpo repository.Repostory) *Task {
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
		vendorTasks: vendorTasks,
		rpo:         rpo,
	}
}

func (t *Task) Name() string {
	return "website_batch_update"
}

func (t *Task) Execute(ctx context.Context, params interface{}) error {
	tr := otel.Tracer("htchan/WebHistory/website-batch-update")
	ctx, span := tr.Start(ctx, "Website Batch Update")
	defer span.End()

	redisParams := params.(rueidiscompat.XMessage)
	ctx = log.With().Str("task", t.Name()).Str("redis_task_id", redisParams.ID).Logger().WithContext(ctx)
	zerolog.Ctx(ctx).
		Info().
		Interface("values", redisParams.Values).
		Msg("execute website batch update task")

	// load all webstes from db
	_, dbSpan := tr.Start(ctx, "Load Websites From DB")
	websites, err := t.rpo.FindWebsites()
	dbSpan.End()

	if err != nil {
		dbSpan.SetStatus(codes.Error, err.Error())
		dbSpan.RecordError(err)

		return fmt.Errorf("load website from db failed: %w", err)
	}

	// publish update job for all website
	iterateCtx, iterateSpan := tr.Start(ctx, "Iterate Websites")
	for _, website := range websites {
		websiteCtx, websiteSpan := tr.Start(iterateCtx, "Publish Update Website")

		supportTasks, errs := t.vendorTasks.Publish(websiteCtx, websiteupdate.WebsiteUpdateParams{Website: website})
		for i, err := range errs {
			if err != nil {
				zerolog.Ctx(ctx).Error().Err(err).
					Str("task_name", supportTasks[i]).
					Str("website_uuid", website.UUID).
					Str("website_url", website.URL).
					Str("website_title", website.Title).
					Msg("publish website update task failed")
			}
		}

		websiteSpan.SetAttributes(
			append(
				website.OtelAttributes(),
				attribute.StringSlice("support_tasks", supportTasks),
			)...,
		)
		websiteSpan.End()

		if len(supportTasks) == 0 {
			zerolog.Ctx(ctx).Warn().
				Str("website_uuid", website.UUID).
				Msg("no support task for website")
		} else if len(supportTasks) > 1 {
			zerolog.Ctx(ctx).Warn().
				Str("website_uuid", website.UUID).
				Strs("support_task_names", supportTasks).
				Msg("multiple support task for website")
		}
	}
	iterateSpan.End()

	ackCtx, ackSpan := tr.Start(ctx, "Acknowledge Message")
	ackErr := t.stream.Acknowledge(ackCtx, params)
	ackSpan.End()

	if ackErr != nil {
		ackSpan.SetStatus(codes.Error, ackErr.Error())
		ackSpan.RecordError(ackErr)

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

	createErr := t.stream.CreateStream(ctx)
	if createErr != nil {
		log.Error().Err(createErr).Str("task", t.Name()).Msg("create stream failed")

		return createErr
	}

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
