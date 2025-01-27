package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/redis/rueidis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/go-chi/chi/v5"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository/sqlc"
	"github.com/htchan/WebHistory/internal/router/website"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/website_update"
	"github.com/htchan/WebHistory/internal/utils"
	vendorhelper "github.com/htchan/WebHistory/internal/vendors/helpers"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
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

	services, err := vendorhelper.NewServiceSet(nil, rpo, conf.BinConfig.VendorServiceConfigs)
	if err != nil {
		log.Fatal().Err(err).Msg("create vendor services failed")
	}

	redisClient, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{conf.RedisStreamConfig.Addr},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create redis client")
	}

	updateTasks := websiteupdate.NewTaskSet(redisClient, services, rpo, &conf.WebsiteConfig)

	r := chi.NewRouter()
	website.AddRoutes(r, rpo, conf, updateTasks)

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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	log.Debug().Msg("received interrupt signal")

	// Setup graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	tp.Shutdown(ctx)
}
