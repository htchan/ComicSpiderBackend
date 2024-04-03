package websiteupdate

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/htchan/WebHistory/internal/executor"
	"github.com/htchan/WebHistory/internal/jobs"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TODO: add missing testcases
type Job struct {
	rpo            repository.Repostory
	vendorServices []vendors.VendorService
	sleepInterval  time.Duration
}

var _ executor.Job = (*Job)(nil)

func NewJob(rpo repository.Repostory, sleepInterval time.Duration, services []vendors.VendorService) *Job {
	return &Job{
		rpo:            rpo,
		sleepInterval:  sleepInterval,
		vendorServices: services,
	}
}

func (job *Job) Execute(ctx context.Context, p interface{}) error {
	params, ok := p.(Params)
	if !ok {
		return jobs.ErrInvalidParams
	}

	defer params.Cleanup()

	tr := otel.Tracer("htchan/WebHistory/update-jobs")
	if params.SpanContext != nil {
		ctx = trace.ContextWithSpanContext(ctx, *params.SpanContext)
	}

	updateCtx, updateSpan := tr.Start(ctx, "Update Website")
	defer updateSpan.End()

	updateSpan.SetAttributes(params.Web.OtelAttributes()...)
	updateSpan.SetAttributes(attribute.String("job_uuid", updateCtx.Value("job_uuid").(string)))

	var err error
	var executed bool
	for _, serv := range job.vendorServices {
		if serv.Support(params.Web) {
			executed = true
			updateErr := serv.Update(updateCtx, params.Web)
			err = errors.Join(err, updateErr)
		}
	}

	_, sleepSpan := tr.Start(updateCtx, "Sleep After Update")
	defer sleepSpan.End()
	time.Sleep(job.sleepInterval)

	runtime.GC()

	if !executed {
		return fmt.Errorf("execute failed: %w: %s", ErrNotSupportedHost, params.Web.Host())
	}

	return err
}
