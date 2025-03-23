package model

import (
	"encoding/json"
	"time"
)

type UserWebsite struct {
	WebsiteUUID string
	UserUUID    string
	GroupName   string
	AccessTime  time.Time
	Website     Website
}

type UserWebsites []UserWebsite

func NewUserWebsite(web Website, userUUID string) UserWebsite {
	return UserWebsite{
		WebsiteUUID: web.UUID,
		UserUUID:    userUUID,
		GroupName:   web.Title,
		AccessTime:  time.Now().UTC().Truncate(time.Second),
		Website:     web,
	}
}

func (webs UserWebsites) WebsiteGroups() WebsiteGroups {
	indexMap := make(map[string]int)
	var groups WebsiteGroups
	for _, web := range webs {
		index, ok := indexMap[web.GroupName]
		if !ok {
			index = len(groups)
			groups = append(groups, WebsiteGroup{web})
			indexMap[web.GroupName] = index
		} else {
			groups[index] = append(groups[index], web)
		}
	}

	return groups
}

func (web UserWebsite) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		UUID       string `json:"uuid"`
		UserUUID   string `json:"user_uuid"`
		URL        string `json:"url"`
		Title      string `json:"title"`
		GroupName  string `json:"group_name"`
		UpdateTime string `json:"update_time"`
		AccessTime string `json:"access_time"`
	}{
		UUID:       web.WebsiteUUID,
		UserUUID:   web.UserUUID,
		URL:        web.Website.URL,
		Title:      web.Website.Title,
		GroupName:  web.GroupName,
		UpdateTime: web.Website.UpdateTime.Format("2006-01-02T15:04:05Z07:00"),
		AccessTime: web.AccessTime.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (web UserWebsite) Equal(compare UserWebsite) bool {
	return web.UserUUID == compare.UserUUID &&
		web.WebsiteUUID == compare.WebsiteUUID &&
		web.GroupName == compare.GroupName &&
		web.AccessTime.Unix()/1000 == compare.AccessTime.Unix()/1000 &&
		web.Website.UUID == compare.Website.UUID &&
		web.Website.URL == compare.Website.URL &&
		web.Website.Title == compare.Website.Title &&
		web.Website.UpdateTime.Unix()/1000 == compare.Website.UpdateTime.Unix()/1000
}
