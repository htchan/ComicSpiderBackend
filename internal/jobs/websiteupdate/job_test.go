package websiteupdate

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/jobs"
	mockrepo "github.com/htchan/WebHistory/internal/mock/repository"
	mockvendor "github.com/htchan/WebHistory/internal/mock/vendor"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/stretchr/testify/assert"
)

func TestNewJob(t *testing.T) {
	t.Parallel()

	type args struct {
		rpo           repository.Repostory
		sleepInterval time.Duration
		services      []vendors.VendorService
	}

	tests := []struct {
		name string
		args args
		want *Job
	}{
		{
			name: "happy flow",
			args: args{rpo: nil, sleepInterval: 5 * time.Second, services: nil},
			want: &Job{rpo: nil, sleepInterval: 5 * time.Second, vendorServices: nil},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := NewJob(test.args.rpo, test.args.sleepInterval, test.args.services)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestJob_Execute(t *testing.T) {
	t.Parallel()

	type jobArgs struct {
		getRepo       func(*gomock.Controller) repository.Repostory
		getServices   func(*gomock.Controller) []vendors.VendorService
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

					return rpo
				},
				getServices: func(ctrl *gomock.Controller) []vendors.VendorService {
					service := mockvendor.NewMockVendorService(ctrl)
					service.EXPECT().Support(&model.Website{
						UUID: "uuid", URL: "https://google.com",
						Conf: &config.WebsiteConfig{Separator: ","},
					}).Return(true)
					service.EXPECT().Update(gomock.Any(), &model.Website{
						UUID: "uuid", URL: "https://google.com",
						Conf: &config.WebsiteConfig{Separator: ","},
					}).Return(nil)

					return []vendors.VendorService{service}
				},
				sleepInterval: 10 * time.Millisecond,
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
			wantSleep: 10 * time.Millisecond,
			wantError: nil,
		},
		{
			name: "invalid params type",
			jobArgs: jobArgs{
				getRepo: func(ctrl *gomock.Controller) repository.Repostory {
					rpo := mockrepo.NewMockRepostory(ctrl)

					return rpo
				},
				getServices: func(ctrl *gomock.Controller) []vendors.VendorService {
					services := mockvendor.NewMockVendorService(ctrl)

					return []vendors.VendorService{services}
				},
				sleepInterval: 10 * time.Millisecond,
			},
			args: args{
				getCtx: func() context.Context { return context.Background() },
				params: "1",
			},
			wantSleep: 2 * time.Nanosecond,
			wantError: jobs.ErrInvalidParams,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			job := NewJob(test.jobArgs.getRepo(ctrl), test.jobArgs.sleepInterval, test.jobArgs.getServices(ctrl))

			start := time.Now()
			err := job.Execute(test.args.getCtx(), test.args.params)
			assert.LessOrEqual(t, test.wantSleep, time.Since(start))

			assert.ErrorIs(t, err, test.wantError)
		})
	}
}
