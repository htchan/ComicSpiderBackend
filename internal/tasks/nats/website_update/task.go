package websiteupdate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type WebsiteUpdateTask struct {
	nc          *nats.Conn
	Service     vendors.VendorService
	rpo         repository.Repostory
	websiteConf *config.WebsiteConfig
}

func NewTask(
	nc *nats.Conn,
	service vendors.VendorService,
	rpo repository.Repostory,
	websiteConf *config.WebsiteConfig,
) *WebsiteUpdateTask {
	return &WebsiteUpdateTask{
		nc:          nc,
		Service:     service,
		rpo:         rpo,
		websiteConf: websiteConf,
	}
}

func (task *WebsiteUpdateTask) subject() string {
	return fmt.Sprintf("web_history.websites.update.%s", strings.ReplaceAll(task.Service.Name(), ".", "_"))
}

func (task *WebsiteUpdateTask) Publish(
	ctx context.Context,
	web *model.Website,
) error {
	params := WebsiteUpdateParams{
		Website: *web,
	}

	data, err := params.ToData(ctx)
	if err != nil {
		return err
	}

	err = task.nc.Publish(task.subject(), data)
	if err != nil {
		return err
	}

	return nil
}

func (task *WebsiteUpdateTask) Subscribe(ctx context.Context) (jetstream.ConsumeContext, error) {
	js, err := jetstream.New(task.nc)
	if err != nil {
		return nil, fmt.Errorf("init jetstream fail: %v", err)
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     strings.ReplaceAll(task.subject(), ".", "-"),
		Subjects: []string{task.subject()},
		MaxAge:   time.Hour * 24 * 7,
	})
	if err != nil {
		return nil, fmt.Errorf("create / update stream fail: %v", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:      strings.ReplaceAll(task.subject(), ".", "-"),
		Durable:   strings.ReplaceAll(task.subject(), ".", "-"),
		AckPolicy: jetstream.AckExplicitPolicy,
		AckWait:   time.Minute * 10,
	})
	if err != nil {
		return nil, fmt.Errorf("create / update consumer fail: %v", err)
	}

	return consumer.Consume(task.handler)
}

func (task *WebsiteUpdateTask) Validate(ctx context.Context, params *WebsiteUpdateParams) error {
	tr := otel.Tracer("htchan/WebHistory/website-update")

	// validate params
	_, validateSpan := tr.Start(ctx, "Validate Params")
	defer validateSpan.End()

	if !task.Service.Support(&params.Website) {
		validateSpan.SetStatus(codes.Error, ErrNotSupportedWebsite.Error())
		validateSpan.RecordError(ErrNotSupportedWebsite)

		return ErrNotSupportedWebsite
	}

	return nil
}

func (task *WebsiteUpdateTask) handler(msg jetstream.Msg) {
	ctx := log.With().
		Str("task", "website-update").
		Str("vendor", task.Service.Name()).
		Logger().WithContext(context.Background())

	defer func() {
		ackErr := msg.Ack()
		if ackErr != nil {
			zerolog.Ctx(ctx).Error().Err(ackErr).Msg("ack failed")
		}
	}()

	tr := otel.Tracer("htchan/WebHistory/website-update")

	// parse message body
	ctx, params, err := ParamsFromData(ctx, msg.Data(), task.websiteConf)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).
			Str("data", string(msg.Data())).
			Msg("failed to parse message body")

		return
	}

	ctx, span := tr.Start(ctx, "Website Update")
	defer span.End()

	span.SetAttributes(
		append(
			params.Website.OtelAttributes(),
			attribute.String("vendor", task.Service.Name()),
		)...,
	)

	// validate params
	validateErr := task.Validate(ctx, params)
	if validateErr != nil {
		zerolog.Ctx(ctx).Error().Err(validateErr).Msg("validate params failed")

		return
	}

	// sleep after update
	defer func() {
		time.Sleep(time.Second)
	}()

	// call vendor service to update website
	updateCtx, updateSpan := tr.Start(ctx, "Vendor Service Call")
	defer updateSpan.End()

	updateErr := task.Service.Update(updateCtx, &params.Website)
	if updateErr != nil {
		zerolog.Ctx(ctx).Error().Err(updateErr).Msg("update website failed")
		updateSpan.SetStatus(codes.Error, updateErr.Error())
		updateSpan.RecordError(updateErr)

		return
	}

	updateSpan.End()
}
