package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/htchan/WebHistory/internal/model"
)

type InMemRepo struct {
	webs        []model.Website
	userWebs    []model.UserWebsite
	webSettings []model.WebsiteSetting
	err         error
}

var _ Repostory = &InMemRepo{}

func NewInMemRepo(webs []model.Website, userWebs []model.UserWebsite, webSettings []model.WebsiteSetting, err error) *InMemRepo {
	return &InMemRepo{
		webs:        webs,
		userWebs:    userWebs,
		webSettings: webSettings,
		err:         err,
	}
}

func (r *InMemRepo) CreateWebsite(web *model.Website) error {
	if r.err != nil {
		return r.err
	}
	for _, w := range r.webs {
		if w.URL == web.URL {
			web = &w
			return r.err
		}
	}
	r.webs = append(r.webs, *web)
	return r.err
}

func (r *InMemRepo) UpdateWebsite(web *model.Website) error {
	if r.err != nil {
		return r.err
	}
	for i, w := range r.webs {
		if w.UUID == web.UUID {
			r.webs[i] = *web
			break
		}
	}
	return r.err
}
func (r *InMemRepo) DeleteWebsite(web *model.Website) error {
	if r.err != nil {
		return r.err
	}
	var result []model.Website
	for _, w := range r.webs {
		if w.UUID == web.UUID {
			continue
		}
		result = append(result, w)
	}
	r.webs = result
	return r.err
}

func (r *InMemRepo) FindWebsites() ([]model.Website, error) {
	return r.webs, r.err
}
func (r *InMemRepo) FindWebsite(uuid string) (*model.Website, error) {
	for _, web := range r.webs {
		if web.UUID == uuid {
			return &web, r.err
		}
	}
	return nil, fmt.Errorf("website not found")
}

func (r *InMemRepo) CreateUserWebsite(web *model.UserWebsite) error {
	if r.err != nil {
		return r.err
	}
	for _, w := range r.userWebs {
		if w.UserUUID == web.UserUUID && w.WebsiteUUID == web.WebsiteUUID {
			web = &w
			return r.err
		}
	}
	r.userWebs = append(r.userWebs, *web)
	return r.err
}

func (r *InMemRepo) UpdateUserWebsite(web *model.UserWebsite) error {
	if r.err != nil {
		return r.err
	}
	for i, w := range r.userWebs {
		if w.UserUUID == web.UserUUID && w.WebsiteUUID == web.WebsiteUUID {
			r.userWebs[i] = *web
			return r.err
		}
	}
	return r.err
}
func (r *InMemRepo) DeleteUserWebsite(web *model.UserWebsite) error {
	if r.err != nil {
		return r.err
	}
	var result model.UserWebsites
	for _, w := range r.userWebs {
		if w.UserUUID == web.UserUUID && w.WebsiteUUID == web.WebsiteUUID {
			continue
		}
		result = append(result, w)
	}
	r.userWebs = result
	return r.err
}

func (r *InMemRepo) FindUserWebsites(userUUID string) (model.UserWebsites, error) {
	return r.userWebs, r.err
}
func (r *InMemRepo) FindUserWebsitesByGroup(userUUID, group string) (model.WebsiteGroup, error) {
	var webs model.WebsiteGroup
	for _, web := range r.userWebs {
		if web.UserUUID == userUUID && web.GroupName == group {
			webs = append(webs, web)
		}
	}
	return webs, r.err
}
func (r *InMemRepo) FindUserWebsite(userUUID, websiteUUID string) (*model.UserWebsite, error) {
	for _, web := range r.userWebs {
		if web.UserUUID == userUUID && web.WebsiteUUID == websiteUUID {
			return &web, r.err
		}
	}
	return nil, fmt.Errorf("user website not found")
}

func (r *InMemRepo) FindWebsiteSettings() ([]model.WebsiteSetting, error) {
	return r.webSettings, nil
}

func (r *InMemRepo) FindWebsiteSetting(domain string) (*model.WebsiteSetting, error) {
	for _, setting := range r.webSettings {
		if setting.Domain == domain {
			return &setting, nil
		}
	}
	return nil, fmt.Errorf("setting not found")
}

func (r InMemRepo) Equal(compare InMemRepo) bool {
	return cmp.Equal(r.webs, compare.webs) &&
		cmp.Equal(r.userWebs, compare.userWebs)
}

func (r InMemRepo) Stats() sql.DBStats {
	return sql.DBStats{}
}
