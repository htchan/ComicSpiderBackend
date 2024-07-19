package website

import (
	"github.com/htchan/WebHistory/internal/model"
)

type errResp struct {
	Error string `json:"error"`
}

type listAllWebsiteGroupsResp struct {
	WebsiteGroups model.WebsiteGroups `json:"website_groups"`
}

type getWebsiteGroupResp struct {
	WebsiteGroup model.WebsiteGroup `json:"website_group"`
}

type createWebsiteResp struct {
	Msg string `json:"message"`
}

type getUserWebsiteResp struct {
	Website model.UserWebsite `json:"website"`
}

type refreshWebsiteResp struct {
	Website model.UserWebsite `json:"website"`
}

type deleteWebsiteResp struct {
	Msg string `json:"message"`
}

type changeWebsiteGroupResp struct {
	Website model.UserWebsite `json:"website"`
}
