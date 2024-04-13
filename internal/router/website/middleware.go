package website

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ContextKey string

const (
	ContextKeyReqID    ContextKey = "req_id"
	ContextKeyUserUUID ContextKey = "user_uuid"
	ContextKeyWebURL   ContextKey = "web_url"
	ContextKeyWebsite  ContextKey = "website"
	ContextKeyGroup    ContextKey = "group"
)

func logRequest() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(res http.ResponseWriter, req *http.Request) {
				requestID := uuid.New()

				ctx := context.WithValue(req.Context(), ContextKeyReqID, requestID)
				logger := log.With().
					Str("request_id", requestID.String()).
					Logger()

				start := time.Now().UTC().Truncate(time.Second)
				next.ServeHTTP(res, req.WithContext(logger.WithContext(ctx)))

				logger.Info().
					Str("path", req.URL.String()).
					Str("duration", time.Since(start).String()).
					Msg("request handled")
			},
		)
	}
}

func AuthenticateMiddleware(conf *config.UserServiceConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(res http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodOptions {
					next.ServeHTTP(res, req)
					return
				}
				token := req.Header.Get("Authorization")
				userUUID := ""

				if token != "" {
					userUUID = utils.FindUserByToken(token, conf)
				}

				if userUUID == "" {
					writeError(res, http.StatusUnauthorized, UnauthorizedError)
					return
				}

				zerolog.Ctx(req.Context()).Debug().
					Str("user_uuid", userUUID).
					Msg("set params")
				ctx := context.WithValue(req.Context(), ContextKeyUserUUID, userUUID)
				next.ServeHTTP(res, req.WithContext(ctx))
			},
		)
	}
}
func SetContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(res http.ResponseWriter, req *http.Request) {
			res.Header().Set("Content-Type", "application/json; charset=utf-8")
			next.ServeHTTP(res, req)
		},
	)
}

func WebsiteParams(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(res http.ResponseWriter, req *http.Request) {
			err := req.ParseForm()
			if err != nil {
				writeError(res, http.StatusBadRequest, InvalidParamsError)
				return
			}
			url := req.Form.Get("url")
			if url == "" || !strings.HasPrefix(url, "http") {
				writeError(res, http.StatusBadRequest, InvalidParamsError)
				return
			}

			zerolog.Ctx(req.Context()).Debug().
				Str("web url", url).
				Msg("set params")
			ctx := context.WithValue(req.Context(), ContextKeyWebURL, url)
			next.ServeHTTP(res, req.WithContext(ctx))
		},
	)
}

func QueryWebsite(r repository.Repostory) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(res http.ResponseWriter, req *http.Request) {
				userUUID := req.Context().Value(ContextKeyUserUUID).(string)
				webUUID := chi.URLParam(req, "webUUID")
				web, err := r.FindUserWebsite(userUUID, webUUID)
				if err != nil {
					writeError(res, http.StatusBadRequest, err)
					return
				}

				zerolog.Ctx(req.Context()).Debug().
					Str("website uuid", web.WebsiteUUID).
					Str("user uuid", web.UserUUID).
					Msg("set params")
				ctx := context.WithValue(req.Context(), ContextKeyWebsite, *web)
				next.ServeHTTP(res, req.WithContext(ctx))
			},
		)
	}
}

func GroupNameParams(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(res http.ResponseWriter, req *http.Request) {
			err := req.ParseForm()
			if err != nil {
				writeError(res, http.StatusBadRequest, InvalidParamsError)
				return
			}

			groupName := req.Form.Get("group_name")
			zerolog.Ctx(req.Context()).Debug().
				Str("group name", groupName).
				Msg("set params")
			ctx := context.WithValue(req.Context(), ContextKeyGroup, groupName)
			next.ServeHTTP(res, req.WithContext(ctx))
		},
	)
}
