package env

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/capillariesio/capillaries/pkg/sc"
	"go.uber.org/zap"
)

type EnvConfig struct {
	HandlerExecutableType             string                     `json:"handler_executable_type"`
	Cassandra                         CassandraConfig            `json:"cassandra"`
	Amqp                              AmqpConfig                 `json:"amqp"`
	ZapConfig                         zap.Config                 `json:"zap_config"`
	ThreadPoolSize                    int                        `json:"thread_pool_size"`
	DeadLetterTtl                     int                        `json:"dead_letter_ttl"`
	CaPath                            string                     `json:"ca_path"`
	Params                            map[string]interface{}     `json:"params,omitempty"`
	CustomProcessorsSettings          map[string]json.RawMessage `json:"custom_processors"`
	CustomProcessorDefFactoryInstance sc.CustomProcessorDefFactory
}

func (ec *EnvConfig) Deserialize(jsonBytes []byte) error {
	err := json.Unmarshal(jsonBytes, ec)
	if err != nil {
		return fmt.Errorf("cannot deserialize env config: %s", err.Error())
	}

	// Defaults

	if ec.ThreadPoolSize <= 0 || ec.ThreadPoolSize > 100 {
		ec.ThreadPoolSize = 5
	}

	if ec.DeadLetterTtl < 100 || ec.DeadLetterTtl > 3600000 { // [100ms,1hr]
		ec.DeadLetterTtl = 1000
	}

	return nil
}

func ReadEnvConfigFile(envConfigFile string) (*EnvConfig, error) {
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

	envBytes, err := ioutil.ReadFile(configFullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read env config file %s: %s", configFullPath, err.Error())
	}

	var envConfig EnvConfig
	if err := envConfig.Deserialize(envBytes); err != nil {
		return nil, fmt.Errorf("cannot parse env config file %s: %s", configFullPath, err.Error())
	}

	return &envConfig, nil
}
