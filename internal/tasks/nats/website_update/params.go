package websiteupdate

import (
	"context"
	"encoding/json"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type WebsiteUpdateParams struct {
	Website    model.Website `json:"website"`
	TraceID    string        `json:"trace_id"`
	SpanID     string        `json:"span_id"`
	TraceFlags byte          `json:"trace_flags"`
}

func ParamsFromData(ctx context.Context, data []byte, conf *config.WebsiteConfig) (context.Context, *WebsiteUpdateParams, error) {
	// parse message body
	params := new(WebsiteUpdateParams)
	if jsonErr := json.Unmarshal(data, params); jsonErr != nil {
		return ctx, nil, jsonErr
	}

	if params.Website.URL != "" {
		params.Website.UpdateTime = params.Website.UpdateTime.UTC()
		params.Website.Conf = conf
	}

	ctx = log.With().
		Str("trace_id", params.TraceID).
		Str("host", params.Website.Host()).
		Str("website_uuid", params.Website.UUID).
		Str("website_url", params.Website.URL).
		Str("website_title", params.Website.Title).
		Logger().WithContext(ctx)

	if params.TraceID != "" && params.SpanID != "" {
		traceID, traceErr := trace.TraceIDFromHex(params.TraceID)
		spanID, spanErr := trace.SpanIDFromHex(params.SpanID)
		if traceErr != nil || spanErr != nil {
			return ctx, params, nil
		}

		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.TraceFlags(params.TraceFlags),
			Remote:     true, // Indicate that this span context is from a remote service
		})
		ctx := trace.ContextWithSpanContext(ctx, spanContext)

		return ctx, params, nil
	}

	return ctx, params, nil
}

func (params WebsiteUpdateParams) MarshalJSON() ([]byte, error) {
	type Alias WebsiteUpdateParams
	type WebsiteParams struct {
		UUID       string    `json:"uuid"`
		URL        string    `json:"url"`
		Title      string    `json:"title"`
		RawContent string    `json:"raw_content"`
		UpdateTime time.Time `json:"update_time"`
	}
	return json.Marshal(&struct {
		Website WebsiteParams `json:"website"`
		Alias
	}{
		Alias: Alias(params),
		Website: WebsiteParams{
			UUID:       params.Website.UUID,
			URL:        params.Website.URL,
			Title:      params.Website.Title,
			RawContent: params.Website.RawContent,
			UpdateTime: params.Website.UpdateTime.UTC(),
		},
	})
}

func (params *WebsiteUpdateParams) ToData(ctx context.Context) ([]byte, error) {
	spanCtx := trace.SpanContextFromContext(ctx)
	params.TraceID = spanCtx.TraceID().String()
	params.SpanID = spanCtx.SpanID().String()
	params.TraceFlags = byte(spanCtx.TraceFlags())

	return json.Marshal(params)
}
