package vendorhelper

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"

	// Import all vendor packages so their init() functions run
	_ "github.com/htchan/WebHistory/internal/vendors/baozimh"
	_ "github.com/htchan/WebHistory/internal/vendors/kuaikanmanhua"
	_ "github.com/htchan/WebHistory/internal/vendors/manhuagui"
	_ "github.com/htchan/WebHistory/internal/vendors/manhuaren"
	_ "github.com/htchan/WebHistory/internal/vendors/qiman6"
	_ "github.com/htchan/WebHistory/internal/vendors/u17"
	_ "github.com/htchan/WebHistory/internal/vendors/webtoons"
)

func NewServiceSet(cli *http.Client, rpo repository.Repository, cfgs map[string]config.VendorServiceConfig) ([]vendors.VendorService, error) {
	var services []vendors.VendorService
	var err error
	for key, cfg := range cfgs {
		cfg := cfg
		factory := vendors.GetFactory(key)
		if factory != nil {
			services = append(services, factory(cli, rpo, &cfg))
		} else {
			err = errors.Join(err, fmt.Errorf("%w: %s", vendors.ErrUnknownHost, key))
		}
	}

	return services, err
}
