package sqlc

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/sqlc"
)

type SqlcRepo struct {
	ctx   context.Context
	db    *sqlc.Queries
	stats func() sql.DBStats
	conf  *config.WebsiteConfig
}

var _ repository.Repostory = &SqlcRepo{}

func NewRepo(db *sql.DB, conf *config.WebsiteConfig) *SqlcRepo {
	return &SqlcRepo{
		ctx:   context.Background(),
		db:    sqlc.New(db),
		stats: db.Stats,
		conf:  conf,
	}
}

func toSqlString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func toSqlTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}

func fromSqlcWebsite(webModel sqlc.Website) model.Website {
	return model.Website{
		UUID:       webModel.Uuid.String,
		URL:        webModel.Url.String,
		Title:      webModel.Title.String,
		RawContent: webModel.Content.String,
		UpdateTime: webModel.UpdateTime.Time.UTC().Truncate(time.Second),
	}
}

func fromSqlcListUserWebsitesRow(userWebModel sqlc.ListUserWebsitesRow) model.UserWebsite {
	return model.UserWebsite{
		WebsiteUUID: userWebModel.WebsiteUuid.String,
		UserUUID:    userWebModel.UserUuid.String,
		GroupName:   userWebModel.GroupName.String,
		AccessTime:  userWebModel.AccessTime.Time.UTC().Truncate(time.Second),
		Website: model.Website{
			UUID:       userWebModel.WebsiteUuid.String,
			URL:        userWebModel.Url.String,
			Title:      userWebModel.Title.String,
			UpdateTime: userWebModel.UpdateTime.Time.UTC().Truncate(time.Second),
		},
	}
}

func fromSqlcListUserWebsitesByGroupRow(userWebModel sqlc.ListUserWebsitesByGroupRow) model.UserWebsite {
	return model.UserWebsite{
		WebsiteUUID: userWebModel.WebsiteUuid.String,
		UserUUID:    userWebModel.UserUuid.String,
		GroupName:   userWebModel.GroupName.String,
		AccessTime:  userWebModel.AccessTime.Time.UTC().Truncate(time.Second),
		Website: model.Website{
			UUID:       userWebModel.WebsiteUuid.String,
			URL:        userWebModel.Url.String,
			Title:      userWebModel.Title.String,
			UpdateTime: userWebModel.UpdateTime.Time.UTC().Truncate(time.Second),
		},
	}
}

func fromSqlcGetUserWebsiteRow(userWebModel sqlc.GetUserWebsiteRow) model.UserWebsite {
	return model.UserWebsite{
		WebsiteUUID: userWebModel.WebsiteUuid.String,
		UserUUID:    userWebModel.UserUuid.String,
		GroupName:   userWebModel.GroupName.String,
		AccessTime:  userWebModel.AccessTime.Time.UTC().Truncate(time.Second),
		Website: model.Website{
			UUID:       userWebModel.WebsiteUuid.String,
			URL:        userWebModel.Url.String,
			Title:      userWebModel.Title.String,
			UpdateTime: userWebModel.UpdateTime.Time.UTC().Truncate(time.Second),
		},
	}
}

func toSqlcListUserWebsitesByGroupParams(userUUID, groupName string) sqlc.ListUserWebsitesByGroupParams {
	return sqlc.ListUserWebsitesByGroupParams{
		UserUuid:  toSqlString(userUUID),
		GroupName: toSqlString(groupName),
	}
}

func toSqlcGetUserWebsitesParams(userUUID, websiteUUID string) sqlc.GetUserWebsiteParams {
	return sqlc.GetUserWebsiteParams{
		UserUuid:    toSqlString(userUUID),
		WebsiteUuid: toSqlString(websiteUUID),
	}
}

func toSqlcCreateWebsiteParams(web *model.Website) sqlc.CreateWebsiteParams {
	return sqlc.CreateWebsiteParams{
		Uuid:       toSqlString(web.UUID),
		Url:        toSqlString(web.URL),
		Title:      toSqlString(web.Title),
		Content:    toSqlString(web.RawContent),
		UpdateTime: toSqlTime(web.UpdateTime),
	}
}

func toSqlcCreateUserWebsiteParams(userWeb *model.UserWebsite) sqlc.CreateUserWebsiteParams {
	return sqlc.CreateUserWebsiteParams{
		UserUuid:    toSqlString(userWeb.UserUUID),
		WebsiteUuid: toSqlString(userWeb.WebsiteUUID),
		AccessTime:  toSqlTime(userWeb.AccessTime),
		GroupName:   toSqlString(userWeb.GroupName),
	}
}

func toSqlcUpdateWebsiteParams(web *model.Website) sqlc.UpdateWebsiteParams {
	return sqlc.UpdateWebsiteParams{
		Url:        toSqlString(web.URL),
		Title:      toSqlString(web.Title),
		Content:    toSqlString(web.RawContent),
		UpdateTime: toSqlTime(web.UpdateTime),
		Uuid:       toSqlString(web.UUID),
	}
}

func toSqlcUpdateUserWebsiteParams(userWeb *model.UserWebsite) sqlc.UpdateUserWebsiteParams {
	return sqlc.UpdateUserWebsiteParams{
		UserUuid:    toSqlString(userWeb.UserUUID),
		WebsiteUuid: toSqlString(userWeb.WebsiteUUID),
		AccessTime:  toSqlTime(userWeb.AccessTime),
		GroupName:   toSqlString(userWeb.GroupName),
	}
}

func toSqlcDeleteUserWebsiteParams(userWeb *model.UserWebsite) sqlc.DeleteUserWebsiteParams {
	return sqlc.DeleteUserWebsiteParams{
		UserUuid:    toSqlString(userWeb.UserUUID),
		WebsiteUuid: toSqlString(userWeb.WebsiteUUID),
	}
}

func (r *SqlcRepo) CreateWebsite(web *model.Website) error {
	// return web if url exist
	webModel, err := r.db.CreateWebsite(
		r.ctx,
		toSqlcCreateWebsiteParams(web),
	)
	if err != nil {
		return err
	}

	web.UUID = webModel.Uuid.String
	web.Title, web.RawContent = webModel.Title.String, webModel.Content.String
	web.UpdateTime = webModel.UpdateTime.Time.UTC().Truncate(time.Second)
	web.Conf = r.conf

	return nil
}

func (r *SqlcRepo) UpdateWebsite(web *model.Website) error {
	_, err := r.db.UpdateWebsite(
		r.ctx,
		toSqlcUpdateWebsiteParams(web),
	)
	if err != nil {
		return fmt.Errorf("update website fail: %w", err)
	}

	return nil
}

func (r *SqlcRepo) DeleteWebsite(web *model.Website) error {
	err := r.db.DeleteWebsite(
		r.ctx,
		toSqlString(web.UUID),
	)
	if err != nil {
		return fmt.Errorf("fail to delete website: %w", err)
	}

	return nil
}

func (r *SqlcRepo) FindWebsites() ([]model.Website, error) {
	webModels, err := r.db.ListWebsites(r.ctx)
	if err != nil {
		return nil, fmt.Errorf("list websites fail: %w", err)
	}

	webs := make([]model.Website, len(webModels))
	for i, webModel := range webModels {
		webs[i] = fromSqlcWebsite(webModel)
		webs[i].Conf = r.conf
	}

	return webs, nil
}

func (r *SqlcRepo) FindWebsite(uuid string) (*model.Website, error) {
	webModel, err := r.db.GetWebsite(r.ctx, sql.NullString{String: uuid, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("get website fail: %w", err)
	}

	web := fromSqlcWebsite(webModel)
	web.Conf = r.conf
	return &web, nil
}

func (r *SqlcRepo) CreateUserWebsite(web *model.UserWebsite) error {
	userWebModel, err := r.db.CreateUserWebsite(r.ctx, toSqlcCreateUserWebsiteParams(web))
	if err != nil {
		return fmt.Errorf("create user website fail: %w", err)
	}

	web.GroupName = userWebModel.GroupName.String
	web.AccessTime = userWebModel.AccessTime.Time.UTC().Truncate(time.Second)
	tempWeb, err := r.FindWebsite(web.WebsiteUUID)
	if err != nil {
		return fmt.Errorf("assign website fail: %w", err)
	}

	web.Website = *tempWeb

	return nil
}

func (r *SqlcRepo) UpdateUserWebsite(web *model.UserWebsite) error {
	_, err := r.db.UpdateUserWebsite(r.ctx, toSqlcUpdateUserWebsiteParams(web))
	if err != nil {
		return fmt.Errorf("fail to update user website: %w", err)
	}

	return nil
}

func (r *SqlcRepo) DeleteUserWebsite(web *model.UserWebsite) error {
	err := r.db.DeleteUserWebsite(r.ctx, toSqlcDeleteUserWebsiteParams(web))
	if err != nil {
		return fmt.Errorf("delete user website fail: %w", err)
	}

	return nil
}

func (r *SqlcRepo) FindUserWebsites(userUUID string) (model.UserWebsites, error) {
	userWebModels, err := r.db.ListUserWebsites(r.ctx, toSqlString(userUUID))
	if err != nil {
		return nil, fmt.Errorf("list user websites fail: %w", err)
	}

	webs := make(model.UserWebsites, len(userWebModels))
	for i, userWebModel := range userWebModels {
		webs[i] = fromSqlcListUserWebsitesRow(userWebModel)
		webs[i].Website.Conf = r.conf
	}

	return webs, nil
}

func (r *SqlcRepo) FindUserWebsitesByGroup(userUUID, groupName string) (model.WebsiteGroup, error) {
	userWebModels, err := r.db.ListUserWebsitesByGroup(r.ctx, toSqlcListUserWebsitesByGroupParams(userUUID, groupName))
	if err != nil {
		return nil, fmt.Errorf("find user websites by group fail: %w", err)
	}

	group := make(model.WebsiteGroup, len(userWebModels))
	for i, userWebModel := range userWebModels {
		group[i] = fromSqlcListUserWebsitesByGroupRow(userWebModel)
		group[i].Website.Conf = r.conf
	}

	return group, nil
}

func (r *SqlcRepo) FindUserWebsite(userUUID, websiteUUID string) (*model.UserWebsite, error) {
	userWebModel, err := r.db.GetUserWebsite(r.ctx, toSqlcGetUserWebsitesParams(userUUID, websiteUUID))
	if err != nil {
		return nil, fmt.Errorf("get user website fail: %w", err)
	}

	web := fromSqlcGetUserWebsiteRow(userWebModel)
	web.Website.Conf = r.conf

	return &web, nil
}

func (r *SqlcRepo) Stats() sql.DBStats {
	return r.stats()
}
