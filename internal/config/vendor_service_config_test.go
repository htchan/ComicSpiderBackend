package config

import (
	"os"
	"testing"

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
			want: &BaozimhConfig{},
		},
		{
			name: "empty cookie string",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "")
			},
			want: &BaozimhConfig{},
		},
		{
			name: "single cookie",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key=value")
			},
			want: &BaozimhConfig{Cookie: map[string]string{"key": "value"}},
		},
		{
			name: "multiple cookies",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key1=value1;key2=value2")
			},
			want: &BaozimhConfig{Cookie: map[string]string{"key1": "value1", "key2": "value2"}},
		},
		{
			name: "cookies with spaces",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", " key1 = value1 ; key2 = value2 ")
			},
			want: &BaozimhConfig{Cookie: map[string]string{"key1": "value1", "key2": "value2"}},
		},
		{
			name: "trailing separator",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key1=value1;")
			},
			want: &BaozimhConfig{Cookie: map[string]string{"key1": "value1"}},
		},
		{
			name: "empty segments are skipped",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", ";;key1=value1;;")
			},
			want: &BaozimhConfig{Cookie: map[string]string{"key1": "value1"}},
		},
		{
			name: "malformed segment without separator is skipped",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key1=value1;malformed;key2=value2")
			},
			want: &BaozimhConfig{Cookie: map[string]string{"key1": "value1", "key2": "value2"}},
		},
		{
			name: "value containing separator",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key=value=with=equals")
			},
			want: &BaozimhConfig{Cookie: map[string]string{"key": "value=with=equals"}},
		},
		{
			name: "empty value",
			setupEnv: func() {
				os.Setenv("BAOZIMH_COOKIES", "key=")
			},
			want: &BaozimhConfig{Cookie: map[string]string{"key": ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer os.Unsetenv("BAOZIMH_COOKIES")

			assert.Equal(t, tt.want, LoadBaozimhConfig())
		})
	}
}
