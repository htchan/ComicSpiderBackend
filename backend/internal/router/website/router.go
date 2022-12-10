package website

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repo"
)

var UnauthorizedError = errors.New("unauthorized")
var InvalidParamsError = errors.New("invalid params")
var RecordNotFoundError = errors.New("record not found")

func writeError(res http.ResponseWriter, statusCode int, err error) {
	res.WriteHeader(statusCode)
	fmt.Fprintln(res, fmt.Sprintf(`{ "error": "%v" }`, err))
}

func redirectLogin(res http.ResponseWriter, req *http.Request) {
	loginURL := os.Getenv("LOGIN_URL")
	serviceUUID := os.Getenv("SERVICE_UUID")
	http.Redirect(res, req, fmt.Sprintf("%v?service=%v", loginURL, serviceUUID), 302)
}

func AddRoutes(router chi.Router, r repo.Repostory, conf *config.Config) {
	api_route_prefix := os.Getenv("WEB_WATCHER_API_ROUTE_PREFIX")
	if api_route_prefix == "" {
		api_route_prefix = "/api/web-watcher"
	}
	router.Route(api_route_prefix, func(router chi.Router) {
		router.Route("/websites", func(router chi.Router) {
			router.Use(
				cors.Handler(
					cors.Options{
						AllowedOrigins: []string{"*"},
						AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
						AllowedHeaders: []string{"*"},
						MaxAge:         300, // Maximum value not ignored by any of major browsers
					},
				),
			)
			router.Use(Authenticate)
			router.Use(SetContentType)

			router.Route("/groups", func(router chi.Router) {
				router.Get("/", getAllWebsiteGroupsHandler(r))
				router.Get("/{groupName}", getWebsiteGroupHandler(r))
			})

			router.With(WebsiteParams).Post("/", createWebsiteHandler(r, conf))

			router.With(QueryWebsite(r)).Route("/{webUUID}", func(router chi.Router) {
				router.Get("/", getWebsiteHandler(r))
				router.Delete("/", deleteWebsiteHandler(r))
				router.Put("/refresh", refreshWebsiteHandler(r))
				router.With(GroupNameParams).Put("/change-group", changeWebsiteGroupHandler(r))
			})
		})
		router.Get("/db-stats", dbStatsHandler(r))
	})
}
