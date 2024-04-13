package websiteupdate

import (
	"github.com/htchan/WebHistory/internal/model"
	"go.opentelemetry.io/otel/trace"
)

type Params struct {
	SpanContext *trace.SpanContext `json:"-"`
	Web         *model.Website     `json:"web"`
	Cleanup     func()             `json:"-"`
}
