package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/goworkers"

	// "github.com/htchan/WebHistory/internal/jobs/websiteupdate"
	"github.com/htchan/WebHistory/internal/repository/sqlc"
	websitebatchupdate "github.com/htchan/WebHistory/internal/tasks/website_batch_update"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/website_update"
	"github.com/htchan/WebHistory/internal/utils"
	vendorhelper "github.com/htchan/WebHistory/internal/vendors/helpers"
	shutdown "github.com/htchan/goshutdown"
	"github.com/redis/rueidis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// TODO: move tracer to helper
func tracerProvider(conf config.TraceConfig) (*tracesdk.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(conf.TraceURL)))
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(conf.TraceServiceName),
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

	tp, err := tracerProvider(conf.TraceConfig)
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

	ctx := context.Background()

	pool := goworkers.NewWorkerPool(goworkers.Config{MaxThreads: 10})
	redisClient, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{conf.RedisStreamConfig.Addr},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create redis client")
	}

	updateTasks := websiteupdate.NewTaskSet(redisClient, services, rpo, &conf.WebsiteConfig)
	for _, task := range updateTasks {
		registerErr := pool.Register(ctx, task)
		if registerErr != nil {
			log.Fatal().Err(err).Msg("failed to register task")
		}
	}

	err = pool.Register(ctx, websitebatchupdate.NewTask(redisClient, updateTasks, rpo))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to register task")
	}

	go func() {
		err := pool.Start(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start worker pool")
		}
	}()

	// todo: run shutdown function for each task if necessary
	shutdownHandler.Register("redis stream", func() error {
		redisClient.Close()

		return nil
	})
	shutdownHandler.Register("worker pool", pool.Stop)
	shutdownHandler.Register("database", db.Close)
	shutdownHandler.Register("tracer", func() error {
		return tp.Shutdown(context.Background())
	})

	shutdownHandler.Listen(60 * time.Second)
}
