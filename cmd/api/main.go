package main

import (
	"context"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
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

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository/sqlc"
	"github.com/htchan/WebHistory/internal/router/website"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/nats/website_update"
	"github.com/htchan/WebHistory/internal/utils"
	vendorhelper "github.com/htchan/WebHistory/internal/vendors/helpers"
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

	conf, err := config.LoadAPIConfig()
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

	db, err := utils.OpenDatabase(&conf.DatabaseConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open database")
	}

	defer db.Close()

	rpo := sqlc.NewRepo(db, &conf.WebsiteConfig)

	nc, err := utils.ConnectNatsQueue(&conf.NatsConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to nats queue")
	}

	services, err := vendorhelper.NewServiceSet(nil, rpo, conf.BinConfig.VendorServiceConfigs)
	if err != nil {
		log.Fatal().Err(err).Msg("create vendor services failed")
	}

	shutdown.LogEnabled = true
	shutdownHandler := shutdown.New(syscall.SIGINT, syscall.SIGTERM)

	websiteUpdateTasks := websiteupdate.NewTaskSet(nc, services, rpo, &conf.WebsiteConfig)

	r := chi.NewRouter()
	website.AddRoutes(r, rpo, websiteUpdateTasks, conf)

	server := http.Server{
		Addr:         conf.BinConfig.Addr,
		ReadTimeout:  conf.BinConfig.ReadTimeout,
		WriteTimeout: conf.BinConfig.WriteTimeout,
		IdleTimeout:  conf.BinConfig.IdleTimeout,
		Handler:      r,
	}

	go func() {
		log.Debug().Msg("start http server")

		if err := server.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Msg("backend stopped")
		}
	}()

	shutdownHandler.Register("api server", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		server.Shutdown(ctx)

		return nil
	})
	shutdownHandler.Register("nats connection", func() error {
		nc.Close()

		return nil
	})
	shutdownHandler.Register("database", db.Close)
	shutdownHandler.Register("tracer", func() error {
		return tp.Shutdown(context.Background())
	})

	shutdownHandler.Listen(60 * time.Second)
}
