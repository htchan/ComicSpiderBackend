package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	MaxRetryCount = 10
	RetryInterval = 10 * time.Second
)

type HTTPClient interface {
	Get(string) (*http.Response, error)
}

var client HTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		DisableKeepAlives: true,
	},
}

func pruneResponse(resp *http.Response, conf *config.WebsiteConfig) string {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile("[\r|\n|\t]")
	bodyStr := re.ReplaceAllString(string(body), "")
	re = regexp.MustCompile("(<script.*?/script>|<style.*?/style>|<path.*?/path>)")
	bodyStr = re.ReplaceAllString(
		bodyStr,
		"<delete/>",
	)
	re = regexp.MustCompile("<(/?title.*?)>")
	bodyStr = re.ReplaceAllString(bodyStr, "[$1]")
	re = regexp.MustCompile("(<.*?>)+")
	bodyStr = re.ReplaceAllString(bodyStr, conf.Separator)
	re = regexp.MustCompile(`\[(/?title.*?)\]`)
	bodyStr = re.ReplaceAllString(bodyStr, "<$1>")
	bodyStr = strings.Trim(bodyStr, conf.Separator)
	return bodyStr
}

func getWebsiteSetting(r repository.Repostory, web *model.Website) (*model.WebsiteSetting, error) {
	u, err := url.Parse(web.URL)
	if err != nil {
		return nil, fmt.Errorf("fail to parse url: %s", web.URL)
	}

	setting, err := r.FindWebsiteSetting(u.Hostname())
	if err == nil {
		return setting, nil
	}

	return r.FindWebsiteSetting("default")
}

func parseAPI(r repository.Repostory, web *model.Website, resp string) (string, []string) {
	setting, err := getWebsiteSetting(r, web)
	if err != nil {
		return "", nil
	}
	return setting.Parse(resp)
}

func fetchWebsite(ctx context.Context, web *model.Website, maxRetry int, retryInterval time.Duration) (string, error) {
	tr := otel.Tracer("htchan/WebHistory/update-jobs")
	_, span := tr.Start(ctx, "Fetch Web")
	defer span.End()

	var (
		resp *http.Response
		err  error
	)
	for i := 0; i < maxRetry; i++ {
		resp, err = client.Get(web.URL)
		if err != nil {
			zerolog.Ctx(ctx).Warn().
				Err(err).
				Int("trial", i).
				Str("url", web.URL).
				Msg("fail to fetch website")
			time.Sleep(retryInterval)
		} else {
			break
		}
	}
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		if web.Title == "" {
			web.Title = "unknown"
		}

		return "", fmt.Errorf("fail to fetch website response: %s", web.URL)
	}

	// body := pruneResponse(resp, web.Conf)
	data, _ := io.ReadAll(resp.Body)
	body := string(data)
	span.SetAttributes(attribute.String("raw response", body))
	return body, nil
}

func checkTimeUpdated(ctx context.Context, web *model.Website, timeStr string) bool {
	tr := otel.Tracer("htchan/WebHistory/update-jobs")
	_, span := tr.Start(ctx, "Check Time")
	defer span.End()

	layout := "Mon, 2 Jan 2006 15:04:05 GMT"
	span.SetAttributes(
		attribute.String("old time", web.UpdateTime.Format(layout)),
		attribute.String("new time", timeStr),
	)

	if timeStr == "" {
		return false
	}
	t, err := time.Parse(layout, timeStr)
	if err == nil && t.After(web.UpdateTime) {
		return true
	}
	return false
}

func checkContentUpdated(ctx context.Context, web *model.Website, content []string) bool {
	tr := otel.Tracer("htchan/WebHistory/update-jobs")
	_, span := tr.Start(ctx, "Check Content")
	defer span.End()
	span.SetAttributes(
		attribute.StringSlice("old content", web.Content()),
		attribute.StringSlice("new content", content),
	)

	if len(content) > 0 && !cmp.Equal(web.Content(), content) {
		web.RawContent = strings.Join(content, web.Conf.Separator)
		web.UpdateTime = time.Now().UTC().Truncate(time.Second)
		return true
	}
	return false
}

func checkTitleUpdated(ctx context.Context, web *model.Website, title string) bool {
	tr := otel.Tracer("htchan/WebHistory/update-jobs")
	_, span := tr.Start(ctx, "Check Title")
	defer span.End()
	span.SetAttributes(
		attribute.String("old title", web.Title),
		attribute.String("new title", title),
	)

	if web.Title != title {
		if web.Title == "" || web.Title == "unknown" {
			web.Title = title
			web.UpdateTime = time.Now().UTC().Truncate(time.Second)
			return true
		}
	}
	return false
}

func checkWeb(ctx context.Context, r repository.Repostory, web *model.Website, title string, content []string) {
	tr := otel.Tracer("htchan/WebHistory/update-jobs")
	ctx, span := tr.Start(ctx, "Checking")
	defer span.End()

	titleUpdated := checkTitleUpdated(ctx, web, title)
	contentUpadted := checkContentUpdated(ctx, web, content)
	span.SetAttributes(
		attribute.Bool("title updated", titleUpdated),
		attribute.Bool("content updated", contentUpadted),
	)

	if titleUpdated || contentUpadted {
		_, span = tr.Start(ctx, "Updated")
		defer span.End()

		err := r.UpdateWebsite(web)
		if err != nil {
			span.SetAttributes(attribute.String("error", err.Error()))
		}
	}
}

func Update(ctx context.Context, r repository.Repostory, web *model.Website) error {
	content, err := fetchWebsite(ctx, web, MaxRetryCount, RetryInterval)
	if err != nil {
		return err
	}

	title, dates := parseAPI(r, web, content)
	checkWeb(ctx, r, web, title, dates)

	return nil
}
