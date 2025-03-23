package website

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/htchan/WebHistory/internal/config"
	mockrepo "github.com/htchan/WebHistory/internal/mock/repository"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/stretchr/testify/assert"
)

func Test_getAllWebsiteGroupsHandler(t *testing.T) {
	tests := []struct {
		name         string
		r            repository.Repostory
		userUUID     string
		expectStatus int
		expectRes    string
	}{
		{
			name: "get all user websites of specific user in group array format",
			r: repository.NewInMemRepo(nil, []model.UserWebsite{
				{
					UserUUID:    "abc",
					WebsiteUUID: "1",
					GroupName:   "group 1",
					AccessTime:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
					Website: model.Website{
						UUID:       "1",
						Title:      "title 1",
						UpdateTime: time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC),
					},
				},
				{
					UserUUID:    "abc",
					WebsiteUUID: "2",
					GroupName:   "group 1",
					AccessTime:  time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
					Website: model.Website{
						UUID:       "2",
						Title:      "title 2",
						UpdateTime: time.Date(2000, 1, 2, 1, 0, 0, 0, time.UTC),
					},
				},
				{
					UserUUID:    "abc",
					WebsiteUUID: "3",
					GroupName:   "group 3",
					AccessTime:  time.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC),
					Website: model.Website{
						UUID:       "3",
						Title:      "title 3",
						UpdateTime: time.Date(2000, 1, 3, 1, 0, 0, 0, time.UTC),
					},
				},
			}, nil, nil),
			userUUID:     "abc",
			expectStatus: 200,
			expectRes:    `{"website_groups":[[{"uuid":"1","user_uuid":"abc","url":"","title":"title 1","group_name":"group 1","update_time":"2000-01-01T01:00:00Z","access_time":"2000-01-01T00:00:00Z"},{"uuid":"2","user_uuid":"abc","url":"","title":"title 2","group_name":"group 1","update_time":"2000-01-02T01:00:00Z","access_time":"2000-01-02T00:00:00Z"}],[{"uuid":"3","user_uuid":"abc","url":"","title":"title 3","group_name":"group 3","update_time":"2000-01-03T01:00:00Z","access_time":"2000-01-03T00:00:00Z"}]]}`,
		},
		{
			name:         "return error if findUserWebsites return error",
			r:            repository.NewInMemRepo(nil, nil, nil, errors.New("some error")),
			userUUID:     "unknown",
			expectStatus: 400,
			expectRes:    `{"error":"record not found"}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest("GET", "/websites/groups/", nil)
			assert.NoError(t, err, "create request")

			ctx := req.Context()
			ctx = context.WithValue(ctx, ContextKeyUserUUID, test.userUUID)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			getAllWebsiteGroupsHandler(test.r).ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectRes, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}

func Test_getWebsiteGroupHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		r            repository.Repostory
		userUUID     string
		group        string
		expectStatus int
		expectRes    string
	}{
		{
			name: "get user websites of existing user and group",
			r: repository.NewInMemRepo(nil, []model.UserWebsite{
				{
					UserUUID:    "abc",
					WebsiteUUID: "1",
					GroupName:   "group 1",
					AccessTime:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
					Website: model.Website{
						UUID:       "1",
						Title:      "title 1",
						UpdateTime: time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC),
					},
				},
				{
					UserUUID:    "abc",
					WebsiteUUID: "2",
					GroupName:   "group 1",
					AccessTime:  time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
					Website: model.Website{
						UUID:       "2",
						Title:      "title 2",
						UpdateTime: time.Date(2000, 1, 2, 1, 0, 0, 0, time.UTC),
					},
				},
			}, nil, nil),
			userUUID:     "abc",
			group:        "group 1",
			expectStatus: 200,
			expectRes:    `{"website_group":[{"uuid":"1","user_uuid":"abc","url":"","title":"title 1","group_name":"group 1","update_time":"2000-01-01T01:00:00Z","access_time":"2000-01-01T00:00:00Z"},{"uuid":"2","user_uuid":"abc","url":"","title":"title 2","group_name":"group 1","update_time":"2000-01-02T01:00:00Z","access_time":"2000-01-02T00:00:00Z"}]}`,
		},
		{
			name:         "return error if user not exist",
			r:            repository.NewInMemRepo(nil, nil, nil, errors.New("some error")),
			userUUID:     "unknown",
			group:        "group 1",
			expectStatus: 400,
			expectRes:    `{"error":"record not found"}`,
		},
		{
			name:         "return error if group not exist",
			r:            repository.NewInMemRepo(nil, nil, nil, errors.New("some error")),
			userUUID:     "abc",
			group:        "group not exist",
			expectStatus: 400,
			expectRes:    `{"error":"record not found"}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest("GET", "/websites/groups/{groupName}", nil)
			assert.NoError(t, err, "create request")

			ctx := req.Context()
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("groupName", test.group)
			ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
			ctx = context.WithValue(ctx, ContextKeyUserUUID, test.userUUID)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			getWebsiteGroupHandler(test.r).ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectRes, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}

func Test_createWebsiteHandler(t *testing.T) {
	uuid.SetClockSequence(1)
	uuid.SetRand(io.NopCloser(bytes.NewReader([]byte(
		"000000000000000000000000000000000000000000000000000000000000000000000000000000",
	))))
	tests := []struct {
		name         string
		conf         *config.WebsiteConfig
		mockRepo     func(*gomock.Controller) repository.Repostory
		userUUID     string
		url          string
		expectStatus int
		expectRes    string
	}{
		{
			name: "get user websites of existing user and group",
			mockRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().CreateWebsite(
					&model.Website{
						UUID:       "30303030-3030-4030-b030-303030303030",
						URL:        "https://example.com/",
						UpdateTime: time.Now().UTC().Truncate(time.Second),
						Conf:       &config.WebsiteConfig{},
					},
				).Return(nil)

				rpo.EXPECT().CreateUserWebsite(
					&model.UserWebsite{
						WebsiteUUID: "30303030-3030-4030-b030-303030303030",
						UserUUID:    "abc",
						AccessTime:  time.Now().UTC().Truncate(time.Second),
						Website: model.Website{
							UUID:       "30303030-3030-4030-b030-303030303030",
							URL:        "https://example.com/",
							UpdateTime: time.Now().UTC().Truncate(time.Second),
							Conf:       &config.WebsiteConfig{},
						},
					},
				).Return(nil)

				return rpo
			},
			conf:         &config.WebsiteConfig{},
			userUUID:     "abc",
			url:          "https://example.com/",
			expectStatus: 200,
			expectRes:    `{"message":"website \u003c\u003e inserted"}`,
		},
		{
			name: "return error if repo return error",
			mockRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().CreateWebsite(
					&model.Website{
						UUID:       "30303030-3030-4030-b030-303030303030",
						URL:        "https://example.com/",
						UpdateTime: time.Now().UTC().Truncate(time.Second),
						Conf:       &config.WebsiteConfig{},
					},
				).Return(errors.New("some error"))

				return rpo
			},
			conf:         &config.WebsiteConfig{},
			userUUID:     "unknown",
			url:          "https://example.com/",
			expectStatus: 400,
			expectRes:    `{"error":"some error"}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest("POST", "/websites/", nil)
			assert.NoError(t, err, "create request")

			ctx := req.Context()
			ctx = context.WithValue(ctx, ContextKeyUserUUID, test.userUUID)
			ctx = context.WithValue(ctx, ContextKeyWebURL, test.url)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			// TODO: mock the update Tasks to make sure it publish tasks for supported website
			createWebsiteHandler(test.mockRepo(ctrl), test.conf, nil).ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectRes, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}

func Test_getWebsiteHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		web          model.UserWebsite
		expectStatus int
		expectRes    string
	}{
		{
			name: "return website with correct format",
			web: model.UserWebsite{
				WebsiteUUID: "web_uuid",
				UserUUID:    "user_uuid",
				GroupName:   "name",
				AccessTime:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				Website: model.Website{
					UUID:       "web_uuid",
					Title:      "title",
					URL:        "http://example.com/",
					UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expectStatus: 200,
			expectRes:    `{"website":{"uuid":"web_uuid","user_uuid":"user_uuid","url":"http://example.com/","title":"title","group_name":"name","update_time":"2000-01-01T00:00:00Z","access_time":"2000-01-01T00:00:00Z"}}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest("GET", "/websites/{webUUID}", nil)
			assert.NoError(t, err, "create request")

			ctx := req.Context()
			ctx = context.WithValue(ctx, ContextKeyWebsite, test.web)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			getUserWebsiteHandler().ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectRes, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}

func Test_refreshWebsiteHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		mockRepo     func(*gomock.Controller) repository.Repostory
		web          model.UserWebsite
		expectStatus int
		expectResp   string
	}{
		{
			name: "return website with updated AccessTime",
			mockRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().UpdateUserWebsite(
					&model.UserWebsite{
						WebsiteUUID: "web_uuid",
						UserUUID:    "user_uuid",
						GroupName:   "name",
						AccessTime:  time.Now().UTC().Truncate(time.Second),
						Website: model.Website{
							UUID:       "web_uuid",
							Title:      "title",
							URL:        "http://example.com/",
							UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
						},
					},
				).Return(nil)

				return rpo
			},
			web: model.UserWebsite{
				WebsiteUUID: "web_uuid",
				UserUUID:    "user_uuid",
				GroupName:   "name",
				AccessTime:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				Website: model.Website{
					UUID:       "web_uuid",
					Title:      "title",
					URL:        "http://example.com/",
					UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expectStatus: 200,
			expectResp:   fmt.Sprintf(`{"website":{"uuid":"web_uuid","user_uuid":"user_uuid","url":"http://example.com/","title":"title","group_name":"name","update_time":"2000-01-01T00:00:00Z","access_time":"%s"}}`, time.Now().UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z07:00")),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest("GET", "/websites/{webUUID}/refresh", nil)
			assert.NoError(t, err, "create request")

			ctx := req.Context()
			ctx = context.WithValue(ctx, ContextKeyWebsite, test.web)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			refreshWebsiteHandler(test.mockRepo(ctrl)).ServeHTTP(rr, req)
			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectResp, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}

func Test_deleteWebsiteHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		mockRepo     func(*gomock.Controller) repository.Repostory
		web          model.UserWebsite
		expectStatus int
		expectResp   string
	}{
		{
			name: "return website of deleted content",
			mockRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().DeleteUserWebsite(
					&model.UserWebsite{
						WebsiteUUID: "web_uuid",
						UserUUID:    "user_uuid",
						GroupName:   "name",
						AccessTime:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
						Website: model.Website{
							UUID:       "web_uuid",
							Title:      "title",
							URL:        "http://example.com/",
							UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
						},
					},
				).Return(nil)

				return rpo
			},
			web: model.UserWebsite{
				WebsiteUUID: "web_uuid",
				UserUUID:    "user_uuid",
				GroupName:   "name",
				AccessTime:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				Website: model.Website{
					UUID:       "web_uuid",
					Title:      "title",
					URL:        "http://example.com/",
					UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expectStatus: 200,
			expectResp:   `{"message":"website \u003ctitle\u003e deleted"}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest("GET", "/websites/{webUUID}/refresh", nil)
			assert.NoError(t, err, "create request")

			ctx := req.Context()
			ctx = context.WithValue(ctx, ContextKeyWebsite, test.web)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			deleteWebsiteHandler(test.mockRepo(ctrl)).ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectResp, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}

func Test_changeWebsiteGroupHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		r            repository.Repostory
		web          model.UserWebsite
		group        string
		expectRepo   repository.Repostory
		expectStatus int
		expectResp   string
	}{
		{
			name: "return website of deleted content",
			r: repository.NewInMemRepo(
				nil,
				[]model.UserWebsite{
					{
						WebsiteUUID: "web_uuid",
						UserUUID:    "user_uuid",
						GroupName:   "name",
						AccessTime:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
						Website: model.Website{
							UUID:       "web_uuid",
							Title:      "title",
							URL:        "http://example.com/",
							UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
						},
					},
				},
				nil, nil,
			),
			web: model.UserWebsite{
				WebsiteUUID: "web_uuid",
				UserUUID:    "user_uuid",
				GroupName:   "name",
				AccessTime:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				Website: model.Website{
					UUID:       "web_uuid",
					Title:      "title",
					URL:        "http://example.com/",
					UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			group: "group_name",
			expectRepo: repository.NewInMemRepo(
				nil,
				[]model.UserWebsite{
					{
						WebsiteUUID: "web_uuid",
						UserUUID:    "user_uuid",
						GroupName:   "group_name",
						AccessTime:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
						Website: model.Website{
							UUID:       "web_uuid",
							Title:      "title",
							URL:        "http://example.com/",
							UpdateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
						},
					},
				},
				nil, nil,
			),
			expectStatus: 200,
			expectResp:   `{"website":{"uuid":"web_uuid","user_uuid":"user_uuid","url":"http://example.com/","title":"title","group_name":"group_name","update_time":"2000-01-01T00:00:00Z","access_time":"2000-01-01T00:00:00Z"}}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest("GET", "/websites/{webUUID}/refresh", nil)
			assert.NoError(t, err, "create request")

			ctx := req.Context()
			ctx = context.WithValue(ctx, ContextKeyWebsite, test.web)
			ctx = context.WithValue(ctx, ContextKeyGroup, test.group)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			changeWebsiteGroupHandler(test.r).ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectResp, strings.Trim(rr.Body.String(), "\n"))
			if !cmp.Equal(test.r, test.expectRepo) {
				t.Error("got different repo as expect")
				t.Error(test.r)
				t.Error(test.expectRepo)
			}
		})
	}
}
