package websiteupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func Test_ParamsFromData(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		data         []byte
		conf         *config.WebsiteConfig
		expectParams *WebsiteUpdateParams
		expectErr    error
	}{
		{
			name: "happy flow/with trace",
			data: []byte(`{"website":{"uuid":"test uuid","url":"https://example.com","title":"test","raw_content":"test content","update_time":"2020-05-01T00:00:00Z"},"trace_id":"01234567890123456789012345678901","span_id":"0123456789012345","trace_flags":1}`),
			conf: &config.WebsiteConfig{},
			expectParams: &WebsiteUpdateParams{
				Website: model.Website{
					UUID:       "test uuid",
					URL:        "https://example.com",
					Title:      "test",
					RawContent: "test content",
					UpdateTime: time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC),
					Conf:       &config.WebsiteConfig{},
				},
				TraceID:    "01234567890123456789012345678901",
				SpanID:     "0123456789012345",
				TraceFlags: 0x1,
			},
			expectErr: nil,
		},
		{
			name: "happy flow/without trace",
			data: []byte(`{"website":{"uuid":"test uuid","url":"https://example.com","title":"test","raw_content":"test content","update_time":"2020-05-01T00:00:00Z"}}`),
			conf: &config.WebsiteConfig{},
			expectParams: &WebsiteUpdateParams{
				Website: model.Website{
					UUID:       "test uuid",
					URL:        "https://example.com",
					Title:      "test",
					RawContent: "test content",
					UpdateTime: time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC),
					Conf:       &config.WebsiteConfig{},
				},
			},
			expectErr: nil,
		},
		{
			name:         "error/invalid json",
			data:         []byte(`abc`),
			conf:         &config.WebsiteConfig{},
			expectParams: nil,
			expectErr:    &json.SyntaxError{},
		},
		{
			name: "error/missing website",
			data: []byte(`{"trace_id":"01234567890123456789012345678901","span_id":"0123456789012345","trace_flags":1}`),
			conf: &config.WebsiteConfig{},
			expectParams: &WebsiteUpdateParams{
				TraceID:    "01234567890123456789012345678901",
				SpanID:     "0123456789012345",
				TraceFlags: 0x1,
			},
			expectErr: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, params, err := ParamsFromData(context.Background(), test.data, test.conf)

			if test.expectErr != nil {
				assert.ErrorAs(t, err, &test.expectErr, "different error")
			}
			assert.Equal(t, test.expectParams, params, "different params")
			if params != nil && params.TraceID != "" && params.SpanID != "" {
				assert.Equal(t, trace.SpanContextFromContext(ctx).TraceID().String(), params.TraceID, "different trace id")
				assert.Equal(t, trace.SpanContextFromContext(ctx).SpanID().String(), params.SpanID, "different span id")
				assert.Equal(t, trace.SpanContextFromContext(ctx).TraceFlags(), trace.TraceFlags(params.TraceFlags), "different trace flags")
			}
		})
	}
}

func TestWebsiteUpdateParams_MarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		params      *WebsiteUpdateParams
		expect      string
		expectError error
	}{
		{
			name: "success",
			params: &WebsiteUpdateParams{
				Website: model.Website{
					UUID:       "test uuid",
					URL:        "https://example.com",
					Title:      "test title",
					RawContent: "test content",
					UpdateTime: time.Date(2020, time.May, 1, 0, 0, 0, 0, time.UTC),
				},
				TraceID:    "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
				SpanID:     "XXXXXXXXXXXXXXXX",
				TraceFlags: 0x1,
			},
			expect:      `{"website":{"uuid":"test uuid","url":"https://example.com","title":"test title","raw_content":"test content","update_time":"2020-05-01T00:00:00Z"},"trace_id":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX","span_id":"XXXXXXXXXXXXXXXX","trace_flags":1}`,
			expectError: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(test.params)
			assert.Equal(t, test.expect, string(data))
			assert.ErrorIs(t, err, test.expectError)
		})
	}
}

func TestWebsiteUpdateParams_ToData(t *testing.T) {
	emptyCtx := context.Background()
	spanCtx, span := otel.Tracer("test").Start(emptyCtx, "test")
	span.End()

	tests := []struct {
		name        string
		ctx         context.Context
		params      *WebsiteUpdateParams
		expect      string
		expectError error
	}{
		{
			name: "success/with span",
			ctx:  spanCtx,
			params: &WebsiteUpdateParams{
				Website: model.Website{
					UUID:       "test uuid",
					URL:        "https://example.com",
					Title:      "test",
					RawContent: "test content",
					UpdateTime: time.Date(2020, time.May, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expect: fmt.Sprintf(
				`{"website":{"uuid":"test uuid","url":"https://example.com","title":"test","raw_content":"test content","update_time":"2020-05-01T00:00:00Z"},"trace_id":"%s","span_id":"%s","trace_flags":1}`,
				span.SpanContext().TraceID().String(),
				span.SpanContext().SpanID().String(),
			),
			expectError: nil,
		},
		{
			name: "success/without span",
			ctx:  emptyCtx,
			params: &WebsiteUpdateParams{
				Website: model.Website{
					UUID:       "test uuid",
					URL:        "https://example.com",
					Title:      "test",
					RawContent: "test content",
					UpdateTime: time.Date(2020, time.May, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expect:      `{"website":{"uuid":"test uuid","url":"https://example.com","title":"test","raw_content":"test content","update_time":"2020-05-01T00:00:00Z"},"trace_id":"00000000000000000000000000000000","span_id":"0000000000000000","trace_flags":0}`,
			expectError: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			data, err := test.params.ToData(test.ctx)
			assert.Equal(t, test.expect, string(data))
			assert.ErrorIs(t, err, test.expectError)
		})
	}

}
