package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_NewUserWebsite(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		web                Website
		userUUID           string
		expectedGroupName  string
		expectedAccessTime time.Time
	}{
		{
			name:               "happy flow",
			web:                Website{UUID: "uuid"},
			userUUID:           "user uuid",
			expectedGroupName:  "",
			expectedAccessTime: time.Now().UTC().Truncate(5 * time.Second),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			userWeb := NewUserWebsite(test.web, test.userUUID)
			assert.Equal(t, test.web.UUID, userWeb.WebsiteUUID)
			assert.Equal(t, test.userUUID, userWeb.UserUUID)
			assert.Equal(t, test.expectedGroupName, userWeb.GroupName)
			assert.Equal(t, test.expectedAccessTime, userWeb.AccessTime)
		})
	}
}

func TestUserWebsite_MarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		web    UserWebsite
		expect string
	}{
		{
			name: "happy flow",
			web: UserWebsite{
				Website: Website{
					UUID:       "uuid",
					URL:        "http://example.com",
					Title:      "title",
					UpdateTime: time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				UserUUID:   "user uuid",
				GroupName:  "group",
				AccessTime: time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			expect: `{"WebsiteUUID":"","UserUUID":"user uuid","GroupName":"group","AccessTime":"2020-01-02T00:00:00Z","Website":{"uuid":"uuid","url":"http://example.com","title":"title","raw_content":"","update_time":"2020-01-02T00:00:00Z"}}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result, err := json.Marshal(test.web)
			assert.NoError(t, err, nil)
			assert.Equal(t, test.expect, string(result))
		})
	}
}

func TestUserWebsites_WebsiteGroups(t *testing.T) {
	tests := []struct {
		name         string
		webs         UserWebsites
		expectGroups WebsiteGroups
	}{
		{
			name: "happy flow",
			webs: UserWebsites{
				UserWebsite{WebsiteUUID: "1", GroupName: "1"},
				UserWebsite{WebsiteUUID: "2", GroupName: "1"},
				UserWebsite{WebsiteUUID: "3", GroupName: "2"},
			},
			expectGroups: WebsiteGroups{
				WebsiteGroup{
					UserWebsite{WebsiteUUID: "1", GroupName: "1"},
					UserWebsite{WebsiteUUID: "2", GroupName: "1"},
				},
				WebsiteGroup{
					UserWebsite{WebsiteUUID: "3", GroupName: "2"},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			groups := test.webs.WebsiteGroups()
			assert.Equal(t, test.expectGroups, groups)
		})
	}
}
