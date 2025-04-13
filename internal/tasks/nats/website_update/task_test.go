package websiteupdate

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/htchan/WebHistory/internal/config"
	mocknats "github.com/htchan/WebHistory/internal/mock/nats"
	mockvendor "github.com/htchan/WebHistory/internal/mock/vendor"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
)

func TestNewTask(t *testing.T) {
	tests := []struct {
		name         string
		nc           *nats.Conn
		serv         vendors.VendorService
		rpo          repository.Repostory
		webConf      *config.WebsiteConfig
		expectedTask *WebsiteUpdateTask
	}{
		{
			name: "assign parameters to correct places",
			nc:   nil,
			serv: nil,
			rpo:  nil,
			webConf: &config.WebsiteConfig{
				Separator:     ",",
				MaxDateLength: 2,
			},
			expectedTask: &WebsiteUpdateTask{
				nc:      nil,
				Service: nil,
				rpo:     nil,
				websiteConf: &config.WebsiteConfig{
					Separator:     ",",
					MaxDateLength: 2,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			task := NewTask(test.nc, test.serv, test.rpo, test.webConf)

			assert.Equal(t, test.expectedTask, task)
		})
	}
}

func TestWebsiteUpdateTask_subject(t *testing.T) {
	tests := []struct {
		name    string
		getServ func(*gomock.Controller) vendors.VendorService
		expect  string
	}{
		{
			name: "subject of service name without .",
			getServ: func(ctrl *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Name().Return("website_update").AnyTimes()

				return serv
			},
			expect: "web_history.websites.update.website_update",
		},
		{
			name: "subject of service name with .",
			getServ: func(ctrl *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Name().Return("example.com").AnyTimes()

				return serv
			},
			expect: "web_history.websites.update.example_com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			task := NewTask(nil, test.getServ(ctrl), nil, nil)

			result := task.subject()
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestWebsiteUpdateTask_Publish(t *testing.T) {
	nc, err := nats.Connect(connString)
	assert.NoError(t, err)
	t.Cleanup(func() {
		nc.Close()
	})

	tests := []struct {
		name            string
		getServ         func(*gomock.Controller) vendors.VendorService
		web             *model.Website
		expectSubscribe func(*testing.T, *nats.Conn)
		expectErr       error
	}{
		{
			name: "publish success",
			getServ: func(ctrl *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Name().Return("publish_success").AnyTimes()

				return serv
			},
			web: &model.Website{
				UUID: "some uuid",
				URL:  "https://example.com",
			},
			expectSubscribe: func(t *testing.T, nc *nats.Conn) {
				var gotMsg *nats.Msg
				sub, err := nc.Subscribe("web_history.websites.update.publish_success", func(msg *nats.Msg) {
					gotMsg = msg
					assert.Equal(t, `{"website":{"uuid":"some uuid","url":"https://example.com","title":"","raw_content":"","update_time":"0001-01-01T00:00:00Z"},"trace_id":"00000000000000000000000000000000","span_id":"0000000000000000","trace_flags":0}`, string(msg.Data))
				})
				assert.NoError(t, err)
				time.Sleep(time.Millisecond)
				sub.Unsubscribe()

				assert.NotNil(t, gotMsg)
			},
			expectErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			task := NewTask(nc, test.getServ(ctrl), nil, nil)

			go func() {
				err := task.Publish(t.Context(), test.web)
				assert.ErrorIs(t, err, test.expectErr)
			}()

			test.expectSubscribe(t, nc)
		})
	}
}

func TestWebsiteUpdateTask_Subscribe(t *testing.T) {
	nc, err := nats.Connect(connString)
	assert.NoError(t, err)
	t.Cleanup(func() {
		nc.Close()
	})

	tests := []struct {
		name      string
		getServ   func(*gomock.Controller) vendors.VendorService
		publish   func(*testing.T, *nats.Conn)
		expectErr error
	}{
		{
			name: "happy flow",
			getServ: func(ctrl *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(ctrl)
				web := &model.Website{
					UUID:       "",
					URL:        "https://example.com",
					Title:      "test",
					UpdateTime: time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC),
				}
				serv.EXPECT().Name().Return("subscribe.happy_flow").AnyTimes()
				serv.EXPECT().Support(web).Return(true)
				serv.EXPECT().Update(gomock.Any(), web).Return(nil)

				return serv
			},
			publish: func(t *testing.T, nc *nats.Conn) {
				err := nc.Publish("web_history.websites.update.subscribe_happy_flow", []byte(`{"website":{"uuid":"","url":"https://example.com","title":"test","update_time":"2020-05-01T00:00:00Z"},"trace_id":"01234567890123456789012345678901","span_id":"0123456789012345","trace_flags":1}`))
				assert.NoError(t, err)
			},
			expectErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			task := NewTask(nc, test.getServ(ctrl), nil, nil)

			ctx, err := task.Subscribe(t.Context())
			assert.ErrorIs(t, err, test.expectErr)

			test.publish(t, nc)
			time.Sleep(100 * time.Millisecond)
			if err == nil {
				defer ctx.Stop()
			}
		})
	}

	t.Run("consume each message once", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		web := &model.Website{
			UUID:       "",
			URL:        "https://example.com",
			Title:      "test",
			UpdateTime: time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC),
		}
		web2 := &model.Website{
			UUID:       "",
			URL:        "https://example2.com",
			Title:      "test2",
			UpdateTime: time.Date(2020, 5, 2, 0, 0, 0, 0, time.UTC),
		}
		serv := mockvendor.NewMockVendorService(ctrl)
		serv.EXPECT().Name().Return("subscribe.each_message_once").AnyTimes()
		serv.EXPECT().Support(web).Return(true).Times(1)
		serv.EXPECT().Update(gomock.Any(), web).Return(nil).Times(1)
		serv.EXPECT().Support(web2).Return(true).Times(1)
		serv.EXPECT().Update(gomock.Any(), web2).Return(nil).Times(1)

		task := NewTask(nc, serv, nil, nil)

		ctx, err := task.Subscribe(t.Context())
		assert.NoError(t, err)
		err = nc.Publish("web_history.websites.update.subscribe_each_message_once", []byte(`{"website":{"uuid":"", "url":"https://example.com", "title":"test", "update_time":"2020-05-01T00:00:00Z"}, "trace_id":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", "span_id":"XXXXXXXXXXXXXXXX", "trace_flags":1}`))
		assert.NoError(t, err)
		time.Sleep(time.Millisecond)
		ctx.Stop()

		ctx2, err := task.Subscribe(t.Context())
		assert.NoError(t, err)
		err = nc.Publish("web_history.websites.update.subscribe_each_message_once", []byte(`{"website":{"uuid":"", "url":"https://example2.com", "title":"test2", "update_time":"2020-05-02T00:00:00Z"}, "trace_id":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", "span_id":"XXXXXXXXXXXXXXXX", "trace_flags":1}`))
		assert.NoError(t, err)
		time.Sleep(time.Millisecond)

		ctx2.Stop()
	})
}

func TestWebsiteUpdateTask_Validate(t *testing.T) {
	tests := []struct {
		name      string
		getServ   func(*gomock.Controller) vendors.VendorService
		params    *WebsiteUpdateParams
		expectErr error
	}{
		{
			name: "supported web",
			getServ: func(c *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(c)
				serv.EXPECT().Name().Return("supported_web").AnyTimes()
				serv.EXPECT().Support(
					&model.Website{URL: "https://example.com"},
				).Return(true)

				return serv
			},
			params: &WebsiteUpdateParams{
				Website: model.Website{URL: "https://example.com"},
			},
			expectErr: nil,
		},
		{
			name: "unsupported web",
			getServ: func(c *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(c)
				serv.EXPECT().Name().Return("supported_web").AnyTimes()
				serv.EXPECT().Support(
					&model.Website{URL: "https://example.com"},
				).Return(false)

				return serv
			},
			params: &WebsiteUpdateParams{
				Website: model.Website{URL: "https://example.com"},
			},
			expectErr: ErrNotSupportedWebsite,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			task := NewTask(nil, test.getServ(ctrl), nil, nil)

			err := task.Validate(t.Context(), test.params)

			assert.ErrorIs(t, err, test.expectErr)
		})
	}
}

func TestWebsiteUpdateTask_handler(t *testing.T) {
	tests := []struct {
		name    string
		getServ func(*gomock.Controller) vendors.VendorService
		getMsg  func(*gomock.Controller) jetstream.Msg
	}{
		{
			name: "happy flow",
			getServ: func(ctrl *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(ctrl)
				web := &model.Website{
					UUID:       "test uuid",
					URL:        "https://example.com",
					Title:      "test",
					RawContent: "content",
					UpdateTime: time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC),
				}
				serv.EXPECT().Name().Return("happy_flow").AnyTimes()
				serv.EXPECT().Support(web).Return(true)
				serv.EXPECT().Update(gomock.Any(), web).Return(nil)

				return serv
			},
			getMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				msg := mocknats.NewMockNatsMsg(ctrl)
				msg.EXPECT().Data().Return([]byte(`{"website":{"uuid":"test uuid","url":"https://example.com","title":"test","raw_content":"content","update_time":"2020-05-01T00:00:00Z"},"trace_id":"01234567890123456789012345678901","span_id":"0123456789012345","trace_flags":1}`))
				msg.EXPECT().Ack()

				return msg
			},
		},
		{
			name: "error/not supported web",
			getServ: func(ctrl *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Name().Return("not_supported_web").AnyTimes()
				serv.EXPECT().Support(&model.Website{
					UUID:       "test uuid",
					URL:        "https://example.com",
					Title:      "test",
					RawContent: "content",
					UpdateTime: time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC),
				}).Return(false)

				return serv
			},
			getMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				msg := mocknats.NewMockNatsMsg(ctrl)
				msg.EXPECT().Data().Return([]byte(`{"website":{"uuid":"test uuid", "url":"https://example.com", "title":"test", "raw_content":"content", "update_time":"2020-05-01T00:00:00Z"}, "trace_id":"01234567890123456789012345678901", "span_id":"0123456789012345", "trace_flags":1}`))
				msg.EXPECT().Ack()

				return msg
			},
		},
		{
			name: "error/non json data",
			getServ: func(ctrl *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Name().Return("non_json_data").AnyTimes()

				return serv
			},
			getMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				msg := mocknats.NewMockNatsMsg(ctrl)
				msg.EXPECT().Data().Return([]byte(`non json data`)).Times(2)
				msg.EXPECT().Ack()

				return msg
			},
		},
		{
			name: "error/fail to ack",
			getServ: func(ctrl *gomock.Controller) vendors.VendorService {
				serv := mockvendor.NewMockVendorService(ctrl)
				web := &model.Website{
					UUID:       "test uuid",
					URL:        "https://example.com",
					Title:      "test",
					RawContent: "content",
					UpdateTime: time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC),
				}
				serv.EXPECT().Name().Return("happy_flow").AnyTimes()
				serv.EXPECT().Support(web).Return(true)
				serv.EXPECT().Update(gomock.Any(), web).Return(nil)

				return serv
			},
			getMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				msg := mocknats.NewMockNatsMsg(ctrl)
				msg.EXPECT().Data().Return([]byte(`{"website":{"uuid":"test uuid","url":"https://example.com","title":"test","raw_content":"content","update_time":"2020-05-01T00:00:00Z"},"trace_id":"01234567890123456789012345678901","span_id":"0123456789012345","trace_flags":1}`))
				msg.EXPECT().Ack().Return(errors.New("ack error"))

				return msg
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			task := NewTask(nil, test.getServ(ctrl), nil, nil)

			task.handler(test.getMsg(ctrl))
		})
	}
}
