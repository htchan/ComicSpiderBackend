package executor

import (
	"context"
	"reflect"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// TODO: add missing testcases
type ExecutorImpl struct {
	executorCount int
	exit          chan interface{}
	workerWg      sync.WaitGroup
	publisherWg   sync.WaitGroup
	jobPublishers []JobTrigger
}

var _ Executor = (*ExecutorImpl)(nil)

func NewExecutor(n int) Executor {
	return &ExecutorImpl{
		executorCount: n,
		exit:          make(chan interface{}),
	}
}

func (executor *ExecutorImpl) Register(trigger ...JobTrigger) {
	executor.jobPublishers = append(executor.jobPublishers, trigger...)
}

func (executor *ExecutorImpl) Start() {
	jobQueue := make(JobTrigger, executor.executorCount)

	for _, publisher := range executor.jobPublishers {
		publisher := publisher
		executor.publisherWg.Add(1)
		go func() {
			defer executor.publisherWg.Done()
			for {
				jobExec, ok := <-publisher
				if !ok {
					return
				}

				jobQueue <- jobExec
			}
		}()
	}

	for i := 0; i < executor.executorCount; i++ {
		executor.workerWg.Add(1)

		go func() {
			defer executor.workerWg.Done()
			for {
				select {
				case <-executor.exit:
					return
				case jobExec, ok := <-jobQueue:
					if !ok {
						continue
					}

					job, params := jobExec.Job, jobExec.Params

					jobUUID := uuid.Must(uuid.NewUUID()).String()
					ctx := log.With().
						Str("name", reflect.ValueOf(job).Type().String()).
						Str("job_uuid", jobUUID).
						Interface("params", params).
						Logger().WithContext(context.Background())

					ctx = context.WithValue(ctx, "job_uuid", jobUUID)

					err := job.Execute(ctx, params)
					if err != nil {
						zerolog.Ctx(ctx).Error().Err(err).Msg("execute job fail")
					} else {
						zerolog.Ctx(ctx).Info().Msg("execute job success")
					}
				}
			}
		}()
	}

	executor.publisherWg.Wait()
	close(jobQueue)
	executor.workerWg.Wait()
}

func (executor *ExecutorImpl) Stop() error {
	executor.publisherWg.Wait()

	executor.exit <- nil
	close(executor.exit)

	executor.workerWg.Wait()

	return nil
}
