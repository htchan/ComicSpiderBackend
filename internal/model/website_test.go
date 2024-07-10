package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
			if web.UUID == "" {
				t.Errorf("empty uuid")
			}
			if web.URL != test.url {
				t.Errorf("wrong url: %s; want: %s", web.URL, test.url)
			}
			if web.Title != test.expectedTitle {
				t.Errorf("wrong title: %s; want: %s", web.Title, test.expectedTitle)
			}
			if web.RawContent != test.expectedRawContent {
				t.Errorf("wrong RawContent: %s; want: %s", web.RawContent, test.expectedRawContent)
			}
			if web.UpdateTime.Second() != test.expectedUpdateTime.Second() {
				t.Errorf("wrong UpdateTime: %s; want: %s", web.UpdateTime, test.expectedUpdateTime)
			}
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
			if !cmp.Equal(result, test.expect) {
				t.Errorf("got unexpected map")
				t.Error(result)
				t.Error(test.expect)
			}
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
			if err != nil {
				t.Errorf("got error: %v", err)
				return
			}
			if !cmp.Equal(string(result), test.expect) {
				t.Errorf("got unexpected json")
				t.Error(string(result))
				t.Error(test.expect)
			}
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
			if !cmp.Equal(string(result), test.expect) {
				t.Errorf("got unexpected host")
				t.Error(string(result))
				t.Error(test.expect)
			}
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
			if !cmp.Equal(result, test.expect) {
				t.Errorf("got unexpected host")
				t.Error(result)
				t.Error(test.expect)
			}
		})
	}
}
