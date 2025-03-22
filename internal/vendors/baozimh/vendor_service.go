package baozimh

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"golang.org/x/sync/semaphore"
)

type VendorService struct {
	cli  *http.Client
	repo repository.Repostory
	lock *semaphore.Weighted
	cfg  *config.VendorServiceConfig
}

var _ vendors.VendorService = (*VendorService)(nil)

var (
	titleGoQuery = "head>title"
	dateGoQuery  = "div.supporting-text>div>span>em"
	// contentGoQuery = "div.comics-detail__info>div.supporting-text>div:nth-child(2)>span>em"
	// fromIndex      = 0
	// toIndex        = 2
	Host              = "baozimh.com"
	dateFormat        = "2006年01月02日"
	dateExtractRegexp = regexp.MustCompile(`\((.*) 更新\)`)
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
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("Failed to parse HTML")

		return false
	}

	isUpdated := false

	title := doc.Find(titleGoQuery).Text()
	if web.Title == "" && title != web.Title {
		web.Title = title
		isUpdated = true
	}

	var (
		updateTimeStr = dateExtractRegexp.FindStringSubmatch(doc.Find(dateGoQuery).Text())
		updateTime    time.Time
	)

	if len(updateTimeStr) < 2 {
		zerolog.Ctx(ctx).Warn().Str("update_time_str", doc.Find(dateGoQuery).Text()).Msg("cannot find update time str")

		return isUpdated
	}

	if strings.Contains(updateTimeStr[1], "小時前") {
		hoursAgo, err := strconv.Atoi(strings.Trim(updateTimeStr[1], "小時前"))
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Str("date", updateTimeStr[1]).Msg("Failed to parse update time")
		} else {
			updateTime = time.Now().
				Add(time.Duration(-hoursAgo) * time.Hour)
		}
	} else {
		updateTime, err = time.Parse(dateFormat, updateTimeStr[1])
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Str("date", updateTimeStr[1]).Msg("Failed to parse update time")
		}
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
	tr := otel.Tracer("htchan/WebHistory/vendors/baozimh")

	_, fetchWebSpan := tr.Start(ctx, "fetch website")
	body, fetchErr := serv.fetchWebsite(ctx, web)
	fetchWebSpan.End()

	if fetchErr != nil {
		fetchWebSpan.RecordError(fetchErr)

		return fetchErr
	}

	if serv.isUpdated(ctx, web, body) {
		_, repoSpan := tr.Start(ctx, "update db record")
		repoErr := serv.repo.UpdateWebsite(web)
		repoSpan.End()

		if repoErr != nil {
			repoSpan.RecordError(repoErr)

			return repoErr
		}
	}

	return nil
}
