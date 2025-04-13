package websiteupdate

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/htchan/WebHistory/internal/config"
	mockvendor "github.com/htchan/WebHistory/internal/mock/vendor"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestNewTaskSet(t *testing.T) {
	tests := []struct {
		name   string
		nc     *nats.Conn
		serv   []vendors.VendorService
		rpo    repository.Repostory
		conf   *config.WebsiteConfig
		expect WebsiteUpdateTasks
	}{
		{
			name:   "happy flow/empty services",
			nc:     nil,
			serv:   nil,
			rpo:    nil,
			conf:   nil,
			expect: WebsiteUpdateTasks{},
		},
		{
			name: "happy flow/non empty services",
			nc:   nil,
			serv: []vendors.VendorService{nil},
			rpo:  nil,
			conf: nil,
			expect: WebsiteUpdateTasks{
				&WebsiteUpdateTask{
					nc:          nil,
					Service:     nil,
					rpo:         nil,
					websiteConf: nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewTaskSet(tt.nc, tt.serv, tt.rpo, tt.conf)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestWebsiteUpdateTasks_Publish(t *testing.T) {
	nc, err := nats.Connect(connString)
	assert.NoError(t, err)

	t.Cleanup(func() {
		nc.Close()
	})

	tests := []struct {
		name            string
		getServs        func(*gomock.Controller) []vendors.VendorService
		web             *model.Website
		expect          []string
		expectErr       error
		expectSubscribe func(t *testing.T, nc *nats.Conn)
	}{
		{
			name: "happy flow/one supported service found",
			getServs: func(ctrl *gomock.Controller) []vendors.VendorService {
				serv := mockvendor.NewMockVendorService(ctrl)
				serv.EXPECT().Support(
					&model.Website{URL: "https://example.com", UUID: "some uuid"},
				).Return(true).AnyTimes()
				serv.EXPECT().Name().Return("set_publish.happy_flow_one_supported").AnyTimes()

				return []vendors.VendorService{serv}
			},
			web:       &model.Website{URL: "https://example.com", UUID: "some uuid"},
			expect:    []string{"set_publish.happy_flow_one_supported"},
			expectErr: nil,
			expectSubscribe: func(t *testing.T, nc *nats.Conn) {
				var gotMsg *nats.Msg
				sub, err := nc.Subscribe("web_history.websites.update.set_publish_happy_flow_one_supported", func(msg *nats.Msg) {
					gotMsg = msg
					assert.Equal(t, `{"website":{"uuid":"some uuid","url":"https://example.com","title":"","raw_content":"","update_time":"0001-01-01T00:00:00Z"},"trace_id":"00000000000000000000000000000000","span_id":"0000000000000000","trace_flags":0}`, string(msg.Data))
				})
				assert.NoError(t, err)
				time.Sleep(time.Millisecond)
				sub.Unsubscribe()
				assert.NotNil(t, gotMsg, "no message received")
			},
		},
		{
			name: "happy flow/multiple supported service found",
			getServs: func(c *gomock.Controller) []vendors.VendorService {
				serv1 := mockvendor.NewMockVendorService(c)
				serv1.EXPECT().Support(
					&model.Website{URL: "https://example.com", UUID: "some uuid"},
				).Return(true).AnyTimes()
				serv1.EXPECT().Name().Return("set_publish.happy_flow_multi_supported_1").AnyTimes()

				serv2 := mockvendor.NewMockVendorService(c)
				serv2.EXPECT().Support(
					&model.Website{URL: "https://example.com", UUID: "some uuid"},
				).Return(true).AnyTimes()
				serv2.EXPECT().Name().Return("set_publish.happy_flow_multi_supported_2").AnyTimes()

				return []vendors.VendorService{serv1, serv2}
			},
			web:       &model.Website{URL: "https://example.com", UUID: "some uuid"},
			expect:    []string{"set_publish.happy_flow_multi_supported_1", "set_publish.happy_flow_multi_supported_2"},
			expectErr: nil,
			expectSubscribe: func(t *testing.T, nc *nats.Conn) {
				var gotMsg1, gotMsg2 *nats.Msg
				sub1, err1 := nc.Subscribe("web_history.websites.update.set_publish_happy_flow_multi_supported_1", func(msg *nats.Msg) {
					gotMsg1 = msg
					assert.Equal(t, `{"website":{"uuid":"some uuid","url":"https://example.com","title":"","raw_content":"","update_time":"0001-01-01T00:00:00Z"},"trace_id":"00000000000000000000000000000000","span_id":"0000000000000000","trace_flags":0}`, string(msg.Data))
				})
				sub2, err2 := nc.Subscribe("web_history.websites.update.set_publish_happy_flow_multi_supported_2", func(msg *nats.Msg) {
					gotMsg2 = msg
					assert.Equal(t, `{"website":{"uuid":"some uuid","url":"https://example.com","title":"","raw_content":"","update_time":"0001-01-01T00:00:00Z"},"trace_id":"00000000000000000000000000000000","span_id":"0000000000000000","trace_flags":0}`, string(msg.Data))
				})

				assert.NoError(t, err1)
				assert.NoError(t, err2)
				time.Sleep(50 * time.Millisecond)
				sub1.Unsubscribe()
				sub2.Unsubscribe()
				assert.NotNil(t, gotMsg1, "no message received in first queue")
				assert.NotNil(t, gotMsg2, "no message received in second queue")
			},
		},
		{
			name: "error/no supported service",
			getServs: func(c *gomock.Controller) []vendors.VendorService {
				return nil
			},
			web:             &model.Website{URL: "https://example.com", UUID: "some uuid"},
			expect:          nil,
			expectErr:       ErrNotSupportedWebsite,
			expectSubscribe: func(t *testing.T, nc *nats.Conn) {},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tasks := NewTaskSet(nc, test.getServs(ctrl), nil, nil)

			go func() {
				result, err := tasks.Publish(context.Background(), test.web)
				assert.Equal(t, test.expect, result)
				assert.ErrorIs(t, err, test.expectErr)
			}()
			test.expectSubscribe(t, nc)
		})
	}
}
