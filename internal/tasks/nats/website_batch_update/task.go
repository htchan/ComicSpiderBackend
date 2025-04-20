package websitebatchupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/nats/website_update"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type WebsiteBatchUpdateTask struct {
	nc                 *nats.Conn
	websiteUpdateTasks websiteupdate.WebsiteUpdateTasks
	rpo                repository.Repostory
}

func NewTask(
	nc *nats.Conn,
	tasks websiteupdate.WebsiteUpdateTasks,
	rpo repository.Repostory,
) WebsiteBatchUpdateTask {
	return WebsiteBatchUpdateTask{
		nc:                 nc,
		websiteUpdateTasks: tasks,
		rpo:                rpo,
	}
}

func (task WebsiteBatchUpdateTask) subject() string {
	return "web_history.websites.batch_update"
}

func (task *WebsiteBatchUpdateTask) Subscribe(ctx context.Context) (jetstream.ConsumeContext, error) {
	js, err := jetstream.New(task.nc)
	if err != nil {
		return nil, err
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     strings.ReplaceAll(task.subject(), ".", "-"),
		Subjects: []string{task.subject()},
		MaxAge:   time.Hour * 24 * 7,
	})
	if err != nil {
		return nil, err
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:      strings.ReplaceAll(task.subject(), ".", "-"),
		Durable:   strings.ReplaceAll(task.subject(), ".", "-"),
		AckPolicy: jetstream.AckExplicitPolicy,
		AckWait:   time.Minute * 10,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(task.handler)
}

func (task *WebsiteBatchUpdateTask) handler(msg jetstream.Msg) {
	ctx := log.With().
		Str("task", "website-batch-update").
		Str("task_id", hashData(msg.Data())).
		Logger().WithContext(context.Background())

	defer func() {
		ackErr := msg.Ack()
		if ackErr != nil {
			zerolog.Ctx(ctx).Error().Err(ackErr).Msg("ack message failed")
		}
	}()

	tr := otel.Tracer("htchan/WebHistory/website-batch-update")
	ctx, span := tr.Start(ctx, "Website Batch Update")
	defer span.End()

	// load all webstes from db
	_, dbSpan := tr.Start(ctx, "Load Websites From DB")
	defer dbSpan.End()

	websites, err := task.rpo.FindWebsites(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("load website from db failed")
		dbSpan.SetStatus(codes.Error, err.Error())
		dbSpan.RecordError(err)

		return
	}

	dbSpan.End()

	// publish update job for all website
	iterateCtx, iterateSpan := tr.Start(ctx, "Iterate Websites")
	for _, web := range websites {
		task.publishWebsiteUpdateTask(iterateCtx, &web)
	}
	iterateSpan.End()
}

func (task *WebsiteBatchUpdateTask) publishWebsiteUpdateTask(ctx context.Context, web *model.Website) {
	tr := otel.Tracer("htchan/WebHistory/website-batch-update")

	loggingCtx := log.With().
		Str("host", web.Host()).
		Str("website_uuid", web.UUID).
		Str("website_url", web.URL).
		Str("website_title", web.Title).
		Logger().WithContext(ctx)

	websiteCtx, websiteSpan := tr.Start(loggingCtx, "Publish Update Website")
	defer websiteSpan.End()

	supportedTasks := make([]string, 0, len(task.websiteUpdateTasks))

	supportedTasks, err := task.websiteUpdateTasks.Publish(websiteCtx, web)
	if err != nil {
		zerolog.Ctx(websiteCtx).Error().Err(err).
			Msg("publish website update task failed")
	}

	websiteSpan.SetAttributes(
		append(
			web.OtelAttributes(),
			attribute.StringSlice("support_tasks", supportedTasks),
		)...,
	)

	if len(supportedTasks) == 0 {
		zerolog.Ctx(ctx).Warn().
			Msg("no support task for website")
	} else if len(supportedTasks) > 1 {
		zerolog.Ctx(ctx).Warn().
			Strs("support_task_names", supportedTasks).
			Msg("multiple support task for website")
	}
}

func hashData(input []byte) string {
	// Create a new SHA256 hash
	hash := sha256.New()

	// Write the input string to the hash
	hash.Write(input)

	// Get the hashed bytes
	hashedBytes := hash.Sum(nil)

	// Convert the hashed bytes to a hexadecimal string
	return hex.EncodeToString(hashedBytes)
}
