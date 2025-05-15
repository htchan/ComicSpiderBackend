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
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/htchan/WebHistory/internal/config"
	mockrepo "github.com/htchan/WebHistory/internal/mock/repository"
	mockvendor "github.com/htchan/WebHistory/internal/mock/vendor"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/nats/website_update"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_getAllWebsiteGroupsHandler(t *testing.T) {
	tests := []struct {
		name         string
		getRepo      func(*gomock.Controller) repository.Repostory
		userUUID     string
		expectStatus int
		expectRes    string
	}{
		{
			name: "get all user websites of specific user in group array format",
			getRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().FindUserWebsites(gomock.Any(), "abc").Return(
					model.UserWebsites{
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
					}, nil,
				)

				return rpo
			},
			userUUID:     "abc",
			expectStatus: 200,
			expectRes:    `{"website_groups":[[{"uuid":"1","user_uuid":"abc","url":"","title":"title 1","group_name":"group 1","update_time":"2000-01-01T01:00:00Z","access_time":"2000-01-01T00:00:00Z"},{"uuid":"2","user_uuid":"abc","url":"","title":"title 2","group_name":"group 1","update_time":"2000-01-02T01:00:00Z","access_time":"2000-01-02T00:00:00Z"}],[{"uuid":"3","user_uuid":"abc","url":"","title":"title 3","group_name":"group 3","update_time":"2000-01-03T01:00:00Z","access_time":"2000-01-03T00:00:00Z"}]]}`,
		},
		{
			name: "return error if findUserWebsites return error",
			getRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().FindUserWebsites(gomock.Any(), "unknown").Return(
					nil, errors.New("some error"),
				)

				return rpo
			},
			userUUID:     "unknown",
			expectStatus: 400,
			expectRes:    `{"error":"record not found"}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest("GET", "/websites/groups/", nil)
			assert.NoError(t, err, "create request")

			ctx := req.Context()
			ctx = context.WithValue(ctx, ContextKeyUserUUID, test.userUUID)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			getAllWebsiteGroupsHandler(test.getRepo(ctrl)).ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectRes, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}

func Test_getWebsiteGroupHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		r            func(*gomock.Controller) repository.Repostory
		userUUID     string
		group        string
		expectStatus int
		expectRes    string
	}{
		{
			name: "get user websites of existing user and group",
			r: func(c *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(c)
				rpo.EXPECT().FindUserWebsitesByGroup(gomock.Any(), "abc", "group 1").Return(
					model.WebsiteGroup{
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
					}, nil,
				)

				return rpo
			},
			userUUID:     "abc",
			group:        "group 1",
			expectStatus: 200,
			expectRes:    `{"website_group":[{"uuid":"1","user_uuid":"abc","url":"","title":"title 1","group_name":"group 1","update_time":"2000-01-01T01:00:00Z","access_time":"2000-01-01T00:00:00Z"},{"uuid":"2","user_uuid":"abc","url":"","title":"title 2","group_name":"group 1","update_time":"2000-01-02T01:00:00Z","access_time":"2000-01-02T00:00:00Z"}]}`,
		},
		{
			name: "return error if user not exist",
			r: func(c *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(c)
				rpo.EXPECT().FindUserWebsitesByGroup(gomock.Any(), "unknown", "group 1").Return(
					nil, errors.New("some error"),
				)

				return rpo
			},
			userUUID:     "unknown",
			group:        "group 1",
			expectStatus: 400,
			expectRes:    `{"error":"record not found"}`,
		},
		{
			name: "return error if group not exist",
			r: func(c *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(c)
				rpo.EXPECT().FindUserWebsitesByGroup(gomock.Any(), "abc", "group not exist").Return(
					nil, errors.New("some error"),
				)

				return rpo
			},
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

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest("GET", "/websites/groups/{groupName}", nil)
			assert.NoError(t, err, "create request")

			ctx := req.Context()
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("groupName", test.group)
			ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
			ctx = context.WithValue(ctx, ContextKeyUserUUID, test.userUUID)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			getWebsiteGroupHandler(test.r(ctrl)).ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectRes, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}

func Test_createWebsiteHandler(t *testing.T) {
	nc, err := nats.Connect(connString)
	assert.NoError(t, err)
	t.Cleanup(func() {
		nc.Close()
	})

	uuid.SetClockSequence(1)
	uuid.SetRand(io.NopCloser(bytes.NewReader([]byte(
		"000000000000000000000000000000000000000000000000000000000000000000000000000000",
	))))
	tests := []struct {
		name            string
		conf            *config.WebsiteConfig
		mockRepo        func(*gomock.Controller) repository.Repostory
		mockTasks       func(*gomock.Controller) websiteupdate.WebsiteUpdateTasks
		userUUID        string
		url             string
		expectStatus    int
		expectRes       string
		expectSubscribe func(*testing.T, *nats.Conn)
	}{
		{
			name: "happy flow/within 24 hrs",
			mockRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().CreateWebsite(gomock.Any(),
					&model.Website{
						UUID:       "30303030-3030-4030-b030-303030303030",
						URL:        "https://example.com/",
						UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
						Conf:       &config.WebsiteConfig{},
					},
				).Return(nil)

				rpo.EXPECT().CreateUserWebsite(gomock.Any(),
					&model.UserWebsite{
						WebsiteUUID: "30303030-3030-4030-b030-303030303030",
						UserUUID:    "abc",
						AccessTime:  time.Now().UTC().Truncate(5 * time.Second),
						Website: model.Website{
							UUID:       "30303030-3030-4030-b030-303030303030",
							URL:        "https://example.com/",
							UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
							Conf:       &config.WebsiteConfig{},
						},
					},
				).Return(nil)

				return rpo
			},
			mockTasks: func(ctrl *gomock.Controller) websiteupdate.WebsiteUpdateTasks {
				serv := mockvendor.NewMockVendorService(ctrl)

				return websiteupdate.WebsiteUpdateTasks{
					websiteupdate.NewTask(nc, serv, nil, nil),
				}
			},
			conf:            &config.WebsiteConfig{},
			userUUID:        "abc",
			url:             "https://example.com/",
			expectStatus:    200,
			expectRes:       `{"message":"website \u003c\u003e inserted"}`,
			expectSubscribe: func(t *testing.T, c *nats.Conn) {},
		},
		{
			name: "happy flow/more than 24 hrs",
			mockRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().CreateWebsite(gomock.Any(),
					&model.Website{
						UUID:       "30303030-3030-4030-b030-303030303030",
						URL:        "https://example.com/",
						UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
						Conf:       &config.WebsiteConfig{},
					},
				).
					Do(func(_ context.Context, web *model.Website) {
						web.UpdateTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(5 * time.Second)
					}).Return(nil)

				rpo.EXPECT().CreateUserWebsite(gomock.Any(),
					&model.UserWebsite{
						WebsiteUUID: "30303030-3030-4030-b030-303030303030",
						UserUUID:    "abc",
						AccessTime:  time.Now().UTC().Truncate(5 * time.Second),
						Website: model.Website{
							UUID:       "30303030-3030-4030-b030-303030303030",
							URL:        "https://example.com/",
							UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(5 * time.Second),
							Conf:       &config.WebsiteConfig{},
						},
					},
				).Return(nil)

				return rpo
			},
			mockTasks: func(ctrl *gomock.Controller) websiteupdate.WebsiteUpdateTasks {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Support(&model.Website{
					UUID:       "30303030-3030-4030-b030-303030303030",
					URL:        "https://example.com/",
					UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(5 * time.Second),
					Conf:       &config.WebsiteConfig{},
				}).Return(true)
				serv.EXPECT().Name().Return("create_web.success.more_than_24_hrs").AnyTimes()

				return websiteupdate.WebsiteUpdateTasks{
					websiteupdate.NewTask(nc, serv, nil, nil),
				}
			},
			conf:         &config.WebsiteConfig{},
			userUUID:     "abc",
			url:          "https://example.com/",
			expectStatus: 200,
			expectRes:    `{"message":"website \u003c\u003e inserted"}`,
			expectSubscribe: func(t *testing.T, c *nats.Conn) {
				var gotMsg *nats.Msg
				sub, err := nc.Subscribe("web_history.websites.update.create_web_success_more_than_24_hrs", func(msg *nats.Msg) {
					gotMsg = msg
					assert.Equal(t, `{"website":{"uuid":"30303030-3030-4030-b030-303030303030","url":"https://example.com/","title":"","raw_content":"","update_time":"2020-01-01T00:00:00Z"},"trace_id":"00000000000000000000000000000000","span_id":"0000000000000000","trace_flags":0}`, string(msg.Data))
				})
				assert.NoError(t, err)
				time.Sleep(100 * time.Millisecond)
				sub.Unsubscribe()

				assert.NotNil(t, gotMsg)
			},
		},
		{
			name: "error/not supported web",
			mockRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().CreateWebsite(gomock.Any(),
					&model.Website{
						UUID:       "30303030-3030-4030-b030-303030303030",
						URL:        "https://example.com/",
						UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
						Conf:       &config.WebsiteConfig{},
					},
				).
					Do(func(_ context.Context, web *model.Website) {
						web.UpdateTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(5 * time.Second)
					}).Return(nil)

				rpo.EXPECT().CreateUserWebsite(gomock.Any(),
					&model.UserWebsite{
						WebsiteUUID: "30303030-3030-4030-b030-303030303030",
						UserUUID:    "abc",
						AccessTime:  time.Now().UTC().Truncate(5 * time.Second),
						Website: model.Website{
							UUID:       "30303030-3030-4030-b030-303030303030",
							URL:        "https://example.com/",
							UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(5 * time.Second),
							Conf:       &config.WebsiteConfig{},
						},
					},
				).Return(nil)

				return rpo
			},
			mockTasks: func(ctrl *gomock.Controller) websiteupdate.WebsiteUpdateTasks {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Support(&model.Website{
					UUID:       "30303030-3030-4030-b030-303030303030",
					URL:        "https://example.com/",
					UpdateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(5 * time.Second),
					Conf:       &config.WebsiteConfig{},
				}).Return(false)
				serv.EXPECT().Name().Return("create_web.error.not_supported_web").AnyTimes()

				return websiteupdate.WebsiteUpdateTasks{
					websiteupdate.NewTask(nc, serv, nil, nil),
				}
			},
			conf:         &config.WebsiteConfig{},
			userUUID:     "abc",
			url:          "https://example.com/",
			expectStatus: 400,
			expectRes:    `{"error":"website is not supported"}`,
			expectSubscribe: func(t *testing.T, c *nats.Conn) {
				var gotMsg *nats.Msg
				sub, err := nc.Subscribe("web_history.websites.update.create_web_error_not_supported_web", func(msg *nats.Msg) {
					gotMsg = msg
					assert.True(t, false, "some msg was sent")
				})
				assert.NoError(t, err)
				time.Sleep(time.Millisecond)
				sub.Unsubscribe()

				assert.Nil(t, gotMsg)
			},
		},
		{
			name: "error/repo return error",
			mockRepo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().CreateWebsite(gomock.Any(),
					&model.Website{
						UUID:       "30303030-3030-4030-b030-303030303030",
						URL:        "https://example.com/",
						UpdateTime: time.Now().UTC().Truncate(5 * time.Second),
						Conf:       &config.WebsiteConfig{},
					},
				).Return(errors.New("some error"))

				return rpo
			},
			mockTasks: func(ctrl *gomock.Controller) websiteupdate.WebsiteUpdateTasks {
				serv := mockvendor.NewMockVendorService(ctrl)

				return websiteupdate.WebsiteUpdateTasks{
					websiteupdate.NewTask(nc, serv, nil, nil),
				}
			},
			conf:            &config.WebsiteConfig{},
			userUUID:        "unknown",
			url:             "https://example.com/",
			expectStatus:    400,
			expectRes:       `{"error":"some error"}`,
			expectSubscribe: func(t *testing.T, c *nats.Conn) {},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := http.NewRequest("POST", "/websites/", nil)
				assert.NoError(t, err, "create request")

				ctx := req.Context()
				ctx = context.WithValue(ctx, ContextKeyUserUUID, test.userUUID)
				ctx = context.WithValue(ctx, ContextKeyWebURL, test.url)
				req = req.WithContext(ctx)
				rr := httptest.NewRecorder()
				createWebsiteHandler(test.mockRepo(ctrl), test.conf, test.mockTasks(ctrl)).ServeHTTP(rr, req)

				assert.Equal(t, test.expectStatus, rr.Code)
				assert.Equal(t, test.expectRes, strings.Trim(rr.Body.String(), "\n"))
			}()

			test.expectSubscribe(t, nc)

			wg.Wait()
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
				rpo.EXPECT().UpdateUserWebsite(gomock.Any(),
					&model.UserWebsite{
						WebsiteUUID: "web_uuid",
						UserUUID:    "user_uuid",
						GroupName:   "name",
						AccessTime:  time.Now().UTC().Truncate(5 * time.Second),
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
			expectResp:   fmt.Sprintf(`{"website":{"uuid":"web_uuid","user_uuid":"user_uuid","url":"http://example.com/","title":"title","group_name":"name","update_time":"2000-01-01T00:00:00Z","access_time":"%s"}}`, time.Now().UTC().Truncate(5*time.Second).Format("2006-01-02T15:04:05Z07:00")),
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
				rpo.EXPECT().DeleteUserWebsite(gomock.Any(),
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
		getRepo      func(*gomock.Controller) repository.Repostory
		web          model.UserWebsite
		group        string
		expectStatus int
		expectResp   string
	}{
		{
			name: "return website of deleted content",
			getRepo: func(c *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(c)
				rpo.EXPECT().UpdateUserWebsite(gomock.Any(),
					&model.UserWebsite{
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
			group:        "group_name",
			expectStatus: 200,
			expectResp:   `{"website":{"uuid":"web_uuid","user_uuid":"user_uuid","url":"http://example.com/","title":"title","group_name":"group_name","update_time":"2000-01-01T00:00:00Z","access_time":"2000-01-01T00:00:00Z"}}`,
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
			ctx = context.WithValue(ctx, ContextKeyGroup, test.group)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			changeWebsiteGroupHandler(test.getRepo(ctrl)).ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatus, rr.Code)
			assert.Equal(t, test.expectResp, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}
