package kuaikanmanhua

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/htchan/WebHistory/internal/config"
	mockrepo "github.com/htchan/WebHistory/internal/mock/repository"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.want, get)
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
				cli:  http.DefaultClient,
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
				cli:  http.DefaultClient,
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
				cli:  http.DefaultClient,
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
			web:  &model.Website{Conf: &config.WebsiteConfig{Separator: "\n"}},
			body: `<head><title>title</title></head>`,
			want: true,
			wantWeb: &model.Website{
				Title:      "title",
				Conf:       &config.WebsiteConfig{Separator: "\n"},
				UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
			},
		},
		{
			name: "title not updateif it is not empty",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{
				Title: "original title",
				Conf:  &config.WebsiteConfig{Separator: "\n"},
			},
			body: `<head><title>new title</title></head>`,
			want: false,
			wantWeb: &model.Website{
				Title: "original title",
				Conf:  &config.WebsiteConfig{Separator: "\n"},
			},
		},
		{
			name: "content update from empty to some value",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{Conf: &config.WebsiteConfig{Separator: "\n"}},
			body: `<html><body><div class="topic-episode">
				<div class="text-warp">
					<div class="detail"> content 1</div>
					<div class="detail"> content 2</div>
					<div class="detail"> content 3</div>
					<div class="detail"> content 4</div>
					<div class="detail"> content 5</div>
					<div class="detail"> content 6</div>
					<div class="detail"> content 7</div>
				</div>
			</div></body></html>`,
			want: true,
			wantWeb: &model.Website{
				RawContent: "content 1\ncontent 2\ncontent 3\ncontent 4\ncontent 5",
				Conf:       &config.WebsiteConfig{Separator: "\n"},
				UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
			},
		},
		{
			name: "content update from one value to another",
			serv: &VendorService{},
			getCtx: func() context.Context {
				return context.Background()
			},
			web: &model.Website{
				RawContent: "content 2\ncontent 3\ncontent 4\ncontent 5",
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			body: `<html><body><div class="topic-episode">
				<div class="text-warp">
					<div class="detail"> content 1</div>
					<div class="detail"> content 2</div>
					<div class="detail"> content 3</div>
					<div class="detail"> content 4</div>
					<div class="detail"> content 5</div>
					<div class="detail"> content 6</div>
					<div class="detail"> content 7</div>
				</div>
			</div></body></html>`,
			want: true,
			wantWeb: &model.Website{
				RawContent: "content 1\ncontent 2\ncontent 3\ncontent 4\ncontent 5",
				Conf:       &config.WebsiteConfig{Separator: "\n"},
				UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
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
			name: "support host www.kuaikanmanhua.com",
			serv: &VendorService{},
			web:  &model.Website{URL: "https://www.kuaikanmanhua.com/testing"},
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

	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusBadRequest)
		} else if r.URL.Path == "/success" {
			w.Write([]byte(`<html>
			<head><title>title</title></head>
			<body><div class="topic-episode">
				<div class="text-warp">
					<div class="detail"> content 1</div>
					<div class="detail"> content 2</div>
					<div class="detail"> content 3</div>
					<div class="detail"> content 4</div>
					<div class="detail"> content 5</div>
					<div class="detail"> content 6</div>
					<div class="detail"> content 7</div>
				</div>
			</div></body>
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
				cli:  http.DefaultClient,
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
					RawContent: "content 1\ncontent 2\ncontent 3\ncontent 4\ncontent 5",
					UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
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
				RawContent: "content 1\ncontent 2\ncontent 3\ncontent 4\ncontent 5",
				UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			wantErr: nil,
		},
		{
			name: "fetch info but not update web",
			serv: &VendorService{
				cli:  http.DefaultClient,
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
				RawContent: "content 1\ncontent 2\ncontent 3\ncontent 4\ncontent 5",
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			wantWeb: &model.Website{
				URL:        serv.URL + "/success",
				Title:      "title",
				RawContent: "content 1\ncontent 2\ncontent 3\ncontent 4\ncontent 5",
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			wantErr: nil,
		},
		{
			name: "repo returning error",
			serv: &VendorService{
				cli:  http.DefaultClient,
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
					RawContent: "content 1\ncontent 2\ncontent 3\ncontent 4\ncontent 5",
					UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
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
				RawContent: "content 1\ncontent 2\ncontent 3\ncontent 4\ncontent 5",
				UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			wantErr: testError,
		},
		{
			name: "send request returning error",
			serv: &VendorService{
				cli:  http.DefaultClient,
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
				cli:  http.DefaultClient,
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
