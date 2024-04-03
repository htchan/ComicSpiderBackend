package main

import (
	"context"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/executor"
	"github.com/htchan/WebHistory/internal/jobs/websiteupdate"
	"github.com/htchan/WebHistory/internal/repository/sqlc"
	"github.com/htchan/WebHistory/internal/utils"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/htchan/WebHistory/internal/vendors/baozimh"
	"github.com/htchan/WebHistory/internal/vendors/kuaikanmanhua"
	"github.com/htchan/WebHistory/internal/vendors/manhuagui"
	"github.com/htchan/WebHistory/internal/vendors/manhuaren"
	"github.com/htchan/WebHistory/internal/vendors/qiman6"
	"github.com/htchan/WebHistory/internal/vendors/u17"
	"github.com/htchan/WebHistory/internal/vendors/webtoons"
	shutdown "github.com/htchan/goshutdown"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
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

	exec := executor.NewExecutor(conf.BinConfig.WorkerExecutorCount)

	cli := &http.Client{Timeout: conf.BinConfig.ClientTimeout}

	services := []vendors.VendorService{}
	for key, cfg := range conf.BinConfig.VendorServiceConfigs {
		switch key {
		case baozimh.Host:
			services = append(services, baozimh.NewVendorService(cli, rpo, &cfg))
		case kuaikanmanhua.Host:
			services = append(services, kuaikanmanhua.NewVendorService(cli, rpo, &cfg))
		case manhuagui.Host:
			services = append(services, manhuagui.NewVendorService(cli, rpo, &cfg))
		case manhuaren.Host:
			services = append(services, manhuaren.NewVendorService(cli, rpo, &cfg))
		case qiman6.Host:
			services = append(services, qiman6.NewVendorService(cli, rpo, &cfg))
		case u17.Host:
			services = append(services, u17.NewVendorService(cli, rpo, &cfg))
		case webtoons.Host:
			services = append(services, webtoons.NewVendorService(cli, rpo, &cfg))
		default:
			log.Error().Str("vendor", key).Msg("unknown vendor")
		}
	}

	// start website update job
	websiteUpdateScheduler := websiteupdate.Setup(rpo, &conf.BinConfig, services)
	exec.Register(websiteUpdateScheduler.Publisher())
	go websiteUpdateScheduler.Start()

	shutdownHandler.Register("websiteupdate.Scheduler", websiteUpdateScheduler.Stop)
	shutdownHandler.Register("executor", exec.Stop)
	shutdownHandler.Register("database", db.Close)
	shutdownHandler.Register("tracer", func() error {
		return tp.Shutdown(context.Background())
	})

	go exec.Start()

	shutdownHandler.Listen(60 * time.Second)
}
