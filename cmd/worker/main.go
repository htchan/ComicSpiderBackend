package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/nats-io/nats.go/jetstream"

	// "github.com/htchan/WebHistory/internal/jobs/websiteupdate"
	"github.com/htchan/WebHistory/internal/repository/sqlc"
	websitebatchupdate "github.com/htchan/WebHistory/internal/tasks/nats/website_batch_update"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/nats/website_update"
	"github.com/htchan/WebHistory/internal/utils"
	vendorhelper "github.com/htchan/WebHistory/internal/vendors/helpers"

	shutdown "github.com/htchan/goshutdown"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func otelProvider(conf config.TraceConfig) (*tracesdk.TracerProvider, error) {
	exp, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(conf.OtelURL),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(conf.OtelServiceName),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}

func main() {
	outputPath := os.Getenv("OUTPUT_PATH")
	if outputPath != "" {
		writer, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err == nil {
			log.Logger = log.Logger.Output(writer)
			defer writer.Close()
		} else {
			log.Fatal().
				Err(err).
				Str("output_path", outputPath).
				Msg("set logger output failed")
		}
	}

	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.99999Z07:00"

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	conf, err := config.LoadWorkerConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("load config failed")
	}

	tp, err := otelProvider(conf.TraceConfig)
	if err != nil {
		log.Error().Err(err).Msg("init tracer failed")
	}

	if err = utils.Migrate(&conf.DatabaseConfig); err != nil {
		log.Fatal().Err(err).Msg("failed to migrate")
	}

	shutdown.LogEnabled = true
	shutdownHandler := shutdown.New(syscall.SIGINT, syscall.SIGTERM)

	db, err := utils.OpenDatabase(&conf.DatabaseConfig)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to open database")
	}

	rpo := sqlc.NewRepo(db, &conf.WebsiteConfig)

	cli := &http.Client{Timeout: conf.BinConfig.ClientTimeout}

	services, err := vendorhelper.NewServiceSet(cli, rpo, conf.BinConfig.VendorServiceConfigs)
	if err != nil {
		log.Fatal().Err(err).Msg("create vendor services failed")
	}

	nc, err := utils.ConnectNatsQueue(&conf.NatsConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to nats server")
	}

	ctx := context.Background()

	updateTasks := make([]jetstream.ConsumeContext, 0, len(services))
	websiteUpdateTasks := websiteupdate.NewTaskSet(nc, services, rpo, &conf.WebsiteConfig)
	for _, task := range websiteUpdateTasks {
		consumer, err := task.Subscribe(ctx)
		if err != nil {
			log.Fatal().Err(err).
				Str("task", "website-update").
				Msg("failed to subscribe to nats server")
		}

		updateTasks = append(updateTasks, consumer)
	}
	batchUpdateTask := websitebatchupdate.NewTask(nc, websiteUpdateTasks, rpo)
	batchConsumer, err := batchUpdateTask.Subscribe(ctx)
	if err != nil {
		log.Fatal().Err(err).
			Str("task", "website-batch-update").
			Msg("failed to subscribe to nats server")
	}

	shutdownHandler.Register("batch update task", func() error {
		batchConsumer.Stop()

		return nil
	})
	for _, updateTask := range updateTasks {
		shutdownHandler.Register("update task", func() error {
			updateTask.Stop()

			return nil
		})
	}
	shutdownHandler.Register("nats connect", func() error {
		nc.Close()

		return nil
	})
	shutdownHandler.Register("database", db.Close)
	shutdownHandler.Register("tracer", func() error {
		return tp.Shutdown(context.Background())
	})

	shutdownHandler.Listen(60 * time.Second)
}
