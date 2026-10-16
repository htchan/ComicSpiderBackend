package config

import (
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
	BrowserCdpUrl       string        `env:"BROWSER_CDP_URL"`
	BrowserWaitDuration time.Duration `env:"BROWSER_WAIT_DURATION" envDefault:"10s"`
}

func LoadBaozimhConfig() *BaozimhConfig {
	conf := BaozimhConfig{}
	env.Parse(&conf)

	return &conf
}
