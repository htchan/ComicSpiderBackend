package vendorhelper

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository"
	"github.com/htchan/WebHistory/internal/vendors"
	"github.com/htchan/WebHistory/internal/vendors/baozimh"
	"github.com/htchan/WebHistory/internal/vendors/kuaikanmanhua"
	"github.com/htchan/WebHistory/internal/vendors/manhuagui"
	"github.com/htchan/WebHistory/internal/vendors/manhuaren"
	"github.com/htchan/WebHistory/internal/vendors/qiman6"
	"github.com/htchan/WebHistory/internal/vendors/u17"
	"github.com/htchan/WebHistory/internal/vendors/webtoons"
)

func NewServiceSet(cli *http.Client, rpo repository.Repostory, cfgs map[string]config.VendorServiceConfig) ([]vendors.VendorService, error) {
	var services []vendors.VendorService
	var err error
	for key, cfg := range cfgs {
		cfg := cfg
		switch key {
		case baozimh.Host:
			services = append(services, baozimh.NewVendorService(cli, rpo, &cfg))
		case kuaikanmanhua.Host:
			services = append(services, kuaikanmanhua.NewVendorService(cli, rpo, &cfg))
		case manhuagui.Host:
			services = append(services, manhuagui.NewVendorService(cli, rpo, &cfg))
		case manhuaren.Host:
			services = append(services, manhuaren.NewVendorService(cli, rpo, &cfg))
		case qiman6.Host:
			services = append(services, qiman6.NewVendorService(cli, rpo, &cfg))
		case u17.Host:
			services = append(services, u17.NewVendorService(cli, rpo, &cfg))
		case webtoons.Host:
			services = append(services, webtoons.NewVendorService(cli, rpo, &cfg))
		default:
			err = errors.Join(err, fmt.Errorf("%w: %s", vendors.ErrUnknownHost, key))
		}
	}

	return services, err
}
