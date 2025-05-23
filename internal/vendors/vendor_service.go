package vendors

import (
	"context"
	"fmt"

	"github.com/htchan/WebHistory/internal/model"
)

var ErrInvalidStatusCode = fmt.Errorf("invalid status code")
var ErrUnknownHost = fmt.Errorf("unknown host")

//go:generate go tool mockgen -destination=../mock/vendor/vendor_service.go -package=mockvendor . VendorService
type VendorService interface {
	Support(*model.Website) bool
	Update(context.Context, *model.Website) error
	Name() string
	// Download(context.Context, *model.Website) error
}
