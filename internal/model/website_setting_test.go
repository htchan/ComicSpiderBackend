package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebsiteSetting_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setting     *WebsiteSetting
		resp        string
		expectTitle string
		expectDates []string
	}{
		{
			name: "happy flow",
			setting: &WebsiteSetting{
				TitleGoquerySelector: "head>title",
				DatesGoquerySelector: "ul>li",
			},
			resp:        "<html><head><title>test</title></head><body><ul><li>1</li><li>2</li></ul></body></html>",
			expectTitle: "test",
			expectDates: []string{"1", "2"},
		},
		{
			name: "happy flow that remove leading and trilling space",
			setting: &WebsiteSetting{
				TitleGoquerySelector: "head>title",
				DatesGoquerySelector: "ul>li",
			},
			resp:        "<html><head><title>test</title></head><body><ul><li> \n \t 1\r \n \t </li><li>2 3</li></ul></body></html>",
			expectTitle: "test",
			expectDates: []string{"1", "2 3"},
		},
		{
			name: "fail parse resp to doc",
			setting: &WebsiteSetting{
				TitleGoquerySelector: "head>title",
				DatesGoquerySelector: "ul>li",
			},
			resp:        "some unknown resp",
			expectTitle: "",
			expectDates: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			title, dates := test.setting.Parse(test.resp)
			assert.Equal(t, test.expectTitle, title)
			assert.Equal(t, test.expectDates, dates)
		})
	}
}
