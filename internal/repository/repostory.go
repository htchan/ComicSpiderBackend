package repository

import (
	"context"
	"database/sql"

	"github.com/htchan/WebHistory/internal/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

//go:generate mockgen -destination=../mock/repository/repository.go -package=mockrepo . Repostory
type Repostory interface {
	CreateWebsite(context.Context, *model.Website) error
	UpdateWebsite(context.Context, *model.Website) error
	DeleteWebsite(context.Context, *model.Website) error

	FindWebsites(context.Context) ([]model.Website, error)
	FindWebsite(ctx context.Context, uuid string) (*model.Website, error)

	CreateUserWebsite(context.Context, *model.UserWebsite) error
	UpdateUserWebsite(context.Context, *model.UserWebsite) error
	DeleteUserWebsite(context.Context, *model.UserWebsite) error

	FindUserWebsites(ctx context.Context, userUUID string) (model.UserWebsites, error)
	FindUserWebsitesByGroup(ctx context.Context, userUUID, group string) (model.WebsiteGroup, error)
	FindUserWebsite(ctx context.Context, userUUID, websiteUUID string) (*model.UserWebsite, error)

	Stats() sql.DBStats
}

func GetTracer() trace.Tracer {
	return otel.Tracer("htchan/WebHistory/repository")
}
