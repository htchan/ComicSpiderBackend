package websiteupdate

import (
	"context"
	"sync"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/executor"
	"github.com/htchan/WebHistory/internal/jobs"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// TODO: add missing testcases
type Scheduler struct {
	job             *Job
	stop            chan struct{}
	jobChan         executor.JobTrigger
	hostLocks       map[string]*sync.Mutex
	hostLocksMutex  sync.Mutex
	publisherWg     sync.WaitGroup
	execAtBeginning bool
}

func NewScheduler(job *Job, conf *config.WorkerBinConfig) *Scheduler {
	return &Scheduler{
		job:             job,
		stop:            make(chan struct{}),
		jobChan:         make(executor.JobTrigger),
		hostLocks:       make(map[string]*sync.Mutex),
		execAtBeginning: conf.ExecAtBeginning,
	}
}

func (scheduler *Scheduler) Start() {
	// calculate next run time
	var lastRunTime time.Time
	if scheduler.execAtBeginning {
		lastRunTime = time.Time{}
	} else {
		lastRunTime = time.Now().UTC().Truncate(time.Second)
	}

	for {
		select {
		case <-scheduler.stop:
			return
		case <-time.NewTimer(time.Until(calculateNexRunTime(lastRunTime))).C:
			lastRunTime = time.Now().UTC().Truncate(time.Second)
			scheduler.batchDeployUpdateJob()
		}
	}
}

// run at specific time at every friday
func calculateNexRunTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC().Truncate(time.Second)
	}

	nDaysLater := int(time.Friday - t.Weekday())
	if nDaysLater < 0 || (nDaysLater == 0 && t.Hour() >= 4) {
		nDaysLater += 7
	}

	result := t.Truncate(24*time.Hour).AddDate(0, 0, nDaysLater).Add(4 * time.Hour)

	return result
}

func (scheduler *Scheduler) batchDeployUpdateJob() {
	tr := otel.Tracer("htchan/WebHistory/update-jobs")

	ctx, span := tr.Start(context.Background(), "batch-update")
	defer span.End()

	logger := log.With().
		Str("scheduler", "websiteupdate").
		Str("operation", "batch-update").
		Logger()

	logger.Info().Msg("batch update start")

	hostSpanMap := make(map[string]trace.Span)

	// list all websites
	webs, err := scheduler.job.rpo.FindWebsites()
	if err != nil {
		logger.Error().Err(err).Msg("failed to list websites")

		return
	}

	var deployWebWg sync.WaitGroup

	for _, web := range webs {
		// deploy job to update website
		web := web

		hostSpan, ok := hostSpanMap[web.Host()]
		if !ok {
			_, hostSpan = tr.Start(ctx, web.Host())
			defer hostSpan.End()

			hostSpanMap[web.Host()] = hostSpan
		}

		hostSpanContext := hostSpan.SpanContext()

		deployWebWg.Add(1)

		go func() {
			defer deployWebWg.Done()
			err := scheduler.DeployJob(Params{Web: &web, SpanContext: &hostSpanContext})
			if err != nil {
				logger.Error().Err(err).Str("website", web.URL).
					Msg("failed to deploy job to update website")
			}
		}()
	}

	deployWebWg.Wait()
}

func (scheduler *Scheduler) Stop() error {
	close(scheduler.stop)
	scheduler.publisherWg.Wait()
	close(scheduler.jobChan)
	return nil
}

func (scheduler *Scheduler) Publisher() executor.JobTrigger {
	return scheduler.jobChan
}

func (scheduler *Scheduler) DeployJob(params Params) error {
	// return error if the scheduler was stopped
	select {
	case <-scheduler.stop:
		return jobs.ErrSchedulerStopped
	default:
	}

	host := params.Web.Host()

	// init executionLock
	scheduler.hostLocksMutex.Lock()
	lock, ok := scheduler.hostLocks[host]
	if !ok {
		lock = &sync.Mutex{}
		scheduler.hostLocks[host] = lock
	}
	scheduler.hostLocksMutex.Unlock()

	if params.Cleanup == nil {
		params.Cleanup = func() { lock.Unlock() }
	} else {
		cleanup := params.Cleanup
		params.Cleanup = func() {
			lock.Unlock()
			cleanup()
		}
	}

	scheduler.publisherWg.Add(1)
	defer scheduler.publisherWg.Done()

	lock.Lock()
	scheduler.jobChan <- &executor.JobExec{
		Job:    scheduler.job,
		Params: params,
	}

	return nil
}
