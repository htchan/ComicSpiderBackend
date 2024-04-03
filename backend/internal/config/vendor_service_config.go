package config

import "time"

type VendorServiceConfig struct {
	MaxConcurrency int64         `yaml:"max_concurrency"`
	FetchInterval  time.Duration `yaml:"fetch_interval"`
	MaxRetry       int           `yaml:"max_retry"`
	RetryInterval  time.Duration `yaml:"retry_interval"`
}
