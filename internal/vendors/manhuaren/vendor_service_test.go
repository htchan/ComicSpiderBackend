package manhuaren

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	mockrepo "github.com/htchan/WebHistory/internal/mock/repository"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/htchan/goclient"
	"github.com/htchan/goclient/middlewares/retry"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

func TestNewVendorService(t *testing.T) {
	t.Parallel()

	type params struct {
		cli  *http.Client
		repo repository.Repostory
		cfg  *config.VendorServiceConfig
	}

	tests := []struct {
		name   string
		params params
		want   *VendorService
	}{
		{
			name: "happy flow",
			params: params{
				cli:  nil,
				repo: nil,
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 10,
					FetchInterval:  10 * time.Second,
				},
			},
			want: &VendorService{
				cli:  nil,
				repo: nil,
				lock: semaphore.NewWeighted(10),
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 10,
					FetchInterval:  10 * time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			get := NewVendorService(tt.params.cli, tt.params.repo, tt.params.cfg)
			assert.Equal(t, tt.want.repo, get.repo)
			assert.Equal(t, tt.want.lock, get.lock)
			assert.Equal(t, tt.want.cfg, get.cfg)
		})
	}
}

func TestVendorService_fetchWebsite(t *testing.T) {
	t.Parallel()

	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("failed"))
		} else if r.URL.Path == "/success" {
			w.Write([]byte("success"))
		} else {
			w.Write([]byte("unknown"))
		}
	}))

	t.Cleanup(func() { serv.Close() })

	unitDuration := 9 * time.Millisecond

	tests := []struct {
		name            string
		serv            *VendorService
		getCtx          func() context.Context
		web             *model.Website
		wantBody        string
		wantError       error
		expectTimeTaken time.Duration
	}{
		{
			name: "send request success",
			serv: &VendorService{
				cli: goclient.NewClient(
					goclient.WithMiddlewares(
						retry.NewRetryMiddleware(
							1,
							retry.RetryForError,
							retry.StaticRetryInterval(0),
						),
						vendors.RaiseStatusCodeErrorMiddleware,
					),
				),
				repo: nil,
				lock: semaphore.NewWeighted(1),
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 1,
					FetchInterval:  10 * time.Millisecond,
					MaxRetry:       1,
				},
			},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{
				URL: serv.URL + "/success",
			},
			wantBody:        "success",
			expectTimeTaken: unitDuration,
		},
		{
			name: "send request failed",
			serv: &VendorService{
				cli: goclient.NewClient(
					goclient.WithMiddlewares(
						retry.NewRetryMiddleware(
							3,
							retry.RetryForError,
							retry.LinearRetryInterval(5*time.Millisecond),
						),
						vendors.RaiseStatusCodeErrorMiddleware,
					),
				),
				repo: nil,
				lock: semaphore.NewWeighted(1),
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 1,
					FetchInterval:  5 * time.Millisecond,
					MaxRetry:       2,
					RetryInterval:  5 * time.Millisecond,
				},
			},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{
				URL: serv.URL + "/fail",
			},
			wantError:       vendors.ErrInvalidStatusCode,
			expectTimeTaken: 2 * unitDuration,
		},
		{
			name: "cancelled context",
			serv: &VendorService{
				cli: goclient.NewClient(
					goclient.WithMiddlewares(
						retry.NewRetryMiddleware(
							1,
							retry.RetryForError,
							retry.StaticRetryInterval(0),
						),
						vendors.RaiseStatusCodeErrorMiddleware,
					),
				),
				repo: nil,
				lock: semaphore.NewWeighted(1),
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 1,
					FetchInterval:  10 * time.Millisecond,
					MaxRetry:       1,
				},
			},
			getCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				return ctx
			},
			web: &model.Website{
				URL: serv.URL + "/success",
			},
			wantError:       context.Canceled,
			expectTimeTaken: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := tt.getCtx()
			start := time.Now()
			body, err := tt.serv.fetchWebsite(ctx, tt.web)
			assert.LessOrEqual(t, tt.expectTimeTaken, time.Since(start).Truncate(unitDuration))
			assert.Equal(t, tt.wantBody, body)
			assert.ErrorIs(t, err, tt.wantError)
			assert.Equal(t, true, tt.serv.lock.TryAcquire(1))
		})
	}
}

func TestVendorService_isUpdated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		serv    *VendorService
		getCtx  func() context.Context
		web     *model.Website
		body    string
		want    bool
		wantWeb *model.Website
	}{
		{
			name: "title update from empty to some value",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{Conf: &config.WebsiteConfig{Separator: "\n"}},
			body: `<html>
				<head><title>title</title></head>
				<div class="detail-list-title">
					<span class="detail-list-title-3">2021-07-30 </span>
				</div>
			</html>`,
			want: true,
			wantWeb: &model.Website{
				Title:      "title",
				UpdateTime: time.Date(2021, 07, 30, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
		},
		{
			name: "title not update if it is not empty",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{
				Title:      "title",
				UpdateTime: time.Date(2021, 07, 30, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			body: `<html>
				<head><title>title</title></head>
				<div class="detail-list-title">
					<span class="detail-list-title-3">2021-07-30 </span>
				</div>
			</html>`,
			want: false,
			wantWeb: &model.Website{
				Title:      "title",
				UpdateTime: time.Date(2021, 07, 30, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
		},
		{
			name: "date updated by a specific date",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{Conf: &config.WebsiteConfig{Separator: "\n"}},
			body: `<html>
				<div class="detail-list-title">
					<span class="detail-list-title-3">2021-07-30 </span>
				</div>
			</html>`,
			want: true,
			wantWeb: &model.Website{
				UpdateTime: time.Date(2021, 7, 30, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
		},
		{
			name: "date updated by a year level related date",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			body: `<html>
				<div class="detail-list-title">
					<span class="detail-list-title-3">07月30号 </span>
				</div>
			</html>`,
			want: true,
			wantWeb: &model.Website{
				UpdateTime: time.Date(time.Now().Year(), 7, 30, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
		},
		{
			name: "date updated at today",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			body: `<html>
				<div class="detail-list-title">
					<span class="detail-list-title-3">今天 18:01 </span>
				</div>
			</html>`,
			want: true,
			wantWeb: &model.Website{
				UpdateTime: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
		},
		{
			name: "date updated at yesterday",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			body: `<html>
				<div class="detail-list-title">
					<span class="detail-list-title-3">昨天 18:01 </span>
				</div>
			</html>`,
			want: true,
			wantWeb: &model.Website{
				UpdateTime: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()-1, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
		},
		{
			name: "date updated at the day before yesterday",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			body: `<html>
				<div class="detail-list-title">
					<span class="detail-list-title-3">前天 18:01 </span>
				</div>
			</html>`,
			want: true,
			wantWeb: &model.Website{
				UpdateTime: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()-2, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := tt.getCtx()
			get := tt.serv.isUpdated(ctx, tt.web, tt.body)
			assert.Equal(t, tt.want, get)
			assert.Equal(t, tt.wantWeb, tt.web)
		})
	}
}

func TestVendorService_Support(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		serv *VendorService
		web  *model.Website
		want bool
	}{
		{
			name: "support host www.manhuaren.com",
			serv: &VendorService{},
			web:  &model.Website{URL: "https://www.manhuaren.com/testing"},
			want: true,
		},
		{
			name: "not support host",
			serv: &VendorService{},
			web:  &model.Website{URL: "https://example.com/testing"},
			want: false,
		},
		{
			name: "not support empty website",
			serv: &VendorService{},
			web:  &model.Website{},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			get := tt.serv.Support(tt.web)
			assert.Equal(t, tt.want, get)
		})
	}
}

func TestVendorService_Update(t *testing.T) {
	t.Parallel()

	testError := fmt.Errorf("testing")
	testClient := goclient.NewClient(
		goclient.WithMiddlewares(
			retry.NewRetryMiddleware(
				1,
				retry.RetryForError,
				retry.StaticRetryInterval(0),
			),
			vendors.RaiseStatusCodeErrorMiddleware,
		),
	)

	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusBadRequest)
		} else if r.URL.Path == "/success" {
			w.Write([]byte(`<html>
				<head><title>title</title></head>
				<div class="detail-list-title">
					<span class="detail-list-title-3">2021-07-30 </span>
				</div>
			</html>`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(func() { serv.Close() })

	tests := []struct {
		name    string
		serv    *VendorService
		getCtx  func() context.Context
		getRepo func(ctrl *gomock.Controller) repository.Repostory
		web     *model.Website
		wantWeb *model.Website
		wantErr error
	}{
		{
			name: "update web successfully",
			serv: &VendorService{
				cli:  testClient,
				lock: semaphore.NewWeighted(1),
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 1,
					MaxRetry:       1,
				},
			},
			getCtx: func() context.Context {
				return context.Background()
			},
			getRepo: func(ctrl *gomock.Controller) repository.Repostory {
				repo := mockrepo.NewMockRepostory(ctrl)
				repo.EXPECT().UpdateWebsite(gomock.Any(), &model.Website{
					URL:        serv.URL + "/success",
					Title:      "title",
					UpdateTime: time.Date(2021, 7, 30, 0, 0, 0, 0, time.UTC),
					Conf:       &config.WebsiteConfig{Separator: "\n"},
				}).Return(nil)

				return repo
			},
			web: &model.Website{
				URL:  serv.URL + "/success",
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			wantWeb: &model.Website{
				URL:        serv.URL + "/success",
				Title:      "title",
				UpdateTime: time.Date(2021, 7, 30, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			wantErr: nil,
		},
		{
			name: "fetch info but not update web",
			serv: &VendorService{
				cli:  testClient,
				lock: semaphore.NewWeighted(1),
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 1,
					MaxRetry:       1,
				},
			},
			getCtx: func() context.Context {
				return context.Background()
			},
			getRepo: func(ctrl *gomock.Controller) repository.Repostory {
				return mockrepo.NewMockRepostory(ctrl)
			},
			web: &model.Website{
				URL:        serv.URL + "/success",
				Title:      "title",
				UpdateTime: time.Date(2021, 7, 30, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			wantWeb: &model.Website{
				URL:        serv.URL + "/success",
				Title:      "title",
				UpdateTime: time.Date(2021, 7, 30, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			wantErr: nil,
		},
		{
			name: "repo returning error",
			serv: &VendorService{
				cli:  testClient,
				lock: semaphore.NewWeighted(1),
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 1,
					MaxRetry:       1,
				},
			},
			getCtx: func() context.Context {
				return context.Background()
			},
			getRepo: func(ctrl *gomock.Controller) repository.Repostory {
				repo := mockrepo.NewMockRepostory(ctrl)
				repo.EXPECT().UpdateWebsite(gomock.Any(), &model.Website{
					URL:        serv.URL + "/success",
					Title:      "title",
					UpdateTime: time.Date(2021, 7, 30, 0, 0, 0, 0, time.UTC),
					Conf:       &config.WebsiteConfig{Separator: "\n"},
				}).Return(testError)

				return repo
			},
			web: &model.Website{
				URL:  serv.URL + "/success",
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			wantWeb: &model.Website{
				URL:        serv.URL + "/success",
				Title:      "title",
				UpdateTime: time.Date(2021, 7, 30, 0, 0, 0, 0, time.UTC),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			wantErr: testError,
		},
		{
			name: "send request returning error",
			serv: &VendorService{
				cli:  testClient,
				lock: semaphore.NewWeighted(1),
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 1,
					MaxRetry:       1,
				},
			},
			getCtx: func() context.Context {
				return context.Background()
			},
			getRepo: func(ctrl *gomock.Controller) repository.Repostory {
				return mockrepo.NewMockRepostory(ctrl)
			},
			web: &model.Website{
				URL:  serv.URL + "/fail",
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			wantWeb: &model.Website{
				URL:  serv.URL + "/fail",
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			wantErr: vendors.ErrInvalidStatusCode,
		},
		{
			name: "context was cancelled",
			serv: &VendorService{
				cli:  testClient,
				lock: semaphore.NewWeighted(1),
				cfg: &config.VendorServiceConfig{
					MaxConcurrency: 1,
					MaxRetry:       1,
				},
			},
			getCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			getRepo: func(ctrl *gomock.Controller) repository.Repostory {
				return mockrepo.NewMockRepostory(ctrl)
			},
			web: &model.Website{
				URL:  serv.URL + "/fail",
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			wantWeb: &model.Website{
				URL:  serv.URL + "/fail",
				Conf: &config.WebsiteConfig{Separator: "\n"},
			},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := tt.getCtx()
			tt.serv.repo = tt.getRepo(ctrl)
			err := tt.serv.Update(ctx, tt.web)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantWeb, tt.web)
		})
	}

}
