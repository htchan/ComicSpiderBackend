package vendors

import (
	"context"
	"fmt"

	"github.com/htchan/WebHistory/internal/model"
)

var ErrInvalidStatusCode = fmt.Errorf("invalid status code")

//go:generate mockgen -destination=../mock/vendor/vendor_service.go -package=mockvendor . VendorService
type VendorService interface {
	Support(*model.Website) bool
	Update(context.Context, *model.Website) error
	// Download(context.Context, *model.Website) error
}
