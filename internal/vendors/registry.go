package vendors

import (
	"net/http"
	"sync"

	"github.com/htchan/WebHistory/internal/config"
	"github.com/htchan/WebHistory/internal/repository"
)

// VendorFactory creates a VendorService given an HTTP client, repository, and config.
type VendorFactory func(cli *http.Client, rpo repository.Repository, cfg *config.VendorServiceConfig) VendorService

var (
	mu       sync.RWMutex
	registry = make(map[string]VendorFactory)
)

// RegisterFactory adds a vendor factory to the registry. Typically called from init() in each vendor package.
func RegisterFactory(host string, factory VendorFactory) {
	mu.Lock()
	defer mu.Unlock()
	registry[host] = factory
}

// RegisteredHosts returns all registered vendor host names.
func RegisteredHosts() []string {
	mu.RLock()
	defer mu.RUnlock()
	hosts := make([]string, 0, len(registry))
	for h := range registry {
		hosts = append(hosts, h)
	}
	return hosts
}

// GetFactory returns the factory for a given host, or nil if not registered.
func GetFactory(host string) VendorFactory {
	mu.RLock()
	defer mu.RUnlock()
	return registry[host]
}
