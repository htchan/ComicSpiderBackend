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

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository/sqlc"
	"github.com/htchan/WebHistory/internal/router/website"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/website_update"
	"github.com/htchan/WebHistory/internal/utils"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/htchan/WebHistory/internal/vendors/baozimh"
	"github.com/htchan/WebHistory/internal/vendors/kuaikanmanhua"
	"github.com/htchan/WebHistory/internal/vendors/manhuagui"
	"github.com/htchan/WebHistory/internal/vendors/manhuaren"
	"github.com/htchan/WebHistory/internal/vendors/qiman6"
	"github.com/htchan/WebHistory/internal/vendors/u17"
	"github.com/htchan/WebHistory/internal/vendors/webtoons"

	"github.com/go-chi/chi/v5"
)

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

	if err = utils.Migrate(&conf.DatabaseConfig); err != nil {
		log.Fatal().Err(err).Msg("failed to migrate")
	}

	db, err := utils.OpenDatabase(&conf.DatabaseConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open database")
	}

	defer db.Close()

	rpo := sqlc.NewRepo(db, &conf.WebsiteConfig)

	services := []vendors.VendorService{}
	for key, cfg := range conf.BinConfig.VendorServiceConfigs {
		switch key {
		case baozimh.Host:
			services = append(services, baozimh.NewVendorService(nil, rpo, &cfg))
		case kuaikanmanhua.Host:
			services = append(services, kuaikanmanhua.NewVendorService(nil, rpo, &cfg))
		case manhuagui.Host:
			services = append(services, manhuagui.NewVendorService(nil, rpo, &cfg))
		case manhuaren.Host:
			services = append(services, manhuaren.NewVendorService(nil, rpo, &cfg))
		case qiman6.Host:
			services = append(services, qiman6.NewVendorService(nil, rpo, &cfg))
		case u17.Host:
			services = append(services, u17.NewVendorService(nil, rpo, &cfg))
		case webtoons.Host:
			services = append(services, webtoons.NewVendorService(nil, rpo, &cfg))
		default:
			log.Error().Str("vendor", key).Msg("unknown vendor")

			return
		}
	}

	redisClient, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{conf.RedisStreamConfig.Addr},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create redis client")
	}

	var updateTasks []*websiteupdate.Task
	for _, service := range services {
		updateTasks = append(updateTasks, websiteupdate.NewTask(redisClient, service, rpo, &conf.WebsiteConfig))
	}

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
}
