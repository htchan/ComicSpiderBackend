package websiteupdate

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/jobs"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/repository/mockrepo"
	"github.com/stretchr/testify/assert"
)

func TestNewJob(t *testing.T) {
	t.Parallel()

	type args struct {
		rpo           repository.Repostory
		sleepInterval time.Duration
	}

	tests := []struct {
		name string
		args args
		want *Job
	}{
		{
			name: "happy flow",
			args: args{rpo: nil, sleepInterval: 5 * time.Second},
			want: &Job{rpo: nil, sleepInterval: 5 * time.Second},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := NewJob(test.args.rpo, test.args.sleepInterval)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestJob_Execute(t *testing.T) {
	t.Parallel()

	type jobArgs struct {
		getRepo       func(*gomock.Controller) repository.Repostory
		sleepInterval time.Duration
	}

	type args struct {
		getCtx func() context.Context
		params interface{}
	}

	tests := []struct {
		name      string
		jobArgs   jobArgs
		args      args
		wantSleep time.Duration
		wantError error
	}{
		// TODO: create interface and mock service to speed up this test
		{
			name: "happy flow",
			jobArgs: jobArgs{
				getRepo: func(c *gomock.Controller) repository.Repostory {
					rpo := mockrepo.NewMockRepostory(c)
					rpo.EXPECT().FindWebsiteSetting("google.com").
						Return(&model.WebsiteSetting{}, nil)

					return rpo
				},
				sleepInterval: 100 * time.Millisecond,
			},
			args: args{
				getCtx: func() context.Context {
					return context.WithValue(context.Background(), "job_uuid", "uuid")
				},
				params: Params{
					Web: &model.Website{
						UUID: "uuid", URL: "https://google.com",
						Conf: &config.WebsiteConfig{Separator: ","},
					},
					Cleanup: func() {},
				},
			},
			wantSleep: 100 * time.Millisecond,
			wantError: nil,
		},
		{
			name: "invalid params type",
			jobArgs: jobArgs{
				getRepo: func(ctrl *gomock.Controller) repository.Repostory {
					rpo := mockrepo.NewMockRepostory(ctrl)

					return rpo
				},
				sleepInterval: 100 * time.Millisecond,
			},
			args: args{
				getCtx: func() context.Context { return context.Background() },
				params: "1",
			},
			wantError: jobs.ErrInvalidParams,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			job := NewJob(test.jobArgs.getRepo(ctrl), test.jobArgs.sleepInterval)

			start := time.Now()
			err := job.Execute(test.args.getCtx(), test.args.params)
			assert.LessOrEqual(t, test.wantSleep, time.Since(start))

			assert.ErrorIs(t, err, test.wantError)
		})
	}
}
