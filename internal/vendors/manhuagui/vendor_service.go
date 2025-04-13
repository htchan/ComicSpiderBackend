package manhuagui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/semaphore"
)

type VendorService struct {
	cli  *http.Client
	repo repository.Repostory
	lock *semaphore.Weighted
	cfg  *config.VendorServiceConfig
}

var _ vendors.VendorService = (*VendorService)(nil)

const (
	titleGoQuery = "head>title"
	dateGoQuery  = "li.status>span>span.red:nth-child(3)"
	// contentGoQuery = "li.status>span>span.red"
	// fromIndex      = 0
	// toIndex        = 2
	Host       = "manhuagui.com"
	dateFormat = "2006-01-02"
)

func NewVendorService(
	cli *http.Client,
	repo repository.Repostory,
	cfg *config.VendorServiceConfig,
) *VendorService {
	return &VendorService{
		cli:  cli,
		repo: repo,
		lock: semaphore.NewWeighted(cfg.MaxConcurrency),
		cfg:  cfg,
	}
}

func (serv *VendorService) Name() string {
	return Host
}

func (serv *VendorService) fetchWebsite(ctx context.Context, web *model.Website) (string, error) {
	if serv.lock.Acquire(ctx, 1) == nil {
		defer func() {
			time.Sleep(serv.cfg.FetchInterval)
			serv.lock.Release(1)
		}()
	}

	url := regexp.MustCompile(fmt.Sprintf("^(http.*?)://.*?%s(.*)$", Host)).
		ReplaceAllString(web.URL, fmt.Sprintf("$1://www.%s$2", Host))

	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		return "", reqErr
	}

	var resp *http.Response
	var respErr error

	// send request with basic retry
	for i := 0; i < serv.cfg.MaxRetry; i++ {
		resp, respErr = serv.cli.Do(req.WithContext(ctx))
		defer func(resp *http.Response) {
			if resp != nil {
				resp.Body.Close()
			}
		}(resp)

		if respErr == nil && (resp.StatusCode >= 200 && resp.StatusCode < 300) {
			break
		}

		if respErr == nil {
			if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
				respErr = fmt.Errorf("fetch website failed: %w (%d)", vendors.ErrInvalidStatusCode, resp.StatusCode)
			} else {
				respErr = fmt.Errorf("fetch website failed: unknown error")
			}
		}

		time.Sleep(time.Duration(i+1) * serv.cfg.RetryInterval)
	}
	if respErr != nil {
		return "", respErr
	}

	data, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return "", bodyErr
	}

	return string(data), nil
}

func (serv *VendorService) isUpdated(ctx context.Context, web *model.Website, body string) bool {
	tr := otel.Tracer("htchan/WebHistory/vendors/manhuagui")

	_, checkUpdateSpan := tr.Start(ctx, "check update")
	defer checkUpdateSpan.End()

	oldTitle := web.Title
	oldUpdateTime := web.UpdateTime
	defer func() {
		attrs := make([]attribute.KeyValue, 0, 6)
		if oldTitle != web.Title {
			attrs = append(
				attrs,
				attribute.Bool("title_updated", true),
				attribute.String("old_title", oldTitle),
				attribute.String("new_title", web.Title),
			)
		}

		if oldUpdateTime != web.UpdateTime {
			attrs = append(
				attrs,
				attribute.Bool("update_time_updated", true),
				attribute.String("old_update_time", oldUpdateTime.String()),
				attribute.String("new_update_time", web.UpdateTime.String()),
			)
		}

		checkUpdateSpan.SetAttributes(attrs...)
	}()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("Failed to parse HTML")
		checkUpdateSpan.SetStatus(codes.Error, err.Error())
		checkUpdateSpan.RecordError(err)

		return false
	}

	isUpdated := false

	title := doc.Find(titleGoQuery).Text()
	if web.Title == "" && title != web.Title {
		web.Title = title
		isUpdated = true
	}

	updateTimeStr := doc.Find(dateGoQuery).Text()

	updateTime, err := time.Parse(dateFormat, updateTimeStr)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("date", updateTimeStr).Msg("Failed to parse update time")
		checkUpdateSpan.SetStatus(codes.Error, err.Error())
		checkUpdateSpan.RecordError(err)
	}

	updateTime = updateTime.UTC().Truncate(24 * time.Hour)
	if updateTime.After(web.UpdateTime) {
		web.UpdateTime = updateTime
		isUpdated = true
	}

	return isUpdated
}

func (serv *VendorService) Support(web *model.Website) bool {
	return web.Host() == Host
}

func (serv *VendorService) Update(ctx context.Context, web *model.Website) error {
	tr := otel.Tracer("htchan/WebHistory/vendors/manhuagui")

	_, fetchWebSpan := tr.Start(ctx, "fetch website")
	defer fetchWebSpan.End()

	fetchWebSpan.SetAttributes(
		append(
			web.OtelAttributes(),
			attribute.String("vendor", serv.Name()),
		)...,
	)
	body, fetchErr := serv.fetchWebsite(ctx, web)
	if fetchErr != nil {
		fetchWebSpan.SetStatus(codes.Error, fetchErr.Error())
		fetchWebSpan.RecordError(fetchErr)

		return fetchErr
	}

	fetchWebSpan.End()

	if serv.isUpdated(ctx, web, body) {
		_, repoSpan := tr.Start(ctx, "update db record")
		defer repoSpan.End()

		repoSpan.SetAttributes(
			attribute.String("updated_title", web.Title),
			attribute.String("updated_content", web.RawContent),
			attribute.String("updated_time", web.UpdateTime.String()),
		)

		repoErr := serv.repo.UpdateWebsite(web)
		if repoErr != nil {
			repoSpan.SetStatus(codes.Error, repoErr.Error())
			repoSpan.RecordError(repoErr)

			return repoErr
		}

		repoSpan.End()
	}

	return nil
}
