package env

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/sethvargo/go-envconfig"
)

type EnvConfig struct {
	HandlerExecutableType string          `json:"handler_executable_type"`
	Cassandra             CassandraConfig `json:"cassandra"`
	Amqp                  AmqpConfig      `json:"amqp"`
	// ZapConfig                         zap.Config                 `json:"zap_config"`
	Log                               LogConfig                    `json:"log"`
	ThreadPoolSize                    int                          `json:"thread_pool_size" env:"CAPI_THREAD_POOL_SIZE, overwrite"`
	DeadLetterTtl                     int                          `json:"dead_letter_ttl" env:"CAPI_DEAD_LETTER_TTL, overwrite"`
	CaPath                            string                       `json:"ca_path" env:"CAPI_CA_PATH, overwrite"`
	PrivateKeys                       map[string]string            `json:"private_keys" env:"CAPI_PRIVATE_KEYS, overwrite"`
	Webapi                            WebapiConfig                 `json:"webapi,omitempty"`
	CustomProcessorsSettings          map[string]json.RawMessage   `json:"custom_processors"`
	CustomProcessorDefFactoryInstance sc.CustomProcessorDefFactory `json:"-"`
}

func (ec *EnvConfig) Deserialize(ctx context.Context, jsonBytes []byte) error {
	err := json.Unmarshal(jsonBytes, ec)
	if err != nil {
		return fmt.Errorf("cannot deserialize env config: %s", err.Error())
	}

	if err := envconfig.Process(ctx, ec); err != nil {
		return fmt.Errorf("cannot process env variables: %s", err.Error())
	}

	// Defaults

	if ec.ThreadPoolSize <= 0 || ec.ThreadPoolSize > 100 {
		ec.ThreadPoolSize = 5
	}

	if ec.DeadLetterTtl < 100 || ec.DeadLetterTtl > 3600000 { // [100ms,1hr]
		ec.DeadLetterTtl = 10000
	}

	return nil
}

func ReadEnvConfigFile(ctx context.Context, envConfigFile string) (*EnvConfig, error) {
	exec, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot find current executable path: %s", err.Error())
	}
	configFullPath := filepath.Join(filepath.Dir(exec), envConfigFile)
	if _, err := os.Stat(configFullPath); err != nil {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot get current dir: [%s]", err.Error())
		}
		configFullPath = filepath.Join(cwd, envConfigFile)
		if _, err := os.Stat(configFullPath); err != nil {
			return nil, fmt.Errorf("cannot find config file [%s], neither at [%s] nor at current dir [%s]: [%s]", envConfigFile, filepath.Dir(exec), filepath.Join(cwd, envConfigFile), err.Error())
		}
	}

	envBytes, err := os.ReadFile(configFullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read env config file %s: %s", configFullPath, err.Error())
	}

	var envConfig EnvConfig
	if err := envConfig.Deserialize(ctx, envBytes); err != nil {
		return nil, fmt.Errorf("cannot parse env config file %s: %s", configFullPath, err.Error())
	}

	return &envConfig, nil
}

func (ec *EnvConfig) String() string {
	jsonBytes, err := json.Marshal(ec)
	if err != nil {
		return err.Error()
	}

	return string(jsonBytes)
}
