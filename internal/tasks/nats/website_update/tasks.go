package websiteupdate

import (
	"context"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/nats-io/nats.go"
)

type WebsiteUpdateTasks []*WebsiteUpdateTask

func NewTaskSet(nc *nats.Conn, services []vendors.VendorService, rpo repository.Repostory, conf *config.WebsiteConfig) WebsiteUpdateTasks {
	updateTasks := make(WebsiteUpdateTasks, 0, len(services))
	for _, service := range services {
		updateTasks = append(updateTasks, NewTask(nc, service, rpo, conf))
	}

	return updateTasks
}

func (tasks WebsiteUpdateTasks) Publish(ctx context.Context, web *model.Website) ([]string, error) {
	supportedTasks := make([]string, 0, len(tasks))

	// publish for supported service
	for _, t := range tasks {
		if t.Service.Support(web) {
			supportedTasks = append(supportedTasks, t.Service.Name())

			err := t.Publish(ctx, web)
			if err != nil {
				return supportedTasks, err
			}
		}
	}

	if len(supportedTasks) == 0 {
		return nil, ErrNotSupportedWebsite
	}

	return supportedTasks, nil
}
