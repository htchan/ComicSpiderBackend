package website

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/go-chi/chi/v5"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	websiteupdate "github.com/htchan/WebHistory/internal/tasks/nats/website_update"
)

func encodeJsonResp(ctx context.Context, res http.ResponseWriter, body any) {
	tr := otel.Tracer("htchan/WebHistory/api")

	_, encodeSpan := tr.Start(ctx, "Encode response")
	defer encodeSpan.End()

	json.NewEncoder(res).Encode(body)
}

// @Summary		Get website group
// @description	get website group
// @Tags			web-history
// @Accept			json
// @Produce		json
// @Param			X-USER-UUID	header		string	true	"user uuid"
// @Success		200			{object}	listAllWebsiteGroupsResp
// @Failure		400			{object}	errResp
// @Router			/api/web-watcher/websites/groups [get]
func getAllWebsiteGroupsHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		tr := otel.Tracer("htchan/WebHistory/api")

		dbCtx, dbSpan := tr.Start(req.Context(), "User Website Query")
		defer dbSpan.End()

		userUUID := req.Context().Value(ContextKeyUserUUID).(string)
		webs, err := r.FindUserWebsites(userUUID)

		if err != nil {
			dbSpan.SetStatus(codes.Error, err.Error())
			dbSpan.RecordError(err)

			zerolog.Ctx(dbCtx).Error().Err(err).Msg("find user websites failed")
			writeError(res, http.StatusBadRequest, ErrRecordNotFound)
			return
		}

		dbSpan.End()

		encodeJsonResp(req.Context(), res, listAllWebsiteGroupsResp{fromModelWebsiteGroups(webs.WebsiteGroups())})
	}
}

// @Summary		Get website group
// @description	get website group
// @Tags			web-history
// @Accept			json
// @Produce		json
// @Param			X-USER-UUID	header		string	true	"user uuid"
// @Param			groupName	path		string	true	"group name"
// @Success		200			{object}	getWebsiteGroupResp
// @Failure		400			{object}	errResp
// @Router			/api/web-watcher/websites/groups/{groupName} [get]
func getWebsiteGroupHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		tr := otel.Tracer("htchan/WebHistory/api")

		userUUID := req.Context().Value(ContextKeyUserUUID).(string)
		groupName := chi.URLParam(req, "groupName")

		dbCtx, dbSpan := tr.Start(req.Context(), "User Website Group Query")
		defer dbSpan.End()

		webs, err := r.FindUserWebsitesByGroup(userUUID, groupName)
		if err != nil || len(webs) == 0 {
			dbSpan.SetStatus(codes.Error, err.Error())
			dbSpan.RecordError(err)

			zerolog.Ctx(dbCtx).Error().Err(err).Msg("find user websites by group failed")
			writeError(res, http.StatusBadRequest, ErrRecordNotFound)
			return
		}

		dbSpan.End()

		encodeJsonResp(req.Context(), res, getWebsiteGroupResp{fromModelWebsiteGroup(webs)})
	}
}

// @Summary		Create website
// @description	create website
// @Tags			web-history
// @Accept			json
// @Produce		json
// @Param			X-USER-UUID	header		string	true	"user uuid"
// @Param			url	formData		string	true	"url"
// @Success		200			{object}	createWebsiteResp
// @Failure		400			{object}	errResp
// @Router			/api/web-watcher/websites [post]
func createWebsiteHandler(r repository.Repostory, conf *config.WebsiteConfig, tasks websiteupdate.WebsiteUpdateTasks) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		tr := otel.Tracer("htchan/WebHistory/api")

		// userUUID, err := UserUUID(req)
		userUUID := req.Context().Value(ContextKeyUserUUID).(string)
		url := req.Context().Value(ContextKeyWebURL).(string)

		web := model.NewWebsite(url, conf)

		dbCtx, dbSpan := tr.Start(req.Context(), "Website Record Creation")
		defer dbSpan.End()

		err := r.CreateWebsite(&web)
		if err != nil {
			dbSpan.SetStatus(codes.Error, err.Error())
			dbSpan.RecordError(err)

			zerolog.Ctx(dbCtx).Error().Err(err).Msg("create website failed")
			writeError(res, http.StatusBadRequest, err)
			return
		}

		dbSpan.End()

		dbCtx, dbSpan = tr.Start(req.Context(), "User Website Record Creation")
		defer dbSpan.End()

		userWeb := model.NewUserWebsite(web, userUUID)
		err = r.CreateUserWebsite(&userWeb)
		if err != nil {
			dbSpan.SetStatus(codes.Error, err.Error())
			dbSpan.RecordError(err)

			zerolog.Ctx(dbCtx).Error().Err(err).Msg("create user website failed")
			writeError(res, http.StatusBadRequest, err)
			return
		}

		dbSpan.End()

		// only publish job it website is updated more than 24 hr ago
		if time.Since(web.UpdateTime) > 24*time.Hour {
			jobCtx, jobSpan := tr.Start(req.Context(), "Website Update Job Creation")
			defer jobSpan.End()

			supportedList, err := tasks.Publish(jobCtx, &web)
			if err != nil {
				jobSpan.SetStatus(codes.Error, err.Error())
				jobSpan.RecordError(err)

				zerolog.Ctx(jobCtx).Error().Err(err).
					Msg("publish website update task failed")
				writeError(res, http.StatusBadRequest, err)

				return
			} else if len(supportedList) == 0 {
				jobSpan.SetStatus(codes.Error, "unsupported website")
				jobSpan.RecordError(errors.New("unsupported website"))

				zerolog.Ctx(jobCtx).Error().Err(err).
					Msg("unsupported website")
				writeError(res, http.StatusBadRequest, errors.New("unsupported website"))

				return
			}

			jobSpan.End()
		}

		encodeJsonResp(req.Context(), res, createWebsiteResp{fmt.Sprintf("website <%v> inserted", web.Title)})
	}
}

// @Summary		Get user website
// @description	get user website
// @Tags			web-history
// @Accept			json
// @Produce		json
// @Param			X-USER-UUID	header		string	true	"user uuid"
// @Param			websiteUUID	path		string	true	"website uuid"
// @Success		200			{object}	getUserWebsiteResp
// @Failure		400			{object}	errResp
// @Router			/api/web-watcher/websites/{websiteUUID} [get]
func getUserWebsiteHandler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		web := req.Context().Value(ContextKeyWebsite).(model.UserWebsite)

		encodeJsonResp(req.Context(), res, getUserWebsiteResp{fromModelUserWebsite(web)})
	}
}

// @Summary		Update user website
// @description	update user website
// @Tags			web-history
// @Accept			json
// @Produce		json
// @Param			X-USER-UUID	header		string	true	"user uuid"
// @Param			websiteUUID	path		string	true	"website uuid"
// @Success		200			{object}	refreshWebsiteResp
// @Failure		400			{object}	errResp
// @Router			/api/web-watcher/websites/{websiteUUID}/refresh [put]
func refreshWebsiteHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		tr := otel.Tracer("htchan/WebHistory/api")

		web := req.Context().Value(ContextKeyWebsite).(model.UserWebsite)
		web.AccessTime = time.Now().UTC().Truncate(5 * time.Second)

		dbCtx, dbSpan := tr.Start(req.Context(), "Refresh User Website")
		defer dbSpan.End()

		err := r.UpdateUserWebsite(&web)
		if err != nil {
			dbSpan.SetStatus(codes.Error, err.Error())
			dbSpan.RecordError(err)

			zerolog.Ctx(dbCtx).Error().Err(err).Msg("refresh user website failed")
			writeError(res, http.StatusInternalServerError, err)

			return
		}

		dbSpan.End()

		encodeJsonResp(req.Context(), res, refreshWebsiteResp{fromModelUserWebsite(web)})
	}
}

// write swagger docs
//
//	@Summary		Delete user website
//	@description	delete user website
//	@Tags			web-history
//	@Accept			json
//	@Produce		json
//	@Param			X-USER-UUID	header		string	true	"user uuid"
//	@Param			websiteUUID	path		string	true	"website uuid"
//	@Success		200			{object}	deleteWebsiteResp
//	@Failure		400			{object}	errResp
//	@Router			/api/web-watcher/websites/{websiteUUID} [delete]
func deleteWebsiteHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		tr := otel.Tracer("htchan/WebHistory/api")

		web := req.Context().Value(ContextKeyWebsite).(model.UserWebsite)

		dbCtx, dbSpan := tr.Start(req.Context(), "Delete User Website")
		defer dbSpan.End()

		err := r.DeleteUserWebsite(&web)
		if err != nil {
			dbSpan.SetStatus(codes.Error, err.Error())
			dbSpan.RecordError(err)

			zerolog.Ctx(dbCtx).Error().Err(err).Msg("delete user website failed")
			writeError(res, http.StatusInternalServerError, err)

			return
		}

		dbSpan.End()

		encodeJsonResp(req.Context(), res, deleteWebsiteResp{fmt.Sprintf("website <%v> deleted", web.Website.Title)})
	}
}

func validGroupName(web model.UserWebsite, groupName string) bool {
	for _, char := range strings.Split(groupName, "") {
		if strings.Contains(web.Website.Title, char) {
			return true
		}
	}
	return false
}

// write swagger docs
//
//	@Summary		Change website group
//	@description	change website group
//	@Tags			web-history
//	@Accept			json
//	@Produce		json
//	@Param			X-USER-UUID	header		string	true	"user uuid"
//	@Param			websiteUUID	path		string	true	"website uuid"
//	@Param			group_name	formData		string	true	"group name"
//	@Success		200			{object}	changeWebsiteGroupResp
//	@Failure		400			{object}	errResp
//	@Router			/api/web-watcher/websites/{websiteUUID}/change-group [put]
func changeWebsiteGroupHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		tr := otel.Tracer("htchan/WebHistory/api")

		web := req.Context().Value(ContextKeyWebsite).(model.UserWebsite)
		groupName := req.Context().Value(ContextKeyGroup).(string)
		if !validGroupName(web, groupName) {
			writeError(res, http.StatusBadRequest, errors.New("invalid group name"))
			return
		}

		web.GroupName = groupName

		dbCtx, dbSpan := tr.Start(req.Context(), "Update User Website Group")
		defer dbSpan.End()

		err := r.UpdateUserWebsite(&web)
		if err != nil {
			dbSpan.SetStatus(codes.Error, err.Error())
			dbSpan.RecordError(err)

			zerolog.Ctx(dbCtx).Error().Err(err).Str("group_name", groupName).Msg("update user website group failed")
			writeError(res, http.StatusBadRequest, err)

			return
		}

		dbSpan.End()

		encodeJsonResp(req.Context(), res, changeWebsiteGroupResp{fromModelUserWebsite(web)})
	}
}

func dbStatsHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		json.NewEncoder(res).Encode(r.Stats())
	}
}
