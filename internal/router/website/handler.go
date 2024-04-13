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

	"github.com/go-chi/chi/v5"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/service"
)

func getAllWebsiteGroupsHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		userUUID := req.Context().Value(ContextKeyUserUUID).(string)
		webs, err := r.FindUserWebsites(userUUID)
		if err != nil {
			zerolog.Ctx(req.Context()).Error().Err(err).Msg("find user websites failed")
			writeError(res, http.StatusBadRequest, RecordNotFoundError)
			return
		}

		json.NewEncoder(res).Encode(map[string]interface{}{
			"website_groups": webs.WebsiteGroups(),
		})
	}
}

func getWebsiteGroupHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		userUUID := req.Context().Value(ContextKeyUserUUID).(string)
		groupName := chi.URLParam(req, "groupName")
		webs, err := r.FindUserWebsitesByGroup(userUUID, groupName)
		if err != nil || len(webs) == 0 {
			zerolog.Ctx(req.Context()).Error().Err(err).Msg("find user websites by group failed")
			writeError(res, http.StatusBadRequest, RecordNotFoundError)
			return
		}

		json.NewEncoder(res).Encode(
			map[string]interface{}{"website_group": webs},
		)
	}
}

func createWebsiteHandler(r repository.Repostory, conf *config.WebsiteConfig) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// userUUID, err := UserUUID(req)
		userUUID := req.Context().Value(ContextKeyUserUUID).(string)
		url := req.Context().Value(ContextKeyWebURL).(string)

		web := model.NewWebsite(url, conf)
		service.Update(context.Background(), r, &web)

		err := r.CreateWebsite(&web)
		if err != nil {
			zerolog.Ctx(req.Context()).Error().Err(err).Msg("create website failed")
			writeError(res, http.StatusBadRequest, err)
			return
		}

		userWeb := model.NewUserWebsite(web, userUUID)
		err = r.CreateUserWebsite(&userWeb)
		if err != nil {
			zerolog.Ctx(req.Context()).Error().Err(err).Msg("create user website failed")
			writeError(res, http.StatusBadRequest, err)
			return
		}

		json.NewEncoder(res).Encode(map[string]interface{}{
			"message": fmt.Sprintf("website <%v> inserted", web.Title),
		})
	}
}

func getWebsiteHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		web := req.Context().Value(ContextKeyWebsite).(model.UserWebsite)

		json.NewEncoder(res).Encode(map[string]interface{}{
			"website": web,
		})
	}
}

func refreshWebsiteHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		web := req.Context().Value(ContextKeyWebsite).(model.UserWebsite)
		web.AccessTime = time.Now().UTC().Truncate(time.Second)

		err := r.UpdateUserWebsite(&web)
		if err != nil {
			zerolog.Ctx(req.Context()).Error().Err(err).Msg("update user website failed")
			writeError(res, http.StatusInternalServerError, err)
			return
		}

		json.NewEncoder(res).Encode(map[string]interface{}{
			"website": web,
		})
	}
}

func deleteWebsiteHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		web := req.Context().Value(ContextKeyWebsite).(model.UserWebsite)

		err := r.DeleteUserWebsite(&web)
		if err != nil {
			zerolog.Ctx(req.Context()).Error().Err(err).Msg("delete user website failed")
			writeError(res, http.StatusInternalServerError, err)
			return
		}

		json.NewEncoder(res).Encode(map[string]interface{}{
			"message": fmt.Sprintf("website <%v> deleted", web.Website.Title),
		})
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

func changeWebsiteGroupHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		web := req.Context().Value(ContextKeyWebsite).(model.UserWebsite)
		groupName := req.Context().Value(ContextKeyGroup).(string)
		if !validGroupName(web, groupName) {
			writeError(res, http.StatusBadRequest, errors.New("invalid group name"))
			return
		}
		web.GroupName = groupName
		err := r.UpdateUserWebsite(&web)
		if err != nil {
			writeError(res, http.StatusBadRequest, err)
			return
		}
		json.NewEncoder(res).Encode(map[string]interface{}{
			"website": web,
		})
	}
}

func dbStatsHandler(r repository.Repostory) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		json.NewEncoder(res).Encode(r.Stats())
	}
}
