package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	leak := flag.Bool("leak", false, "check for memory leaks")
	flag.Parse()

	if *leak {
		goleak.VerifyTestMain(m)
	} else {
		os.Exit(m.Run())
	}
}

func Test_LoadAPIConfig(t *testing.T) {
	tests := []struct {
		name         string
		envMap       map[string]string
		expectedConf *APIConfig
		expectError  bool
	}{
		{
			name: "happy flow with default",
			envMap: map[string]string{
				"PSQL_HOST":          "host",
				"PSQL_PORT":          "port",
				"PSQL_USER":          "user",
				"PSQL_PASSWORD":      "password",
				"PSQL_NAME":          "name",
				"USER_SERVICE_ADDR":  "user_serv_addr",
				"USER_SERVICE_TOKEN": "user_serv_token",
				"VENDOR_CONFIG_PATH": "../../data/testing/vendor_configs.yml",
				"REDIS_CLIENT_ADDR":  "redis:6379",
			},
			expectedConf: &APIConfig{
				VendorConfigPath: "../../data/testing/vendor_configs.yml",
				BinConfig: APIBinConfig{
					ReadTimeout:    5 * time.Second,
					WriteTimeout:   5 * time.Second,
					IdleTimeout:    5 * time.Second,
					APIRoutePrefix: "/api/web-watcher",
					VendorServiceConfigs: map[string]VendorServiceConfig{
						"testing": {
							MaxConcurrency: 10,
							FetchInterval:  time.Second,
							MaxRetry:       10,
							RetryInterval:  time.Second,
						},
					},
				},
				DatabaseConfig: DatabaseConfig{
					Driver:   "postgres",
					Host:     "host",
					Port:     "port",
					User:     "user",
					Password: "password",
					Database: "name",
				},
				UserServiceConfig: UserServiceConfig{
					Addr: "user_serv_addr", Token: "user_serv_token",
				},
				WebsiteConfig: WebsiteConfig{
					Separator:     "\n",
					MaxDateLength: 2,
				},
				RedisStreamConfig: RedisStreamConfig{
					Addr: "redis:6379",
				},
			},
			expectError: false,
		},
		{
			name: "happy flow without default",
			envMap: map[string]string{
				"WEB_WATCHER_SEPARATOR":        ",",
				"WEB_WATCHER_DATE_MAX_LENGTH":  "10",
				"ADDR":                         "addr",
				"API_READ_TIMEOUT":             "1s",
				"API_WRITE_TIMEOUT":            "1s",
				"API_IDLE_TIMEOUT":             "1s",
				"WEB_WATCHER_API_ROUTE_PREFIX": "prefix",
				"OTEL_URL":                     "otel_url",
				"OTEL_SERVICE_NAME":            "otel_service_name",
				"DRIVER":                       "driver",
				"PSQL_HOST":                    "host",
				"PSQL_PORT":                    "port",
				"PSQL_USER":                    "user",
				"PSQL_PASSWORD":                "password",
				"PSQL_NAME":                    "name",
				"USER_SERVICE_ADDR":            "user_serv_addr",
				"USER_SERVICE_TOKEN":           "user_serv_token",
				"VENDOR_CONFIG_PATH":           "../../data/testing/vendor_configs.yml",
				"REDIS_CLIENT_ADDR":            "redis:6379",
			},
			expectedConf: &APIConfig{
				VendorConfigPath: "../../data/testing/vendor_configs.yml",
				BinConfig: APIBinConfig{
					Addr:           "addr",
					ReadTimeout:    1 * time.Second,
					WriteTimeout:   1 * time.Second,
					IdleTimeout:    1 * time.Second,
					APIRoutePrefix: "prefix",
					VendorServiceConfigs: map[string]VendorServiceConfig{
						"testing": {
							MaxConcurrency: 10,
							FetchInterval:  time.Second,
							MaxRetry:       10,
							RetryInterval:  time.Second,
						},
					},
				},
				TraceConfig: TraceConfig{
					OtelURL:         "otel_url",
					OtelServiceName: "otel_service_name",
				},
				DatabaseConfig: DatabaseConfig{
					Driver:   "driver",
					Host:     "host",
					Port:     "port",
					User:     "user",
					Password: "password",
					Database: "name",
				},
				UserServiceConfig: UserServiceConfig{
					Addr: "user_serv_addr", Token: "user_serv_token",
				},
				WebsiteConfig: WebsiteConfig{
					Separator:     ",",
					MaxDateLength: 10,
				},
				RedisStreamConfig: RedisStreamConfig{
					Addr: "redis:6379",
				},
			},
			expectError: false,
		},
		{
			name:         "not providing required error",
			envMap:       map[string]string{},
			expectedConf: nil,
			expectError:  true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			// populate env
			for key, value := range test.envMap {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			conf, err := LoadAPIConfig()
			assert.Equal(t, test.expectedConf, conf)
			if !assert.Equal(t, test.expectError, (err != nil)) {
				assert.Fail(t, "error is %s", err)
			}
		})
	}
}

func Test_LoadWorkerConfig(t *testing.T) {
	tests := []struct {
		name      string
		envMap    map[string]string
		want      *WorkerConfig
		wantError error
	}{
		{
			name: "happy flow with default",
			envMap: map[string]string{
				"WEBSITE_UPDATE_SLEEP_INTERVAL": "10s",
				"WORKER_EXECUTOR_COUNT":         "10",
				"PSQL_HOST":                     "host",
				"PSQL_PORT":                     "port",
				"PSQL_USER":                     "user",
				"PSQL_PASSWORD":                 "password",
				"PSQL_NAME":                     "name",
				"VENDOR_CONFIG_PATH":            "../../data/testing/vendor_configs.yml",
				"REDIS_CLIENT_ADDR":             "redis:6379",
			},
			want: &WorkerConfig{
				VendorConfigPath: "../../data/testing/vendor_configs.yml",
				BinConfig: WorkerBinConfig{
					WebsiteUpdateSleepInterval: 10 * time.Second,
					WorkerExecutorCount:        10,
					ClientTimeout:              30 * time.Second,
					VendorServiceConfigs: map[string]VendorServiceConfig{
						"testing": {
							MaxConcurrency: 10,
							FetchInterval:  time.Second,
							MaxRetry:       10,
							RetryInterval:  time.Second,
						},
					},
				},
				DatabaseConfig: DatabaseConfig{
					Driver:   "postgres",
					Host:     "host",
					Port:     "port",
					User:     "user",
					Password: "password",
					Database: "name",
				},
				WebsiteConfig: WebsiteConfig{
					Separator:     "\n",
					MaxDateLength: 2,
				},
				RedisStreamConfig: RedisStreamConfig{
					Addr: "redis:6379",
				},
			},
			wantError: nil,
		},
		{
			name: "happy flow without default",
			envMap: map[string]string{
				"WEB_WATCHER_SEPARATOR":         ",",
				"WEB_WATCHER_DATE_MAX_LENGTH":   "10",
				"WEBSITE_UPDATE_SLEEP_INTERVAL": "10s",
				"CLIENT_TIMEOUT":                "1s",
				"WORKER_EXECUTOR_COUNT":         "10",
				"OTEL_URL":                      "otel_url",
				"OTEL_SERVICE_NAME":             "otel_service_name",
				"DRIVER":                        "driver",
				"PSQL_HOST":                     "host",
				"PSQL_PORT":                     "port",
				"PSQL_USER":                     "user",
				"PSQL_PASSWORD":                 "password",
				"PSQL_NAME":                     "name",
				"VENDOR_CONFIG_PATH":            "../../data/testing/vendor_configs.yml",
				"REDIS_CLIENT_ADDR":             "redis:6379",
			},
			want: &WorkerConfig{
				VendorConfigPath: "../../data/testing/vendor_configs.yml",
				BinConfig: WorkerBinConfig{
					WebsiteUpdateSleepInterval: 10 * time.Second,
					WorkerExecutorCount:        10,
					ClientTimeout:              1 * time.Second,
					VendorServiceConfigs: map[string]VendorServiceConfig{
						"testing": {
							MaxConcurrency: 10,
							FetchInterval:  time.Second,
							MaxRetry:       10,
							RetryInterval:  time.Second,
						},
					},
				},
				TraceConfig: TraceConfig{
					OtelURL:         "otel_url",
					OtelServiceName: "otel_service_name",
				},
				DatabaseConfig: DatabaseConfig{
					Driver:   "driver",
					Host:     "host",
					Port:     "port",
					User:     "user",
					Password: "password",
					Database: "name",
				},
				WebsiteConfig: WebsiteConfig{
					Separator:     ",",
					MaxDateLength: 10,
				},
				RedisStreamConfig: RedisStreamConfig{
					Addr: "redis:6379",
				},
			},
			wantError: nil,
		},
		{
			name:      "not providing required error",
			envMap:    map[string]string{},
			want:      nil,
			wantError: env.EnvVarIsNotSetError{Key: "PSQL_NAME"},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			// populate env
			for key, value := range test.envMap {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			conf, err := LoadWorkerConfig()
			assert.Equal(t, test.want, conf)
			assert.ErrorIs(t, err, test.wantError)
		})
	}
}
