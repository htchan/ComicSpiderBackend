package baozimh

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
	titleGoQuery   = "head>title"
	contentGoQuery = "div.bot>div.fl>span"
	fromIndex      = 0
	toIndex        = 2
	host           = "u17.com"
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

func (serv *VendorService) fetchWebsite(ctx context.Context, web *model.Website) (string, error) {
	serv.lock.Acquire(ctx, 1)
	defer func() {
		time.Sleep(serv.cfg.FetchInterval)
		serv.lock.Release(1)
	}()

	url := regexp.MustCompile(fmt.Sprintf("^(http.*?)://.*?%s(.*)$", host)).
		ReplaceAllString(web.URL, fmt.Sprintf("$1://www.%s$2", host))

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

	var content []string
	doc.Find(contentGoQuery).Each(func(i int, s *goquery.Selection) {
		content = append(content, strings.TrimSpace(s.Text()))
	})

	fromN, toN := fromIndex, toIndex

	if len(content) < toIndex {
		toN = len(content)
	}

	content = content[fromN:toN]
	if strings.Join(content, web.Conf.Separator) != web.RawContent {
		web.RawContent = strings.Join(content, web.Conf.Separator)
		isUpdated = true
	}

	return isUpdated
}

func (serv *VendorService) Support(web *model.Website) bool {
	return web.Host() == host
}

func (serv *VendorService) Update(ctx context.Context, web *model.Website) error {
	body, fetchErr := serv.fetchWebsite(ctx, web)
	if fetchErr != nil {
		return fetchErr
	}

	if serv.isUpdated(ctx, web, body) {
		web.UpdateTime = time.Now().UTC().Truncate(time.Second)

		repoErr := serv.repo.UpdateWebsite(web)
		if repoErr != nil {
			return repoErr
		}
	}

	return nil
}
