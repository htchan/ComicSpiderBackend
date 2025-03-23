package websiteupdate

import (
	"context"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type WebsiteUpdateParams struct {
	Website     model.Website `json:"website"`
	traceParent string
}

func FromMap(data map[string]interface{}, conf *config.WebsiteConfig) WebsiteUpdateParams {
	updateTime, err := time.Parse(time.RFC3339, data["website_update_time"].(string))
	if err != nil {
		log.Warn().Err(err).Str("time", data["website_update_time"].(string)).Msg("failed to parse update time, use current time instead")
		// use current time as fallback
		updateTime = time.Now().UTC()
	}

	return WebsiteUpdateParams{
		Website: model.Website{
			UUID:       data["website_uuid"].(string),
			URL:        data["website_url"].(string),
			Title:      data["website_title"].(string),
			RawContent: data["website_raw_content"].(string),
			UpdateTime: updateTime.UTC(),
			Conf:       conf,
		},
		traceParent: data["traceparent"].(string),
	}
}

func (params WebsiteUpdateParams) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"website_uuid":        params.Website.UUID,
		"website_url":         params.Website.URL,
		"website_title":       params.Website.Title,
		"website_raw_content": params.Website.RawContent,
		"website_update_time": params.Website.UpdateTime,
		"traceparent":         params.traceParent,
	}
}

func (params WebsiteUpdateParams) UpdateCtxOtel(ctx context.Context) context.Context {
	if params.traceParent == "" {
		return ctx
	}

	carrier := propagation.MapCarrier{}
	carrier.Set("traceparent", params.traceParent)

	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
