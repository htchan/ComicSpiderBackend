package vendorhelper

import (
	"net/http"
	"testing"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/htchan/WebHistory/internal/vendors/baozimh"
	"github.com/htchan/WebHistory/internal/vendors/kuaikanmanhua"
	"github.com/htchan/WebHistory/internal/vendors/manhuagui"
	"github.com/htchan/WebHistory/internal/vendors/manhuaren"
	"github.com/htchan/WebHistory/internal/vendors/qiman6"
	"github.com/htchan/WebHistory/internal/vendors/u17"
	"github.com/htchan/WebHistory/internal/vendors/webtoons"
	"github.com/stretchr/testify/assert"
)

func TestNewServiceSet(t *testing.T) {
	t.Parallel()

	type params struct {
		cli  *http.Client
		repo repository.Repostory
		cfg  map[string]config.VendorServiceConfig
	}

	tests := []struct {
		name    string
		params  params
		want    []vendors.VendorService
		wantErr error
	}{
		{
			name: "happy flow",
			params: params{
				cli:  nil,
				repo: nil,
				cfg: map[string]config.VendorServiceConfig{
					baozimh.Host: {
						MaxConcurrency: 1,
						FetchInterval:  1 * time.Second,
					},
					kuaikanmanhua.Host: {
						MaxConcurrency: 2,
						FetchInterval:  2 * time.Second,
					},
					manhuagui.Host: {
						MaxConcurrency: 3,
						FetchInterval:  3 * time.Second,
					},
					manhuaren.Host: {
						MaxConcurrency: 4,
						FetchInterval:  4 * time.Second,
					},
					qiman6.Host: {
						MaxConcurrency: 5,
						FetchInterval:  5 * time.Second,
					},
					u17.Host: {
						MaxConcurrency: 6,
						FetchInterval:  6 * time.Second,
					},
					webtoons.Host: {
						MaxConcurrency: 7,
						FetchInterval:  7 * time.Second,
					},
				},
			},
			want: []vendors.VendorService{
				baozimh.NewVendorService(nil, nil, &config.VendorServiceConfig{
					MaxConcurrency: 1,
					FetchInterval:  1 * time.Second,
				}),
				kuaikanmanhua.NewVendorService(nil, nil, &config.VendorServiceConfig{
					MaxConcurrency: 2,
					FetchInterval:  2 * time.Second,
				}),
				manhuagui.NewVendorService(nil, nil, &config.VendorServiceConfig{
					MaxConcurrency: 3,
					FetchInterval:  3 * time.Second,
				}),
				manhuaren.NewVendorService(nil, nil, &config.VendorServiceConfig{
					MaxConcurrency: 4,
					FetchInterval:  4 * time.Second,
				}),
				qiman6.NewVendorService(nil, nil, &config.VendorServiceConfig{
					MaxConcurrency: 5,
					FetchInterval:  5 * time.Second,
				}),
				u17.NewVendorService(nil, nil, &config.VendorServiceConfig{
					MaxConcurrency: 6,
					FetchInterval:  6 * time.Second,
				}),
				webtoons.NewVendorService(nil, nil, &config.VendorServiceConfig{
					MaxConcurrency: 7,
					FetchInterval:  7 * time.Second,
				}),
			},
		},
		{
			name: "unknown host",
			params: params{
				cli:  nil,
				repo: nil,
				cfg: map[string]config.VendorServiceConfig{
					"invalid-host": {
						MaxConcurrency: 1,
						FetchInterval:  1 * time.Second,
					},
				},
			},
			want:    []vendors.VendorService{},
			wantErr: vendors.ErrUnknownHost,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			get, err := NewServiceSet(tt.params.cli, tt.params.repo, tt.params.cfg)

			assert.Equal(t, len(tt.want), len(get))
			for _, item := range get {
				assert.Contains(t, get, item)
			}

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
