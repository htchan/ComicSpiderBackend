package sqlc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/sqlc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const MinTimeUnit = 5 * time.Second

type SqlcRepo struct {
	db    *sqlc.Queries
	stats func() sql.DBStats
	conf  *config.WebsiteConfig
}

var _ repository.Repostory = &SqlcRepo{}

func NewRepo(db *sql.DB, conf *config.WebsiteConfig) *SqlcRepo {
	return &SqlcRepo{
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
		UpdateTime: webModel.UpdateTime.Time.UTC().Truncate(MinTimeUnit),
		Status:     webModel.Status,
	}
}

func fromSqlcListUserWebsitesRow(userWebModel sqlc.ListUserWebsitesRow) model.UserWebsite {
	return model.UserWebsite{
		WebsiteUUID: userWebModel.WebsiteUuid.String,
		UserUUID:    userWebModel.UserUuid.String,
		GroupName:   userWebModel.GroupName.String,
		AccessTime:  userWebModel.AccessTime.Time.UTC().Truncate(MinTimeUnit),
		Website: model.Website{
			UUID:       userWebModel.WebsiteUuid.String,
			URL:        userWebModel.Url.String,
			Title:      userWebModel.Title.String,
			UpdateTime: userWebModel.UpdateTime.Time.UTC().Truncate(MinTimeUnit),
		},
	}
}

func fromSqlcListUserWebsitesByGroupRow(userWebModel sqlc.ListUserWebsitesByGroupRow) model.UserWebsite {
	return model.UserWebsite{
		WebsiteUUID: userWebModel.WebsiteUuid.String,
		UserUUID:    userWebModel.UserUuid.String,
		GroupName:   userWebModel.GroupName.String,
		AccessTime:  userWebModel.AccessTime.Time.UTC().Truncate(MinTimeUnit),
		Website: model.Website{
			UUID:       userWebModel.WebsiteUuid.String,
			URL:        userWebModel.Url.String,
			Title:      userWebModel.Title.String,
			UpdateTime: userWebModel.UpdateTime.Time.UTC().Truncate(MinTimeUnit),
		},
	}
}

func fromSqlcGetUserWebsiteRow(userWebModel sqlc.GetUserWebsiteRow) model.UserWebsite {
	return model.UserWebsite{
		WebsiteUUID: userWebModel.WebsiteUuid.String,
		UserUUID:    userWebModel.UserUuid.String,
		GroupName:   userWebModel.GroupName.String,
		AccessTime:  userWebModel.AccessTime.Time.UTC().Truncate(MinTimeUnit),
		Website: model.Website{
			UUID:       userWebModel.WebsiteUuid.String,
			URL:        userWebModel.Url.String,
			Title:      userWebModel.Title.String,
			UpdateTime: userWebModel.UpdateTime.Time.UTC().Truncate(MinTimeUnit),
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

func (r *SqlcRepo) CreateWebsite(ctx context.Context, web *model.Website) error {
	_, createWebsiteSpan := repository.GetTracer().Start(ctx, "create website")
	defer createWebsiteSpan.End()

	params := toSqlcCreateWebsiteParams(web)
	jsonByte, jsonErr := json.Marshal(params)
	if jsonErr == nil {
		createWebsiteSpan.SetAttributes(attribute.String("params", string(jsonByte)))
	}

	// return web if url exist
	webModel, err := r.db.CreateWebsite(ctx, params)
	if err != nil {
		if err != sql.ErrNoRows {
			createWebsiteSpan.SetStatus(codes.Error, err.Error())
			createWebsiteSpan.RecordError(err)
		}

		return err
	}

	web.UUID = webModel.Uuid.String
	web.Title, web.RawContent = webModel.Title.String, webModel.Content.String
	web.UpdateTime = webModel.UpdateTime.Time.UTC().Truncate(MinTimeUnit)
	web.Conf = r.conf

	return nil
}

func (r *SqlcRepo) UpdateWebsite(ctx context.Context, web *model.Website) error {
	_, updateWebsiteSpan := repository.GetTracer().Start(ctx, "update website")
	defer updateWebsiteSpan.End()

	params := toSqlcUpdateWebsiteParams(web)
	jsonByte, jsonErr := json.Marshal(params)
	if jsonErr == nil {
		updateWebsiteSpan.SetAttributes(attribute.String("params", string(jsonByte)))
	}

	_, err := r.db.UpdateWebsite(ctx, params)
	if err != nil {
		updateWebsiteSpan.SetStatus(codes.Error, err.Error())
		updateWebsiteSpan.RecordError(err)

		return fmt.Errorf("update website fail: %w", err)
	}

	return nil
}

func (r *SqlcRepo) DeleteWebsite(ctx context.Context, web *model.Website) error {
	_, deleteWebsiteSpan := repository.GetTracer().Start(ctx, "delete website")
	defer deleteWebsiteSpan.End()

	deleteWebsiteSpan.SetAttributes(attribute.String("params.website_uuid", web.UUID))

	err := r.db.DeleteWebsite(ctx, toSqlString(web.UUID))
	if err != nil {
		deleteWebsiteSpan.SetStatus(codes.Error, err.Error())
		deleteWebsiteSpan.RecordError(err)

		return fmt.Errorf("fail to delete website: %w", err)
	}

	return nil
}

func (r *SqlcRepo) FindWebsites(ctx context.Context) ([]model.Website, error) {
	_, listWebsitesSpan := repository.GetTracer().Start(ctx, "find websites")
	defer listWebsitesSpan.End()

	webModels, err := r.db.ListActiveWebsites(ctx)
	if err != nil {
		listWebsitesSpan.SetStatus(codes.Error, err.Error())
		listWebsitesSpan.RecordError(err)

		return nil, fmt.Errorf("list websites fail: %w", err)
	}

	webs := make([]model.Website, len(webModels))
	for i, webModel := range webModels {
		webs[i] = fromSqlcWebsite(webModel)
		webs[i].Conf = r.conf
	}

	return webs, nil
}

func (r *SqlcRepo) FindWebsite(ctx context.Context, uuid string) (*model.Website, error) {
	_, findWebsiteSpan := repository.GetTracer().Start(ctx, "find website")
	defer findWebsiteSpan.End()

	findWebsiteSpan.SetAttributes(attribute.String("params.website_uuid", uuid))

	webModel, err := r.db.GetWebsite(ctx, toSqlString(uuid))
	if err != nil {
		return nil, fmt.Errorf("get website fail: %w", err)
	}

	web := fromSqlcWebsite(webModel)
	web.Conf = r.conf

	return &web, nil
}

func (r *SqlcRepo) CreateUserWebsite(ctx context.Context, web *model.UserWebsite) error {
	_, createUserWebsiteSpan := repository.GetTracer().Start(ctx, "create user website")
	defer createUserWebsiteSpan.End()

	params := toSqlcCreateUserWebsiteParams(web)

	jsonByte, jsonErr := json.Marshal(params)
	if jsonErr == nil {
		createUserWebsiteSpan.SetAttributes(attribute.String("params", string(jsonByte)))
	}

	userWebModel, err := r.db.CreateUserWebsite(ctx, params)
	if err != nil {
		createUserWebsiteSpan.SetStatus(codes.Error, fmt.Errorf("create user website fail:%w", err).Error())
		createUserWebsiteSpan.RecordError(err)

		return fmt.Errorf("create user website fail: %w", err)
	}

	web.GroupName = userWebModel.GroupName.String
	web.AccessTime = userWebModel.AccessTime.Time.UTC().Truncate(MinTimeUnit)
	tempWeb, err := r.FindWebsite(ctx, web.WebsiteUUID)
	if err != nil {
		createUserWebsiteSpan.SetStatus(codes.Error, fmt.Errorf("assign website fail:%w", err).Error())
		createUserWebsiteSpan.RecordError(err)

		return fmt.Errorf("assign website fail: %w", err)
	}

	web.Website = *tempWeb

	return nil
}

func (r *SqlcRepo) UpdateUserWebsite(ctx context.Context, web *model.UserWebsite) error {
	_, updateUserWebsiteSpan := repository.GetTracer().Start(ctx, "update user website")
	defer updateUserWebsiteSpan.End()

	params := toSqlcUpdateUserWebsiteParams(web)
	jsonByte, jsonErr := json.Marshal(params)
	if jsonErr == nil {
		updateUserWebsiteSpan.SetAttributes(attribute.String("params", string(jsonByte)))
	}

	_, err := r.db.UpdateUserWebsite(ctx, params)
	if err != nil {
		updateUserWebsiteSpan.SetStatus(codes.Error, err.Error())
		updateUserWebsiteSpan.RecordError(err)

		return fmt.Errorf("fail to update user website: %w", err)
	}

	return nil
}

func (r *SqlcRepo) DeleteUserWebsite(ctx context.Context, web *model.UserWebsite) error {
	_, deleteUserWebsiteSpan := repository.GetTracer().Start(ctx, "delete user website")
	defer deleteUserWebsiteSpan.End()

	params := toSqlcDeleteUserWebsiteParams(web)
	jsonByte, jsonErr := json.Marshal(params)
	if jsonErr == nil {
		deleteUserWebsiteSpan.SetAttributes(attribute.String("params", string(jsonByte)))
	}

	err := r.db.DeleteUserWebsite(ctx, params)
	if err != nil {
		deleteUserWebsiteSpan.SetStatus(codes.Error, err.Error())
		deleteUserWebsiteSpan.RecordError(err)

		return fmt.Errorf("delete user website fail: %w", err)
	}

	return nil
}

func (r *SqlcRepo) FindUserWebsites(ctx context.Context, userUUID string) (model.UserWebsites, error) {
	_, listUserWebsitesSpan := repository.GetTracer().Start(ctx, "find user websites")
	defer listUserWebsitesSpan.End()

	listUserWebsitesSpan.SetAttributes(attribute.String("params.user_uuid", userUUID))

	userWebModels, err := r.db.ListUserWebsites(ctx, toSqlString(userUUID))
	if err != nil {
		listUserWebsitesSpan.SetStatus(codes.Error, err.Error())
		listUserWebsitesSpan.RecordError(err)

		return nil, fmt.Errorf("list user websites fail: %w", err)
	}

	webs := make(model.UserWebsites, len(userWebModels))
	for i, userWebModel := range userWebModels {
		webs[i] = fromSqlcListUserWebsitesRow(userWebModel)
		webs[i].Website.Conf = r.conf
	}

	return webs, nil
}

func (r *SqlcRepo) FindUserWebsitesByGroup(ctx context.Context, userUUID, groupName string) (model.WebsiteGroup, error) {
	_, listUserWebsitesByGroupSpan := repository.GetTracer().Start(ctx, "find user websites by group")
	defer listUserWebsitesByGroupSpan.End()

	params := toSqlcListUserWebsitesByGroupParams(userUUID, groupName)
	jsonByte, jsonErr := json.Marshal(params)
	if jsonErr == nil {
		listUserWebsitesByGroupSpan.SetAttributes(attribute.String("params", string(jsonByte)))
	}

	userWebModels, err := r.db.ListUserWebsitesByGroup(ctx, params)
	if err != nil {
		listUserWebsitesByGroupSpan.SetStatus(codes.Error, err.Error())
		listUserWebsitesByGroupSpan.RecordError(err)

		return nil, fmt.Errorf("find user websites by group fail: %w", err)
	}

	group := make(model.WebsiteGroup, len(userWebModels))
	for i, userWebModel := range userWebModels {
		group[i] = fromSqlcListUserWebsitesByGroupRow(userWebModel)
		group[i].Website.Conf = r.conf
	}

	return group, nil
}

func (r *SqlcRepo) FindUserWebsite(ctx context.Context, userUUID, websiteUUID string) (*model.UserWebsite, error) {
	_, findUserWebsiteSpan := repository.GetTracer().Start(ctx, "find user website")
	defer findUserWebsiteSpan.End()

	findUserWebsiteSpan.SetAttributes(attribute.String("params.user_uuid", userUUID), attribute.String("params.website_uuid", websiteUUID))

	userWebModel, err := r.db.GetUserWebsite(ctx, toSqlcGetUserWebsitesParams(userUUID, websiteUUID))
	if err != nil {
		findUserWebsiteSpan.SetStatus(codes.Error, err.Error())
		findUserWebsiteSpan.RecordError(err)

		return nil, fmt.Errorf("get user website fail: %w", err)
	}

	web := fromSqlcGetUserWebsiteRow(userWebModel)
	web.Website.Conf = r.conf

	return &web, nil
}

func (r *SqlcRepo) Stats() sql.DBStats {
	return r.stats()
}
