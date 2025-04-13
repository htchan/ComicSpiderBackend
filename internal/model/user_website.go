package model

import (
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
