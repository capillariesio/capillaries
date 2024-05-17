package env

import "encoding/json"

// This was not tested outside of the EnableHostVerification=false scenario
type SslOptions struct {
	CertPath               string `json:"cert_path" env:"CAPI_CASSANDRA_CERT_PATH, overwrite"`
	KeyPath                string `json:"key_path" env:"CAPI_CASSANDRA_KEY_PATH, overwrite"`
	CaPath                 string `json:"ca_path" env:"CAPI_CASSANDRA_CA_PATH, overwrite"`
	EnableHostVerification bool   `json:"enable_host_verification" env:"CAPI_CASSANDRA_ENABLE_HOST_VERIFICATION, overwrite"`
}

type CassandraConfig struct {
	Hosts                     []string    `json:"hosts" env:"CAPI_CASSANDRA_HOSTS, overwrite"`
	Port                      int         `json:"port" env:"CAPI_CASSANDRA_PORT, overwrite"`
	Username                  string      `json:"username" env:"CAPI_CASSANDRA_USERNAME, overwrite"`
	Password                  string      `json:"password" env:"CAPI_CASSANDRA_PASSWORD, overwrite"`
	WriterWorkers             int         `json:"writer_workers" env:"CAPI_CASSANDRA_WRITER_WORKERS, overwrite"`                           // 20 is conservative, 80 is very aggressive
	MinInserterRate           int         `json:"min_inserter_rate" env:"CAPI_CASSANDRA_MIN_INSERTER_RATE, overwrite"`                     // writes/sec; if the rate falls below this, we consider the db too slow and throw an error
	NumConns                  int         `json:"num_conns" env:"CAPI_CASSANDRA_NUM_CONNS, overwrite"`                                     // gocql default is 2, don't make it too high
	Timeout                   int         `json:"timeout" env:"CAPI_CASSANDRA_TIMEOUT, overwrite"`                                         // in ms, set it to 5s, gocql default 600ms is way too aggressive for heavy writes by multiple workers
	ConnectTimeout            int         `json:"connect_timeout" env:"CAPI_CASSANDRA_CONNECT_TIMEOUT, overwrite"`                         // in ms, set it to 1s, gocql default 600ms may be ok, but let's stay on the safe side
	KeyspaceReplicationConfig string      `json:"keyspace_replication_config" env:"CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG, overwrite"` // { 'class' : 'NetworkTopologyStrategy', 'datacenter1' : 1 }
	SslOpts                   *SslOptions `json:"ssl_opts"`
}

func (c *CassandraConfig) ShallowCopy() CassandraConfig {
	return CassandraConfig{
		Hosts:                     c.Hosts,
		Port:                      c.Port,
		Username:                  c.Username,
		Password:                  c.Password,
		WriterWorkers:             c.WriterWorkers,
		MinInserterRate:           c.MinInserterRate,
		NumConns:                  c.NumConns,
		Timeout:                   c.Timeout,
		ConnectTimeout:            c.ConnectTimeout,
		KeyspaceReplicationConfig: c.KeyspaceReplicationConfig,
		SslOpts:                   c.SslOpts,
	}
}
func (c *CassandraConfig) MarshalJSON() ([]byte, error) {
	safeCopy := c.ShallowCopy()
	if len(safeCopy.Username) > 0 {
		safeCopy.Username = "..."
	}
	if len(safeCopy.Password) > 0 {
		safeCopy.Password = "..."
	}
	return json.Marshal(safeCopy)
}
