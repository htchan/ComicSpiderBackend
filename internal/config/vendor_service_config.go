package config

import (
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

type VendorServiceConfig struct {
	MaxConcurrency int64         `yaml:"max_concurrency"`
	FetchInterval  time.Duration `yaml:"fetch_interval"`
	MaxRetry       int           `yaml:"max_retry"`
	RetryInterval  time.Duration `yaml:"retry_interval"`
}

type BaozimhConfig struct {
	Cookie              map[string]string `yaml:"cookie"`
	BrowserCdpUrl       string            `env:"BROWSER_CDP_URL"`
	BrowserWaitDuration time.Duration     `env:"BROWSER_WAIT_DURATION" envDefault:"10s"`
}

func LoadBaozimhConfig() *BaozimhConfig {
	conf := BaozimhConfig{}
	env.Parse(&conf)

	cookieStr := os.Getenv("BAOZIMH_COOKIES")
	if cookieStr == "" {
		return &conf
	}

	conf.Cookie = make(map[string]string)
	for pair := range strings.SplitSeq(cookieStr, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split by the first '=' to separate name and value
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			conf.Cookie[key] = val
		}
	}

	return &conf
}
