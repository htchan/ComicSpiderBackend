package website

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type ContextKey string

const (
	ContextKeyReqID    ContextKey = "req_id"
	ContextKeyUserUUID ContextKey = "user_uuid"
	ContextKeyWebURL   ContextKey = "web_url"
	ContextKeyWebsite  ContextKey = "website"
	ContextKeyGroup    ContextKey = "group"

	HeaderKeyUserUUID string = "X-USER-UUID"
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

				start := time.Now().UTC().Truncate(5 * time.Second)
				next.ServeHTTP(res, req.WithContext(logger.WithContext(ctx)))

				logger.Info().
					Str("path", req.URL.String()).
					Str("duration", time.Since(start).String()).
					Msg("request handled")
			},
		)
	}
}

func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(res http.ResponseWriter, req *http.Request) {
			tr := otel.Tracer("htchan/WebHistory/api")
			ctx, span := tr.Start(req.Context(), fmt.Sprintf("%s %s", req.Method, req.RequestURI))
			defer span.End()

			next.ServeHTTP(res, req.WithContext(ctx))
		},
	)
}

func AuthenticateMiddleware(conf *config.UserServiceConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(res http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodOptions {
					next.ServeHTTP(res, req)
					return
				}

				tr := otel.Tracer("htchan/WebHistory/api")

				_, authSpan := tr.Start(req.Context(), "authentication")
				defer authSpan.End()

				userUUID := req.Header.Get(HeaderKeyUserUUID)

				if _, err := uuid.Parse(userUUID); err != nil {
					authSpan.SetStatus(codes.Error, ErrUnauthorized.Error())
					authSpan.RecordError(ErrUnauthorized)

					writeError(res, http.StatusUnauthorized, ErrUnauthorized)

					return
				}

				zerolog.Ctx(req.Context()).Debug().
					Str("user_uuid", userUUID).
					Msg("set params")
				ctx := context.WithValue(req.Context(), ContextKeyUserUUID, userUUID)
				authSpan.End()

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
			tr := otel.Tracer("htchan/WebHistory/api")

			_, paramsSpan := tr.Start(req.Context(), "parse website params")
			defer paramsSpan.End()

			err := req.ParseForm()
			if err != nil {
				paramsSpan.SetStatus(codes.Error, ErrInvalidParams.Error())
				paramsSpan.RecordError(ErrInvalidParams)

				writeError(res, http.StatusBadRequest, ErrInvalidParams)

				return
			}

			url := req.Form.Get("url")
			if url == "" || !strings.HasPrefix(url, "http") {
				paramsSpan.SetStatus(codes.Error, ErrInvalidParams.Error())
				paramsSpan.RecordError(ErrInvalidParams)

				writeError(res, http.StatusBadRequest, ErrInvalidParams)

				return
			}

			zerolog.Ctx(req.Context()).Debug().
				Str("web url", url).
				Msg("set params")
			ctx := context.WithValue(req.Context(), ContextKeyWebURL, url)
			paramsSpan.End()

			next.ServeHTTP(res, req.WithContext(ctx))
		},
	)
}

func QueryUserWebsite(r repository.Repostory) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(res http.ResponseWriter, req *http.Request) {
				tr := otel.Tracer("htchan/WebHistory/api")

				userUUID := req.Context().Value(ContextKeyUserUUID).(string)
				webUUID := chi.URLParam(req, "webUUID")

				_, dbSpan := tr.Start(req.Context(), "query user website")
				defer dbSpan.End()

				web, err := r.FindUserWebsite(req.Context(), userUUID, webUUID)
				if err != nil {
					dbSpan.SetStatus(codes.Error, ErrInvalidParams.Error())
					dbSpan.RecordError(ErrInvalidParams)

					writeError(res, http.StatusBadRequest, err)
					return
				}

				dbSpan.End()

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
			tr := otel.Tracer("htchan/WebHistory/api")
			_, paramsSpan := tr.Start(req.Context(), "parse website params")
			defer paramsSpan.End()

			err := req.ParseForm()
			if err != nil {
				paramsSpan.SetStatus(codes.Error, ErrInvalidParams.Error())
				paramsSpan.RecordError(ErrInvalidParams)

				writeError(res, http.StatusBadRequest, ErrInvalidParams)

				return
			}

			groupName := req.Form.Get("group_name")
			zerolog.Ctx(req.Context()).Debug().
				Str("group name", groupName).
				Msg("set params")
			ctx := context.WithValue(req.Context(), ContextKeyGroup, groupName)
			paramsSpan.End()

			next.ServeHTTP(res, req.WithContext(ctx))
		},
	)
}
