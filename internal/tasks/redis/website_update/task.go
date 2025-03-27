package websiteupdate

import (
	"context"
	"fmt"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
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
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/sync/semaphore"
)

type Task struct {
	stream        stream.Stream
	vendorService vendors.VendorService
	rpo           repository.Repostory
	sema          *semaphore.Weighted
	websiteConf   *config.WebsiteConfig
}

var _ goworkers.Task = (*Task)(nil)

type Tasks []*Task

func NewTaskSet(redisClient rueidis.Client, services []vendors.VendorService, rpo repository.Repostory, conf *config.WebsiteConfig) Tasks {
	var updateTasks Tasks
	for _, service := range services {
		updateTasks = append(updateTasks, NewTask(redisClient, service, rpo, conf))
	}

	return updateTasks
}

func (ts Tasks) Publish(ctx context.Context, params WebsiteUpdateParams) ([]string, []error) {
	var errs []error
	var supportTasks []string

	for _, task := range ts {
		if !task.Support(&params.Website) {
			continue
		}
		supportTasks = append(supportTasks, task.Name())
		errs = append(errs, task.Publish(ctx, params))
	}

	return supportTasks, errs
}

func NewTask(redisClient rueidis.Client, vendorService vendors.VendorService, rpo repository.Repostory, conf *config.WebsiteConfig) *Task {
	return &Task{
		stream: redis.NewRedisStream(
			redisClient,
			fmt.Sprintf("web-history/website-update/%s", vendorService.Name()),
			"web-history-group",
			fmt.Sprintf("update-consumer/%s", vendorService.Name()),
			redis.Config{
				BlockDuration: 10 * time.Minute,
				IdleDuration:  24 * time.Hour,
			},
		),
		vendorService: vendorService,
		rpo:           rpo,
		sema:          semaphore.NewWeighted(1),
		websiteConf:   conf,
	}
}

func (t *Task) Name() string {
	return fmt.Sprintf("website_update/%s", t.vendorService.Name())
}

func (t *Task) Execute(ctx context.Context, params interface{}) error {
	defer func() {
		// sleep between each execution
		time.Sleep(1 * time.Second)
		t.sema.Release(1)
	}()

	tr := otel.Tracer("htchan/WebHistory/website-update")

	redisParams := params.(rueidiscompat.XMessage)
	parsedParams := FromMap(redisParams.Values, t.websiteConf)
	ctx = log.With().Str("task", t.Name()).Str("redis_task_id", redisParams.ID).Logger().WithContext(ctx)
	zerolog.Ctx(ctx).
		Info().
		Interface("values", redisParams.Values).
		Msg("execute website update task")

	ctx, span := tr.Start(parsedParams.UpdateCtxOtel(ctx), "Website Update")
	defer span.End()
	span.SpanContext()

	defer func() {
		ackCtx, ackSpan := tr.Start(ctx, "Acknowledge Message")
		ackErr := t.stream.Acknowledge(ackCtx, params)
		ackSpan.End()

		if ackErr != nil {
			ackSpan.SetStatus(codes.Error, ackErr.Error())
			ackSpan.RecordError(ackErr)

			zerolog.Ctx(ctx).Error().Err(ackErr).Msg("ack message failed")
		}
	}()

	span.SetAttributes(
		append(
			parsedParams.Website.OtelAttributes(),
			attribute.String("vendor", t.vendorService.Name()),
		)...,
	)

	// call vendor service to update website
	updateCtx, updateSpan := tr.Start(ctx, "Vendor Service Call")
	err := t.vendorService.Update(updateCtx, &parsedParams.Website)
	updateSpan.End()

	if err != nil {
		updateSpan.SetStatus(codes.Error, err.Error())
		updateSpan.RecordError(err)

		return fmt.Errorf("update website failed: %w", err)
	}

	return nil
}

// todo: include otel ctx to message
func (t *Task) Publish(ctx context.Context, params interface{}) error {
	parsedParams, ok := params.(WebsiteUpdateParams)
	if !ok {
		return fmt.Errorf("invalid params type: %T", params)
	}

	// add traceparent to redis message to enable async trace
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	parsedParams.traceParent = carrier.Get("traceparent")

	err := t.stream.Publish(ctx, parsedParams.ToMap())
	if err != nil {
		return fmt.Errorf("publish website update failed: %w", err)
	}

	return nil
}

func (t *Task) Subscribe(ctx context.Context, ch chan goworkers.Msg) error {
	log.Info().Msg("subscribe website update task")

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
				// we will only process 1 website for each vendor.
				t.sema.Acquire(ctx, 1)
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

func (t *Task) Support(web *model.Website) bool {
	return t.vendorService.Support(web)
}

func (t *Task) Unsubscribe() error {
	return nil
}
