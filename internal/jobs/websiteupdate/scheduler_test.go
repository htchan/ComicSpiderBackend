package websiteupdate

import (
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/executor"
	"github.com/htchan/WebHistory/internal/jobs"
	mockrepo "github.com/htchan/WebHistory/internal/mock/repository"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestNewScheduler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  *Job
		conf *config.WorkerBinConfig
		want *Scheduler
	}{
		{
			name: "happy flow",
			job:  nil,
			conf: &config.WorkerBinConfig{
				ExecAtBeginning: false,
			},
			want: &Scheduler{
				job: nil,
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := NewScheduler(test.job, test.conf)
			assert.Equal(t, test.want.job, got.job)
			assert.Equal(t, test.want.execAtBeginning, test.conf.ExecAtBeginning)
			assert.NotNil(t, got.stop)
			assert.NotNil(t, got.jobChan)
			assert.NotNil(t, got.hostLocks)
		})
	}
}

func TestScheduler_Start(t *testing.T) {
	t.Parallel()

	// TODO: think carefull about what to be tested here
	t.Skip()
}

func Test_calculateNextRunTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args time.Time
		want time.Time
	}{
		{
			name: "output coming Fri if it is monday",
			args: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			want: time.Date(2023, 1, 6, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "output next Fri if it it Thur",
			args: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC),
			want: time.Date(2023, 1, 6, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "output next Fri if it is pass",
			args: time.Date(2023, 1, 6, 4, 0, 0, 0, time.UTC),
			want: time.Date(2023, 1, 13, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "output current Fri if it is not pass",
			args: time.Date(2023, 1, 6, 3, 59, 0, 0, time.UTC),
			want: time.Date(2023, 1, 6, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "output now if it is empty",
			args: time.Time{},
			want: time.Now().UTC().Truncate(time.Second),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := calculateNexRunTime(test.args)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestScheduler_batchDeployUpdateJob(t *testing.T) {
	t.Parallel()

	type jobArgs struct {
		getRepo       func(*gomock.Controller) repository.Repostory
		sleepInterval time.Duration
	}

	tests := []struct {
		name             string
		jobArgs          jobArgs
		wantPublishTotal int
		wantDur          time.Duration
	}{
		{
			name: "happy flow",
			jobArgs: jobArgs{
				getRepo: func(c *gomock.Controller) repository.Repostory {
					rpo := mockrepo.NewMockRepostory(c)
					rpo.EXPECT().FindWebsites().Return([]model.Website{
						{UUID: "1", URL: "http://testing.com/1"},
						{UUID: "2", URL: "http://testing.com/2"},
						{UUID: "3", URL: "http://another_testing.com/2"},
					}, nil)

					return rpo
				},
				sleepInterval: 10 * time.Millisecond,
			},
			wantPublishTotal: 3,
			wantDur:          10 * time.Millisecond,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			scheduler := &Scheduler{
				job: &Job{
					rpo:           test.jobArgs.getRepo(ctrl),
					sleepInterval: test.jobArgs.sleepInterval,
				},
				jobChan:   make(executor.JobTrigger),
				hostLocks: make(map[string]*sync.Mutex),
			}

			go func() {
				publishTotal := 0

				for exec := range scheduler.jobChan {
					exec := exec
					publishTotal += 1

					go func() {
						time.Sleep(scheduler.job.sleepInterval)
						exec.Params.(Params).Cleanup()
					}()
				}

				assert.Equal(t, publishTotal, test.wantPublishTotal)
			}()

			start := time.Now()

			scheduler.batchDeployUpdateJob()
			close(scheduler.jobChan)

			assert.LessOrEqual(t, test.wantDur, time.Since(start))
		})
	}
}

func TestScheduler_Stop(t *testing.T) {
	t.Parallel()

	scheduler := NewScheduler(nil, &config.WorkerBinConfig{})
	err := scheduler.Stop()
	assert.ErrorIs(t, err, nil)
	_, stopOk := <-scheduler.stop
	assert.False(t, stopOk)
	_, jobOk := <-scheduler.jobChan
	assert.False(t, jobOk)
}

func TestScheduler_Publisher(t *testing.T) {
	t.Parallel()

	scheduler := NewScheduler(nil, &config.WorkerBinConfig{})
	assert.Equal(t, scheduler.jobChan, scheduler.Publisher())
}

func TestScheduler_DeployJob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  Params
		before  func(*testing.T, *Scheduler)
		after   func(*testing.T, *Scheduler)
		wantErr error
	}{
		{
			name:   "happy flow",
			params: Params{Web: &model.Website{URL: "http://testing.com"}},
			before: func(t *testing.T, s *Scheduler) {
				go func() {
					data, ok := <-s.jobChan
					assert.True(t, ok)
					assert.Equal(t, &model.Website{URL: "http://testing.com"}, data.Params.(Params).Web)
					data.Params.(Params).Cleanup()
				}()
			},
			after: func(t *testing.T, s *Scheduler) {
				for _, lock := range s.hostLocks {
					defer lock.Unlock()
					lock.Lock()
				}
				close(s.stop)
				close(s.jobChan)
			},
			wantErr: nil,
		},
		{
			name:    "return error if scheduler is stopped",
			params:  Params{},
			before:  func(t *testing.T, s *Scheduler) { close(s.stop) },
			after:   func(t *testing.T, s *Scheduler) { close(s.jobChan) },
			wantErr: jobs.ErrSchedulerStopped,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			scheduler := &Scheduler{
				stop:      make(chan struct{}),
				jobChan:   make(chan *executor.JobExec),
				hostLocks: make(map[string]*sync.Mutex),
			}

			test.before(t, scheduler)
			err := scheduler.DeployJob(test.params)
			assert.ErrorIs(t, err, test.wantErr)
			test.after(t, scheduler)
		})
	}
}
