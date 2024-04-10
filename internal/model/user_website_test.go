package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
			expectedAccessTime: time.Now().UTC().Truncate(time.Second),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			userWeb := NewUserWebsite(test.web, test.userUUID)
			if userWeb.WebsiteUUID != test.web.UUID {
				t.Errorf("empty uuid")
			}
			if userWeb.UserUUID != test.userUUID {
				t.Errorf("wrong url: %s; want: %s", userWeb.UserUUID, test.userUUID)
			}
			if userWeb.GroupName != test.expectedGroupName {
				t.Errorf("wrong title: %s; want: %s", userWeb.GroupName, test.expectedGroupName)
			}
			if userWeb.AccessTime.Second() != test.expectedAccessTime.Second() {
				t.Errorf("wrong UpdateTime: %s; want: %s", userWeb.AccessTime, test.expectedAccessTime)
			}
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
			expect: `{"uuid":"","user_uuid":"user uuid","url":"http://example.com","title":"title","group_name":"group","update_time":"2020-01-02T00:00:00 UTC","access_time":"2020-01-02T00:00:00 UTC"}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result, err := json.Marshal(test.web)
			if err != nil {
				t.Errorf("got error: %v", err)
				return
			}
			if !cmp.Equal(string(result), test.expect) {
				t.Errorf("got unexpected json")
				t.Error(string(result))
				t.Error(test.expect)
			}
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
			if !cmp.Equal(groups, test.expectGroups) {
				t.Errorf("got wrong group")
				t.Error(groups)
				t.Error(test.expectGroups)
			}
		})
	}
}
