package websiteupdate

import (
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
)

// TODO: add missing testcases
func Setup(rpo repository.Repostory, conf *config.WorkerBinConfig, services []vendors.VendorService) *Scheduler {
	websiteUpdateJob := NewJob(rpo, conf.WebsiteUpdateSleepInterval, services)
	scheduler := NewScheduler(websiteUpdateJob, conf)

	return scheduler
}
