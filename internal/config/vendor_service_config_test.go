package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadBaozimhConfig(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func()
		want     *BaozimhConfig
	}{
		{
			name: "unset env var",
			setupEnv: func() {
				os.Unsetenv("BROWSER_WAIT_DURATION")
				os.Unsetenv("BROWSER_CDP_URL")
			},
			want: &BaozimhConfig{BrowserWaitDuration: 10 * time.Second},
		},
		{
			name: "browser wait duration from env",
			setupEnv: func() {
				os.Setenv("BROWSER_WAIT_DURATION", "5s")
			},
			want: &BaozimhConfig{BrowserWaitDuration: 5 * time.Second},
		},
		{
			name: "browser cdp url from env",
			setupEnv: func() {
				os.Setenv("BROWSER_CDP_URL", "http://127.0.0.1:9222")
			},
			want: &BaozimhConfig{
				BrowserCdpUrl:       "http://127.0.0.1:9222",
				BrowserWaitDuration: 10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer os.Unsetenv("BROWSER_WAIT_DURATION")
			defer os.Unsetenv("BROWSER_CDP_URL")

			assert.Equal(t, tt.want, LoadBaozimhConfig())
		})
	}
}
