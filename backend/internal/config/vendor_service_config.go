package config

import "time"

type VendorServiceConfig struct {
	MaxConcurrency int64
	FetchInterval  time.Duration
	MaxRetry       int
	RetryInterval  time.Duration
}
