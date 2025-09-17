package vendors

import (
	"context"
	"fmt"
	"net/http"

	"github.com/htchan/WebHistory/internal/model"
	"github.com/htchan/goclient"
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

func RaiseStatusCodeErrorMiddleware(f goclient.Requester) goclient.Requester {
	return func(req *http.Request) (*http.Response, error) {
		resp, err := f(req)
		if err != nil || (resp.StatusCode >= 200 && resp.StatusCode < 300) {
			return resp, err
		}

		return resp, fmt.Errorf("fetch website failed: %w (%d)", ErrInvalidStatusCode, resp.StatusCode)
	}
}
