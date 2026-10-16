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
				os.Unsetenv("BAOZIMH_COOKIES")
			},
			want: &BaozimhConfig{BrowserWaitDuration: 10 * time.Second},
		},
		{
			name: "empty cookie string",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "")
			},
			want: &BaozimhConfig{BrowserWaitDuration: 10 * time.Second},
		},
		{
			name: "single cookie",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key=value")
			},
			want: &BaozimhConfig{
				Cookie:              map[string]string{"key": "value"},
				BrowserWaitDuration: 10 * time.Second,
			},
		},
		{
			name: "multiple cookies",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key1=value1;key2=value2")
			},
			want: &BaozimhConfig{
				Cookie:              map[string]string{"key1": "value1", "key2": "value2"},
				BrowserWaitDuration: 10 * time.Second,
			},
		},
		{
			name: "cookies with spaces",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", " key1 = value1 ; key2 = value2 ")
			},
			want: &BaozimhConfig{
				Cookie:              map[string]string{"key1": "value1", "key2": "value2"},
				BrowserWaitDuration: 10 * time.Second,
			},
		},
		{
			name: "trailing separator",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key1=value1;")
			},
			want: &BaozimhConfig{
				Cookie:              map[string]string{"key1": "value1"},
				BrowserWaitDuration: 10 * time.Second,
			},
		},
		{
			name: "empty segments are skipped",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", ";;key1=value1;;")
			},
			want: &BaozimhConfig{
				Cookie:              map[string]string{"key1": "value1"},
				BrowserWaitDuration: 10 * time.Second,
			},
		},
		{
			name: "malformed segment without separator is skipped",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key1=value1;malformed;key2=value2")
			},
			want: &BaozimhConfig{
				Cookie:              map[string]string{"key1": "value1", "key2": "value2"},
				BrowserWaitDuration: 10 * time.Second,
			},
		},
		{
			name: "value containing separator",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key=value=with=equals")
			},
			want: &BaozimhConfig{
				Cookie:              map[string]string{"key": "value=with=equals"},
				BrowserWaitDuration: 10 * time.Second,
			},
		},
		{
			name: "empty value",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key=")
			},
			want: &BaozimhConfig{
				Cookie:              map[string]string{"key": ""},
				BrowserWaitDuration: 10 * time.Second,
			},
		},
		{
			name: "browser wait duration from env",
			setupEnv: func() {
				os.Unsetenv("BAOZIMH_COOKIES")
				os.Setenv("BROWSER_WAIT_DURATION", "5s")
			},
			want: &BaozimhConfig{BrowserWaitDuration: 5 * time.Second},
		},
		{
			name: "browser cdp url from env",
			setupEnv: func() {
				os.Unsetenv("BAOZIMH_COOKIES")
				os.Setenv("BROWSER_CDP_URL", "ws://127.0.0.1:9222")
			},
			want: &BaozimhConfig{
				BrowserCdpUrl:       "ws://127.0.0.1:9222",
				BrowserWaitDuration: 10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer os.Unsetenv("BAOZIMH_COOKIES")
			defer os.Unsetenv("BROWSER_WAIT_DURATION")
			defer os.Unsetenv("BROWSER_CDP_URL")

			assert.Equal(t, tt.want, LoadBaozimhConfig())
		})
	}
}
