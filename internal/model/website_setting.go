package model

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type WebsiteSetting struct {
	Domain               string
	TitleGoquerySelector string
	DatesGoquerySelector string
	FocusIndexFrom       int
	FocusIndexTo         int
}

func (setting *WebsiteSetting) Parse(response string) (string, []string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(response))
	if err != nil {
		fmt.Println("fail to parse error", err)

		return "", nil
	}

	title := doc.Find(setting.TitleGoquerySelector).Text()

	var dates []string
	doc.Find(setting.DatesGoquerySelector).Each(func(i int, s *goquery.Selection) {
		dates = append(dates, strings.TrimSpace(s.Text()))
	})

	fromN, toN := setting.FocusIndexFrom, setting.FocusIndexTo
	if fromN < 0 {
		fromN = len(dates) + fromN
		if fromN < 0 {
			fromN = 0
		}
	} else if fromN > len(dates) {
		fromN = len(dates) - 1
	}

	if toN <= 0 {
		toN = len(dates) + toN
		if toN < 0 {
			toN = len(dates)
		}
	} else if toN > len(dates) {
		toN = len(dates)
	}
	if fromN <= toN {
		dates = dates[fromN:toN]
	}

	return title, dates
}
