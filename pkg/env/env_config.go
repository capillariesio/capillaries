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
	HandlerExecutableType             string                       `json:"handler_executable_type"` // daemon,webapi,toolbelt
	Cassandra                         CassandraConfig              `json:"cassandra"`
	Amqp                              Amqp091Config                `json:"amqp091"`
	Log                               LogConfig                    `json:"log"`
	CaPath                            string                       `json:"ca_path" env:"CAPI_CA_PATH, overwrite"`           // Used for HTTP, host's CA dir if empty
	PrivateKeys                       map[string]string            `json:"private_keys" env:"CAPI_PRIVATE_KEYS, overwrite"` // Used for SFTP only
	Daemon                            DaemonConfig                 `json:"daemon,omitempty"`
	Webapi                            WebapiConfig                 `json:"webapi,omitempty"`
	CustomProcessorsSettings          map[string]json.RawMessage   `json:"custom_processors"`
	CustomProcessorDefFactoryInstance sc.CustomProcessorDefFactory `json:"-"`
	// ZapConfig                      zap.Config                   `json:"zap_config"`
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

	if ec.Daemon.ThreadPoolSize <= 0 || ec.Daemon.ThreadPoolSize > 100 {
		ec.Daemon.ThreadPoolSize = 5
	}

	if ec.Daemon.DeadLetterTtl < 100 || ec.Daemon.DeadLetterTtl > 3600000 { // [100ms,1hr]
		ec.Daemon.DeadLetterTtl = 10000
	}
	if ec.Cassandra.InserterCapacity < 50 || ec.Cassandra.InserterCapacity > 10000 {
		ec.Cassandra.InserterCapacity = 500
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
