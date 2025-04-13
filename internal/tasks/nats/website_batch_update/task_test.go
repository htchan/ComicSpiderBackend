package websitebatchupdate

import (
	"errors"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mocknats "github.com/htchan/WebHistory/internal/mock/nats"
	mockrepo "github.com/htchan/WebHistory/internal/mock/repository"
	mockvendor "github.com/htchan/WebHistory/internal/mock/vendor"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/nats/website_update"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
)

func TestNewTask(t *testing.T) {
	tests := []struct {
		name   string
		nc     *nats.Conn
		tasks  websiteupdate.WebsiteUpdateTasks
		rpo    repository.Repostory
		expect WebsiteBatchUpdateTask
	}{
		{
			name:  "new task",
			nc:    nil,
			tasks: nil,
			rpo:   nil,
			expect: WebsiteBatchUpdateTask{
				nc:                 nil,
				websiteUpdateTasks: nil,
				rpo:                nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := NewTask(test.nc, test.tasks, test.rpo)
			assert.Equal(t, test.expect, task)
		})
	}
}

func TestWebsiteBatchUpdateTask_subject(t *testing.T) {
	tests := []struct {
		name   string
		task   WebsiteBatchUpdateTask
		expect string
	}{
		{
			name:   "happy flow",
			task:   WebsiteBatchUpdateTask{},
			expect: "web_history.websites.batch_update",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expect, test.task.subject())
		})
	}
}

func TestWebsiteBatchUpdateTask_Subscribe(t *testing.T) {
	nc, err := nats.Connect(connString)
	assert.NoError(t, err)

	t.Cleanup(func() {
		nc.Close()
	})

	tests := []struct {
		name      string
		getTasks  func(*gomock.Controller) websiteupdate.WebsiteUpdateTasks
		getRpo    func(*gomock.Controller) repository.Repostory
		publish   func(*testing.T, *nats.Conn)
		expectErr error
	}{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			task := NewTask(nc, test.getTasks(ctrl), test.getRpo(ctrl))

			ctx, err := task.Subscribe(t.Context())
			assert.ErrorIs(t, err, test.expectErr)

			test.publish(t, nc)
			time.Sleep(time.Millisecond)
			if err == nil {
				defer ctx.Stop()
			}
		})
	}
}

func TestWebsiteBatchUpdateTask_handler(t *testing.T) {
	nc, err := nats.Connect(connString)
	assert.NoError(t, err)
	t.Cleanup(func() {
		nc.Close()
	})

	web := model.Website{
		UUID:       "some uuid",
		Title:      "title",
		RawContent: "raw content",
		URL:        "https://example.com",
	}
	tests := []struct {
		name            string
		getTasks        func(*gomock.Controller) websiteupdate.WebsiteUpdateTasks
		getRpo          func(*gomock.Controller) repository.Repostory
		getMsg          func(*gomock.Controller) jetstream.Msg
		expectSubscribe func(*testing.T, *nats.Conn)
	}{
		{
			name: "happy flow",
			getTasks: func(ctrl *gomock.Controller) websiteupdate.WebsiteUpdateTasks {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Support(&web).Return(true).AnyTimes()
				serv.EXPECT().Name().Return("handler.happy_flow").AnyTimes()

				return websiteupdate.WebsiteUpdateTasks{
					websiteupdate.NewTask(nc, serv, nil, nil),
				}
			},
			getRpo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().FindWebsites().Return(
					[]model.Website{web},
					nil,
				).Times(1)

				return rpo
			},
			getMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				msg := mocknats.NewMockNatsMsg(ctrl)
				msg.EXPECT().Data().Return([]byte(`trigger`)).Times(1)
				msg.EXPECT().Ack().Return(nil).Times(1)
				return msg
			},
			expectSubscribe: func(t *testing.T, c *nats.Conn) {
				var gotMsg *nats.Msg
				sub, err := nc.Subscribe("web_history.websites.update.handler_happy_flow", func(msg *nats.Msg) {
					gotMsg = msg
					assert.Equal(
						t,
						`{"website":{"uuid":"some uuid","url":"https://example.com","title":"title","raw_content":"raw content","update_time":"0001-01-01T00:00:00Z"},"trace_id":"00000000000000000000000000000000","span_id":"0000000000000000","trace_flags":0}`,
						string(msg.Data),
					)
				})
				assert.NoError(t, err)
				time.Sleep(10 * time.Millisecond)
				sub.Unsubscribe()

				assert.NotNil(t, gotMsg)
			},
		},
		{
			name: "error/fail to find website",
			getTasks: func(ctrl *gomock.Controller) websiteupdate.WebsiteUpdateTasks {
				serv := mockvendor.NewMockVendorService(ctrl)

				return websiteupdate.WebsiteUpdateTasks{
					websiteupdate.NewTask(nil, serv, nil, nil),
				}
			},
			getRpo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().FindWebsites().Return(
					nil,
					errors.New("fail to query website"),
				).AnyTimes()

				return rpo
			},
			getMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				msg := mocknats.NewMockNatsMsg(ctrl)
				msg.EXPECT().Data().Return([]byte(`trigger`)).Times(1)
				msg.EXPECT().Ack().Return(nil).Times(1)
				return msg
			},
			expectSubscribe: func(t *testing.T, c *nats.Conn) {},
		},
		{
			name: "error/fail to ack",
			getTasks: func(ctrl *gomock.Controller) websiteupdate.WebsiteUpdateTasks {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Support(&web).Return(true).AnyTimes()
				serv.EXPECT().Name().Return("handler.error.fail_to_ack").AnyTimes()

				return websiteupdate.WebsiteUpdateTasks{
					websiteupdate.NewTask(nc, serv, nil, nil),
				}
			},
			getRpo: func(ctrl *gomock.Controller) repository.Repostory {
				rpo := mockrepo.NewMockRepostory(ctrl)
				rpo.EXPECT().FindWebsites().Return(
					[]model.Website{web},
					nil,
				).Times(1)

				return rpo
			},
			getMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				msg := mocknats.NewMockNatsMsg(ctrl)
				msg.EXPECT().Data().Return([]byte(`trigger`)).Times(1)
				msg.EXPECT().Ack().Return(errors.New("fail to ack")).Times(1)
				return msg
			},
			expectSubscribe: func(t *testing.T, c *nats.Conn) {
				var gotMsg *nats.Msg
				sub, err := nc.Subscribe("web_history.websites.update.handler_error_fail_to_ack", func(msg *nats.Msg) {
					gotMsg = msg
					assert.Equal(
						t,
						`{"website":{"uuid":"some uuid","url":"https://example.com","title":"title","raw_content":"raw content","update_time":"0001-01-01T00:00:00Z"},"trace_id":"00000000000000000000000000000000","span_id":"0000000000000000","trace_flags":0}`,
						string(msg.Data),
					)
				})
				assert.NoError(t, err)
				time.Sleep(10 * time.Millisecond)
				sub.Unsubscribe()

				assert.NotNil(t, gotMsg)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			task := NewTask(nc, test.getTasks(ctrl), test.getRpo(ctrl))
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				task.handler(test.getMsg(ctrl))
			}()

			test.expectSubscribe(t, nc)

			wg.Wait()
		})
	}
}

func TestWebsiteBatchUpdateTask_publishWebsiteUpdateTask(t *testing.T) {
	nc, err := nats.Connect(connString)
	assert.NoError(t, err)
	t.Cleanup(func() {
		nc.Close()
	})

	tests := []struct {
		name            string
		getTask         func(*gomock.Controller) websiteupdate.WebsiteUpdateTasks
		web             *model.Website
		expectSubscribe func(*testing.T, *nats.Conn)
	}{
		{
			name: "happy flow",
			getTask: func(ctrl *gomock.Controller) websiteupdate.WebsiteUpdateTasks {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Support(gomock.Any()).Return(true).AnyTimes()
				serv.EXPECT().Name().Return("publish.happy_flow").AnyTimes()

				return websiteupdate.WebsiteUpdateTasks{
					websiteupdate.NewTask(nc, serv, nil, nil),
				}
			},
			web: &model.Website{
				UUID:       "some uuid",
				Title:      "title",
				RawContent: "raw content",
				URL:        "https://example.com",
			},
			expectSubscribe: func(t *testing.T, c *nats.Conn) {
				var gotMsg *nats.Msg
				sub, err := c.Subscribe("web_history.websites.update.publish_happy_flow", func(msg *nats.Msg) {
					gotMsg = msg
					assert.Equal(
						t,
						`{"website":{"uuid":"some uuid","url":"https://example.com","title":"title","raw_content":"raw content","update_time":"0001-01-01T00:00:00Z"},"trace_id":"skipped_data","span_id":"skipped_data","trace_flags":1}`,
						regexp.MustCompile(`"(trace_id|span_id)":"\w+"`).ReplaceAllString(string(msg.Data), `"$1":"skipped_data"`),
					)
				})
				assert.NoError(t, err)
				time.Sleep(10 * time.Millisecond)
				sub.Unsubscribe()

				assert.NotNil(t, gotMsg)
			},
		},
		{
			name: "error/not support website",
			getTask: func(ctrl *gomock.Controller) websiteupdate.WebsiteUpdateTasks {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Support(gomock.Any()).Return(false).AnyTimes()
				serv.EXPECT().Name().Return("publish.error.not_support_website").AnyTimes()

				return websiteupdate.WebsiteUpdateTasks{
					websiteupdate.NewTask(nc, serv, nil, nil),
				}
			},
			web: &model.Website{
				UUID:       "some uuid",
				Title:      "title",
				RawContent: "raw content",
				URL:        "https://example.com",
			},
			expectSubscribe: func(t *testing.T, c *nats.Conn) {
				var gotMsg *nats.Msg
				sub, err := c.Subscribe("web_history.websites.update.publish_error_not_support_website", func(msg *nats.Msg) {
					gotMsg = msg
					assert.Nil(t, gotMsg)
				})
				assert.NoError(t, err)
				time.Sleep(10 * time.Millisecond)
				sub.Unsubscribe()

				assert.Nil(t, gotMsg)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			task := NewTask(nil, test.getTask(ctrl), nil)
			task.publishWebsiteUpdateTask(t.Context(), test.web)
		})
	}
}

func TestWebsiteBatchUpdateTask_hashData(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		expect string
	}{
		{
			name:   "empty",
			input:  []byte{},
			expect: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:   "test json",
			input:  []byte(`{"website":{"uuid":"","url":"https://example.com","title":"test","raw_content":"raw content","update_time":"2020-05-01T00:00:00Z"},"trace_id":"01234567890123456789012345678901","span_id":"0123456789012345","trace_flags":1}`),
			expect: "d98e03a372153f9f7980d08c66a2e7ca310dc2a2fd0ab5c881b5176222777426",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := hashData(test.input)
			assert.Equal(t, test.expect, result)
		})
	}
}
