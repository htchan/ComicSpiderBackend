package config

import (
	"os"
	"strings"
	"time"
)

type VendorServiceConfig struct {
	MaxConcurrency int64         `yaml:"max_concurrency"`
	FetchInterval  time.Duration `yaml:"fetch_interval"`
	MaxRetry       int           `yaml:"max_retry"`
	RetryInterval  time.Duration `yaml:"retry_interval"`
}

type BaozimhConfig struct {
	Cookie map[string]string `yaml:"cookie"`
}

func LoadBaozimhConfig() *BaozimhConfig {
	cookieStr := os.Getenv("BAOZIMH_COOKIES")
	if cookieStr == "" {
		return &BaozimhConfig{}
	}

	cookieMap := make(map[string]string)
	pairs := strings.Split(cookieStr, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split by the first '=' to separate name and value
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			cookieMap[key] = val
		}
	}

	return &BaozimhConfig{Cookie: cookieMap}
}
