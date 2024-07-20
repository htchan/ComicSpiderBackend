package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/stretchr/testify/assert"
)

func Test_NewWebsite(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		url                string
		conf               *config.WebsiteConfig
		expectedTitle      string
		expectedRawContent string
		expectedUpdateTime time.Time
	}{
		{
			name:               "happy flow",
			url:                "https://google.com",
			expectedTitle:      "",
			expectedRawContent: "",
			expectedUpdateTime: time.Now().UTC().Truncate(time.Second),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			web := NewWebsite(test.url, test.conf)
			assert.NotEmptyf(t, web.UUID, "web uuid")
			assert.Equalf(t, test.url, web.URL, "web url")
			assert.Equalf(t, test.expectedTitle, web.Title, "web title")
			assert.Equalf(t, test.expectedRawContent, web.RawContent, "web RawContent")
			assert.Equal(t, test.expectedUpdateTime, web.UpdateTime, "web UpdateTime")

		})
	}
}

func TestWebsite_Map(t *testing.T) {
	tests := []struct {
		name   string
		web    Website
		expect map[string]interface{}
	}{
		{
			name: "happy flow",
			web: Website{
				UUID:       "uuid",
				URL:        "http://example.com",
				Title:      "title",
				UpdateTime: time.Date(2020, 1, 2, 0, 0, 0, 0, time.Local),
			},
			expect: map[string]interface{}{
				"uuid":       "uuid",
				"url":        "http://example.com",
				"title":      "title",
				"updateTime": time.Date(2020, 1, 2, 0, 0, 0, 0, time.Local),
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result := test.web.Map()
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestWebsite_MarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		web    Website
		expect string
	}{
		{
			name: "happy flow",
			web: Website{
				UUID:       "uuid",
				URL:        "http://example.com",
				Title:      "title",
				UpdateTime: time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			expect: `{"uuid":"uuid","url":"http://example.com","title":"title","update_time":"2020-01-02T00:00:00 UTC"}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result, err := json.Marshal(test.web)
			assert.NoError(t, err)
			assert.Equal(t, test.expect, string(result))
		})
	}
}

func TestWebsite_FullHost(t *testing.T) {
	tests := []struct {
		name   string
		web    Website
		expect string
	}{
		{
			name:   "happy flow with www",
			web:    Website{URL: "http://www.example.com"},
			expect: "www.example.com",
		},
		{
			name:   "happy flow with m",
			web:    Website{URL: "http://m.example.com"},
			expect: "m.example.com",
		},
		{
			name:   "fail flow",
			web:    Website{URL: ""},
			expect: "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result := test.web.FullHost()
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestWebsite_Host(t *testing.T) {
	tests := []struct {
		name   string
		web    Website
		expect string
	}{
		{
			name:   "happy flow",
			web:    Website{URL: "http://example.com"},
			expect: "example.com",
		},
		{
			name:   "fail flow",
			web:    Website{URL: ""},
			expect: "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result := test.web.Host()
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestWebsite_Content(t *testing.T) {
	tests := []struct {
		name   string
		web    Website
		expect []string
	}{
		{
			name: "happy flow",
			web: Website{
				RawContent: strings.Join([]string{"1", "2", "3"}, "\n"),
				Conf:       &config.WebsiteConfig{Separator: "\n"},
			},
			expect: []string{"1", "2", "3"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result := test.web.Content()
			assert.Equal(t, test.expect, result)
		})
	}
}
