package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
)

func Test_pruneResponse(t *testing.T) {
	conf := &config.WebsiteConfig{Separator: ","}

	tests := []struct {
		name   string
		resp   http.Response
		expect string
	}{
		{
			name: "replace content script, style, path node to SEP",
			resp: http.Response{Body: io.NopCloser(bytes.NewReader([]byte(
				"a<scriptijk>ijk</script>b<styleijk>ijk</style>c<pathijk>ijk</path>d",
			)))},
			expect: "a,b,c,d",
		},
		{
			name: "replace <.*> to model sep",
			resp: http.Response{Body: io.NopCloser(bytes.NewReader([]byte(
				"a<a>b</a>c<pre>d</pre>e",
			)))},
			expect: "a,b,c,d,e",
		},
		{
			name: "keep <title> in content",
			resp: http.Response{Body: io.NopCloser(bytes.NewReader([]byte(
				"a<title>title</title>b",
			)))},
			expect: "a<title>title</title>b",
		},
		{
			name: "remove sep in beginning",
			resp: http.Response{Body: io.NopCloser(bytes.NewReader([]byte(
				"<beginning>a<a>b</a>c<pre>d</pre>e</beginning>",
			)))},
			expect: "a,b,c,d,e",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := pruneResponse(&test.resp, conf)
			if result != test.expect {
				t.Errorf("got different result as expect")
				t.Error(result)
				t.Error(test.expect)
			}
		})
	}
}

func Test_getWebsiteSetting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		r         repository.Repostory
		web       *model.Website
		expect    *model.WebsiteSetting
		expectErr bool
	}{
		{
			name: "works",
			r: repository.NewInMemRepo(
				nil, nil,
				[]model.WebsiteSetting{{Domain: "hello"}}, nil,
			),
			web:       &model.Website{URL: "https://hello/data"},
			expect:    &model.WebsiteSetting{Domain: "hello"},
			expectErr: false,
		},
		{
			name: "works with fallback default domain",
			r: repository.NewInMemRepo(
				nil, nil,
				[]model.WebsiteSetting{{Domain: "default"}}, nil,
			),
			web:       &model.Website{URL: "https://hello/data"},
			expect:    &model.WebsiteSetting{Domain: "default"},
			expectErr: false,
		},
		{
			name: "return error if no setting found",
			r: repository.NewInMemRepo(
				nil, nil, nil, nil,
			),
			web:       &model.Website{URL: "https://hello/data"},
			expect:    nil,
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			setting, err := getWebsiteSetting(test.r, test.web)

			if (err != nil) != test.expectErr {
				t.Errorf("got error: %v; expect error: %v", err, test.expectErr)
				return
			}

			if !cmp.Equal(setting, test.expect) {
				t.Errorf("setting diff: %v", cmp.Diff(setting, test.expect))
			}
		})
	}
}

func Test_parseAPI(t *testing.T) {
	t.Parallel()
	setting := model.WebsiteSetting{Domain: "hello", TitleGoquerySelector: "head>title", DatesGoquerySelector: "dates>date"}

	tests := []struct {
		name          string
		r             repository.Repostory
		web           *model.Website
		resp          string
		expectTitle   string
		expectContent []string
	}{
		{
			name: "works with selector",
			r: repository.NewInMemRepo(nil, nil, []model.WebsiteSetting{
				setting,
			}, nil),
			web: &model.Website{URL: "http://hello/data"},
			resp: `<html><head>
			<title>title-1</title>
			<dates><date>date-1</date><date>date-2</date><date>date-3</date><date>date-4</date></dates>
			</head></html>`,
			expectTitle:   "title-1",
			expectContent: []string{"date-1", "date-2", "date-3", "date-4"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			title, content := parseAPI(test.r, test.web, test.resp)

			if title != test.expectTitle {
				t.Errorf("got title: %v; want title: %v", title, test.expectTitle)
			}
			if !cmp.Equal(content, test.expectContent) {
				t.Errorf("content diff: %v", cmp.Diff(content, test.expectContent))
			}
		})
	}
}

func Test_fetchWebsite(t *testing.T) {
	t.Parallel()
	workingClient := MockClient{
		get: func(url string) (*http.Response, error) {
			return &http.Response{Body: io.NopCloser(bytes.NewReader([]byte(
				"response",
			)))}, nil
		},
	}
	errorClient := MockClient{
		get: func(url string) (*http.Response, error) {
			return nil, errors.New("error")
		},
	}
	conf := &config.WebsiteConfig{Separator: "\n", MaxDateLength: 2}
	tests := []struct {
		name          string
		client        HTTPClient
		web           *model.Website
		maxRetry      int
		retryInterval time.Duration
		expect        string
		expectErr     bool
	}{
		{
			name:          "works",
			client:        workingClient,
			web:           &model.Website{URL: "http://hello.com", Conf: conf},
			maxRetry:      10,
			retryInterval: 100 * time.Millisecond,
			expect:        "response",
			expectErr:     false,
		},
		{
			name:          "return error when fail",
			client:        errorClient,
			web:           &model.Website{URL: "http://hello.com", Conf: conf},
			maxRetry:      10,
			retryInterval: 100 * time.Millisecond,
			expect:        "",
			expectErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client = test.client

			resp, err := fetchWebsite(context.Background(), test.web, test.maxRetry, test.retryInterval)

			if (err != nil) != test.expectErr {
				t.Errorf("got error: %v; expect error: %v", err, test.expectErr)
				return
			}

			if resp != test.expect {
				t.Errorf("got resp: %v; want resp: %v", resp, test.expect)
			}
		})
	}
}

func Test_checkTimeUpdated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		web    model.Website
		time   string
		expect bool
	}{
		{
			name:   "updated time",
			web:    model.Website{UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
			time:   "Sat, 1 Jan 2000 00:00:01 GMT",
			expect: true,
		},
		{
			name:   "not updated time",
			web:    model.Website{UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
			time:   "Sat, 1 Jan 2000 00:00:00 GMT",
			expect: false,
		},
		{
			name:   "time in wrong format",
			web:    model.Website{UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
			time:   "Sat, 1 Abc 2000 00:00:00 GMT",
			expect: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := checkTimeUpdated(context.Background(), &test.web, test.time)
			if result != test.expect {
				t.Errorf("got different result as expect")
				t.Error(result)
				t.Error(test.expect)
			}
		})
	}
}

func Test_checkContentUpdated(t *testing.T) {
	conf := &config.WebsiteConfig{Separator: ","}

	tests := []struct {
		name   string
		web    model.Website
		dates  []string
		expect bool
	}{
		{
			name:   "different dates",
			web:    model.Website{RawContent: "1,2,3,4,5", Conf: conf},
			dates:  []string{"0", "1", "2", "3", "4", "5"},
			expect: true,
		},
		{
			name:   "exact same dates",
			web:    model.Website{RawContent: "1,2,3,4,5", Conf: conf},
			dates:  []string{"1", "2", "3", "4", "5"},
			expect: false,
		},
		{
			name:   "exact same date with length model.DateLength at the beginning",
			web:    model.Website{RawContent: "1,2,3,4,5", Conf: conf},
			dates:  []string{"1", "2", "3", "4", "999"},
			expect: true,
		},
		{
			name:   "shorter dates",
			web:    model.Website{RawContent: "1,2,3,4,5", Conf: conf},
			dates:  []string{"1"},
			expect: true,
		},
		{
			name:   "empty dates",
			web:    model.Website{RawContent: "1,2,3,4,5", Conf: conf},
			dates:  nil,
			expect: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			// t.Parallel()
			fmt.Println(test.name)
			fmt.Println(test.dates)
			result := checkContentUpdated(context.Background(), &test.web, test.dates)
			if result != test.expect {
				t.Errorf("got: %v; want: %v", result, test.expect)
			}
		})
	}
}

func Test_checkTitleUpdated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		web    model.Website
		title  string
		expect bool
	}{
		{
			name:   "different title for new Website",
			web:    model.Website{},
			title:  "new title",
			expect: true,
		},
		{
			name:   "different title for existing Website",
			web:    model.Website{Title: "title"},
			title:  "new title",
			expect: false,
		},
		{
			name:   "exact same title",
			web:    model.Website{Title: "title"},
			title:  "title",
			expect: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := checkTitleUpdated(context.Background(), &test.web, test.title)
			if result != test.expect {
				t.Errorf("got different result as expect")
				t.Error(result)
				t.Error(test.expect)
			}
		})
	}
}

type MockClient struct {
	get func(string) (*http.Response, error)
}

func (m MockClient) Get(url string) (*http.Response, error) {
	return m.get(url)
}

func Test_Update(t *testing.T) {
	conf := &config.WebsiteConfig{Separator: ",", MaxDateLength: 2}

	refArray := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		refArray = append(refArray, strconv.Itoa(i))
	}

	mockRespWithoutDates := `<html><head>
	<title>new title</title>
	</head></html>`
	mockRespWithDates := `<html><head>
	<title>new title</title>
	<dates><date>date-1</date><date>date-2</date>
	<date>date-3</date><date>date-4</date></dates>
	</head></html>`
	mockSetting := model.WebsiteSetting{Domain: "domain", TitleGoquerySelector: "head>title", DatesGoquerySelector: "dates>date"}

	tests := []struct {
		name       string
		r          repository.Repostory
		web        model.Website
		mockClient MockClient
		expectWeb  model.Website
		expectErr  bool
	}{
		{
			name: "not updated title of web already have title",
			r: repository.NewInMemRepo(
				[]model.Website{{UUID: "uuid", URL: "http://domain", Title: "original title"}},
				nil,
				[]model.WebsiteSetting{mockSetting},
				nil,
			),
			web: model.Website{UUID: "uuid", URL: "http://domain", Title: "original title", Conf: conf},
			mockClient: MockClient{get: func(s string) (*http.Response, error) {
				return &http.Response{Body: io.NopCloser(strings.NewReader(mockRespWithoutDates))}, nil
			}},
			expectWeb: model.Website{UUID: "uuid", URL: "http://domain", Title: "original title"},
		},
		{
			name: "updated title of web not having title",
			r: repository.NewInMemRepo(
				[]model.Website{{UUID: "uuid", URL: "http://domain"}},
				nil,
				[]model.WebsiteSetting{mockSetting},
				nil,
			),
			web: model.Website{UUID: "uuid", URL: "http://domain", Conf: conf},
			mockClient: MockClient{get: func(s string) (*http.Response, error) {
				return &http.Response{Body: io.NopCloser(strings.NewReader(mockRespWithoutDates))}, nil
			}},
			expectWeb: model.Website{
				UUID: "uuid", URL: "http://domain", Title: "new title",
				UpdateTime: time.Now().UTC().Truncate(time.Second),
			},
		},
		{
			name: "updated content",
			r: repository.NewInMemRepo(
				[]model.Website{{UUID: "uuid", URL: "http://domain", RawContent: "date-1,date-2"}},
				nil,
				[]model.WebsiteSetting{mockSetting},
				nil,
			),
			web: model.Website{UUID: "uuid", URL: "http://domain", RawContent: "date-1,date-2", Conf: conf},
			mockClient: MockClient{get: func(s string) (*http.Response, error) {
				return &http.Response{Body: io.NopCloser(strings.NewReader(mockRespWithDates))}, nil
			}},
			expectWeb: model.Website{
				UUID: "uuid", URL: "http://domain", Title: "new title",
				RawContent: "date-1,date-2,date-3,date-4",
				UpdateTime: time.Now().UTC().Truncate(time.Second),
			},
		},
		{
			name: "not updated content",
			r: repository.NewInMemRepo(
				[]model.Website{{UUID: "uuid", URL: "http://domain", Title: "new title", RawContent: "date-1,date-2,date-3,date-4"}},
				nil,
				[]model.WebsiteSetting{mockSetting},
				nil,
			),
			web: model.Website{UUID: "uuid", URL: "http://domain", RawContent: "11-1-1,22-2-2", Conf: conf},
			mockClient: MockClient{get: func(s string) (*http.Response, error) {
				return &http.Response{Body: io.NopCloser(strings.NewReader(mockRespWithDates))}, nil
			}},
			expectWeb: model.Website{
				UUID: "uuid", URL: "http://domain", Title: "new title",
				RawContent: "date-1,date-2,date-3,date-4",
				UpdateTime: time.Now().UTC().Truncate(time.Second),
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			client = test.mockClient
			err := Update(context.Background(), test.r, &test.web)

			if (err != nil) != test.expectErr {
				t.Errorf("got error: %v; want error: %v", err, test.expectErr)
			}

			web, err := test.r.FindWebsite(test.expectWeb.UUID)
			if err != nil {
				t.Errorf("find website got error: %v", err)
			} else if !cmp.Equal(*web, test.expectWeb) {
				t.Error("got different repo result")
				t.Error(web)
				t.Error(test.expectWeb)
			}

			if !cmp.Equal(test.web, test.expectWeb) {
				t.Error("got different web")
				t.Error(test.web)
				t.Error(test.expectWeb)
			}
		})
	}
}
