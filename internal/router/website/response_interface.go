package website

import (
	"time"

	"github.com/htchan/WebHistory/internal/model"
)

type errResp struct {
	Error string `json:"error"`
}
type WebsiteResp struct {
	UUID       string    `json:"uuid"`
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	UpdateTime time.Time `json:"update_time"`
}

type UserWebsiteResp struct {
	UUID       string    `json:"uuid"`
	UserUUID   string    `json:"user_uuid"`
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	GroupName  string    `json:"group_name"`
	UpdateTime time.Time `json:"update_time"`
	AccessTime time.Time `json:"access_time"`
}

type WebsiteGroupResp []UserWebsiteResp
type WebsiteGroupsResp []WebsiteGroupResp

func fromModelWebsite(web model.Website) WebsiteResp {
	return WebsiteResp{
		UUID:       web.UUID,
		URL:        web.URL,
		Title:      web.Title,
		UpdateTime: web.UpdateTime,
	}
}

func fromModelUserWebsite(web model.UserWebsite) UserWebsiteResp {
	return UserWebsiteResp{
		UUID:       web.WebsiteUUID,
		UserUUID:   web.UserUUID,
		URL:        web.Website.URL,
		Title:      web.Website.Title,
		GroupName:  web.GroupName,
		UpdateTime: web.Website.UpdateTime,
		AccessTime: web.AccessTime,
	}
}

func fromModelWebsiteGroup(group model.WebsiteGroup) WebsiteGroupResp {
	webs := WebsiteGroupResp{}
	for _, web := range group {
		webs = append(webs, fromModelUserWebsite(web))
	}

	return webs
}

func fromModelWebsiteGroups(groups model.WebsiteGroups) WebsiteGroupsResp {
	webs := WebsiteGroupsResp{}
	for _, group := range groups {
		webs = append(webs, fromModelWebsiteGroup(group))
	}

	return webs
}

type listAllWebsiteGroupsResp struct {
	WebsiteGroups WebsiteGroupsResp `json:"website_groups"`
}

type getWebsiteGroupResp struct {
	WebsiteGroup WebsiteGroupResp `json:"website_group"`
}

type createWebsiteResp struct {
	Msg string `json:"message"`
}

type getUserWebsiteResp struct {
	Website UserWebsiteResp `json:"website"`
}

type refreshWebsiteResp struct {
	Website UserWebsiteResp `json:"website"`
}

type deleteWebsiteResp struct {
	Msg string `json:"message"`
}

type changeWebsiteGroupResp struct {
	Website UserWebsiteResp `json:"website"`
}
