package env

// This was not tested outside of the EnableHostVerification=false scenario
type SslOptions struct {
	CertPath               string `json:"cert_path"`
	KeyPath                string `json:"key_path"`
	CaPath                 string `json:"ca_path"`
	EnableHostVerification bool   `json:"enable_host_verification"`
}

type CassandraConfig struct {
	Hosts                     []string    `json:"hosts"`
	Port                      int         `json:"port"`
	Username                  string      `json:"username"`
	Password                  string      `json:"password"`
	WriterWorkers             int         `json:"writer_workers"`              // 20 is conservative, 80 is very aggressive
	MinInserterRate           int         `json:"min_inserter_rate"`           // writes/sec; if the rate falls below this, we consider the db too slow and throw an error
	NumConns                  int         `json:"num_conns"`                   // gocql default is 2, don't make it too high
	Timeout                   int         `json:"timeout"`                     // in ms, set it to 5s, gocql default 600ms is way too aggressive for heavy writes by multiple workers
	ConnectTimeout            int         `json:"connect_timeout"`             // in ms, set it to 1s, gocql default 600ms may be ok, but let's stay on the safe side
	KeyspaceReplicationConfig string      `json:"keyspace_replication_config"` // { 'class' : 'NetworkTopologyStrategy', 'datacenter1' : 1 }
	SslOpts                   *SslOptions `json:"ssl_opts"`
}
