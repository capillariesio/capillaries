package db

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

type CassandraEngineType int

const (
	CassandraEngineNone CassandraEngineType = iota
	CassandraEngineCassandra
	CassandraEngineAmazonKeyspaces
)

const ErrorPrefixDb string = "dberror:"

func WrapDbErrorWithQuery(msg string, query string, dbErr error) error {
	if len(query) > 500 {
		query = query[:500]
	}
	return fmt.Errorf("%s, query:%s, %s%s", msg, query, ErrorPrefixDb, dbErr.Error())
}

func IsDbConnError(err error) bool {
	return strings.Contains(err.Error(), ErrorPrefixDb+gocql.ErrNoConnections.Error()) ||
		strings.Contains(err.Error(), ErrorPrefixDb+"EOF")

}

func createWfTable(cqlSession *gocql.Session, keyspace string, t reflect.Type, tableName string) error {
	q := wfmodel.GetCreateTableCql(t, keyspace, tableName)
	if err := cqlSession.Query(q).Exec(); err != nil {
		return WrapDbErrorWithQuery("failed to create WF table", q, err)
	}
	return nil
}

type CreateKeyspaceEnumType int

const DoNotCreateKeyspaceOnConnect CreateKeyspaceEnumType = 0
const CreateKeyspaceOnConnect CreateKeyspaceEnumType = 1

const CreateKeyspaceCheckAttempts int = 24
const CreateKeyspaceCheckIntervalSeconds int = 5
const DeleteKeyspaceCheckAttempts int = 24
const DeleteKeyspaceCheckIntervalSeconds int = 5
const CreateTableCheckAttempts int = 24
const CreateTableCheckIntervalSeconds int = 5

func verifyKeyspaceExists(cqlSession *gocql.Session, keyspace string) error {
	checkKsQuery := fmt.Sprintf("SELECT * FROM system_schema.keyspaces where keyspace_name='%s'", keyspace)
	for ksCheckAttempt := range CreateKeyspaceCheckAttempts {
		rows, ksCheckErr := cqlSession.Query(checkKsQuery).Iter().SliceMap()
		if ksCheckErr != nil {
			return WrapDbErrorWithQuery("failed to check keyspace exists", checkKsQuery, ksCheckErr)
		}
		if len(rows) == 1 {
			return nil
		}

		if ksCheckAttempt < CreateKeyspaceCheckAttempts-1 {
			time.Sleep(time.Duration(CreateKeyspaceCheckIntervalSeconds) * time.Second)
		}
	}
	return WrapDbErrorWithQuery("failed to check keyspace exists, giving up", checkKsQuery, errors.New("number of check attempts reached"))
}

func VerifyKeyspaceDeleted(cqlSession *gocql.Session, keyspace string) error {
	checkKsQuery := fmt.Sprintf("SELECT * FROM system_schema.keyspaces where keyspace_name='%s'", keyspace)
	for ksCheckAttempt := range DeleteKeyspaceCheckAttempts {
		rows, ksCheckErr := cqlSession.Query(checkKsQuery).Iter().SliceMap()
		if ksCheckErr != nil {
			return WrapDbErrorWithQuery("failed to check keyspace deleted", checkKsQuery, ksCheckErr)
		}
		if len(rows) == 0 {
			return nil
		}

		if ksCheckAttempt < DeleteKeyspaceCheckAttempts-1 {
			time.Sleep(time.Duration(DeleteKeyspaceCheckIntervalSeconds) * time.Second)
		}
	}
	return WrapDbErrorWithQuery("failed to check keyspace deleted, giving up", checkKsQuery, errors.New("number of check attempts reached"))
}

func checkIfAmazonKeyspaces(cqlSession *gocql.Session) (bool, error) {
	checkMcsKsQuery := "SELECT * FROM system_schema.keyspaces where keyspace_name='system_schema_mcs'"
	rows, ksCheckErr := cqlSession.Query(checkMcsKsQuery).Iter().SliceMap()
	if ksCheckErr != nil {
		return false, WrapDbErrorWithQuery("failed to check system_schema_mcs keyspace presense", checkMcsKsQuery, ksCheckErr)
	}
	if len(rows) == 0 {
		// This is not Amazon Keyspaces
		return false, nil
	}
	return true, nil
}

func VerifyAmazonKeyspacesTablesReady(cqlSession *gocql.Session, keyspace string, tableNames []string) error {
	tableCheckQuery := fmt.Sprintf("SELECT table_name, status from system_schema_mcs.tables where keyspace_name='%s'", keyspace)
	for tableCheckAttempt := range CreateTableCheckAttempts {
		rows, tableCheckErr := cqlSession.Query(tableCheckQuery).Iter().SliceMap()
		if tableCheckErr != nil {
			return WrapDbErrorWithQuery("failed to check tables", tableCheckQuery, tableCheckErr)
		}

		foundTables := map[string]struct{}{}
		if len(rows) >= len(tableNames) {
			for _, r := range rows {
				if foundTableName, ok := r["table_name"].(string); ok {
					if foundTableStatus, ok := r["status"].(string); ok {
						if foundTableStatus == "ACTIVE" {
							foundTables[foundTableName] = struct{}{}
						}
					}
				}
			}
			matchingTableCount := 0
			for _, tableName := range tableNames {
				if _, ok := foundTables[tableName]; ok {
					matchingTableCount++
				}
			}
			if matchingTableCount == len(tableNames) {
				return nil
			}
		}

		if tableCheckAttempt < CreateTableCheckAttempts-1 {
			time.Sleep(time.Duration(CreateTableCheckIntervalSeconds) * time.Second)
		}
	}
	return WrapDbErrorWithQuery("failed to check tables, giving up", tableCheckQuery, errors.New("number of check attempts reached"))
}

// func stringToCassandraConsistency(s string) (gocql.Consistency, error) {
// 	switch(s) {
// 	case "Any":
// 		return gocql.Any, nil
// 	case "One":
// 		return gocql.One, nil
// 	case "Three":
// 		return gocql.Three, nil
// 	case "Quorum":
// 		return gocql.Quorum, nil
// 	case "All":
// 		return gocql.All, nil
// 	case "LocalQuorum":
// 		return gocql.LocalQuorum, nil
// 	case "EachQuorum":
// 		return gocql.EachQuorum, nil
// 	case "LocalOne":
// 		return gocql.LocalOne, nil
// 	default:
// 		return fmt.Errorf("unknown Cassandra consistency")
// 	}
// }

func NewSession(envConfig *env.EnvConfig, keyspace string, createKeyspace CreateKeyspaceEnumType) (*gocql.Session, CassandraEngineType, error) {
	dataCluster := gocql.NewCluster(envConfig.Cassandra.Hosts...)
	dataCluster.Port = envConfig.Cassandra.Port

	// AWS Keyspaces require LocalQuorum
	// If empty, gocql sets it to Quorum by default
	if envConfig.Cassandra.Consistency != "" {
		if err := dataCluster.Consistency.UnmarshalText([]byte(envConfig.Cassandra.Consistency)); err != nil {
			return nil, CassandraEngineNone, err
		}
	}
	dataCluster.DisableInitialHostLookup = envConfig.Cassandra.DisableInitialHostLookup
	dataCluster.Authenticator = gocql.PasswordAuthenticator{Username: envConfig.Cassandra.Username, Password: envConfig.Cassandra.Password}
	dataCluster.NumConns = envConfig.Cassandra.NumConns
	dataCluster.Timeout = time.Duration(envConfig.Cassandra.Timeout * int(time.Millisecond))
	dataCluster.ConnectTimeout = time.Duration(envConfig.Cassandra.ConnectTimeout * int(time.Millisecond))
	// Token-aware policy should give better perf results when used together with prepared queries, and Capillaries chatty inserts are killing Cassandra.
	// TODO: consider making it configurable
	dataCluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	// When testing, we load Cassandra cluster at 100%. There will be "Operation timed out - received only 0 responses" errors.
	// It's up to admins how to handle the load, but we should not give up quickly in any case. Make it 3 attempts.
	dataCluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}
	if envConfig.Cassandra.SslOpts != nil &&
		(envConfig.Cassandra.SslOpts.EnableHostVerification || len(envConfig.Cassandra.SslOpts.CaPath) > 0 || len(envConfig.Cassandra.SslOpts.CertPath) > 0 || len(envConfig.Cassandra.SslOpts.KeyPath) > 0) {
		dataCluster.SslOpts = &gocql.SslOptions{
			EnableHostVerification: envConfig.Cassandra.SslOpts.EnableHostVerification,
			CaPath:                 envConfig.Cassandra.SslOpts.CaPath,
			CertPath:               envConfig.Cassandra.SslOpts.CertPath,
			KeyPath:                envConfig.Cassandra.SslOpts.KeyPath}
	}
	cqlSession, err := dataCluster.CreateSession()
	if err != nil {
		return nil, CassandraEngineNone, fmt.Errorf("failed to connect to data cluster %v, keyspace [%s]: %s", envConfig.Cassandra.Hosts, keyspace, err.Error())
	}

	cassandraEngine := CassandraEngineNone
	if isAmazonKeyspaces, err := checkIfAmazonKeyspaces(cqlSession); err == nil {
		if isAmazonKeyspaces {
			cassandraEngine = CassandraEngineAmazonKeyspaces
		} else {
			cassandraEngine = CassandraEngineCassandra
		}
	} else {
		return nil, cassandraEngine, err
	}

	// Create keyspace if needed
	if len(keyspace) > 0 {
		dataCluster.Keyspace = keyspace

		if createKeyspace == CreateKeyspaceOnConnect {
			createKsQuery := fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = %s", keyspace, envConfig.Cassandra.KeyspaceReplicationConfig)
			if err := cqlSession.Query(createKsQuery).Exec(); err != nil {
				return nil, cassandraEngine, WrapDbErrorWithQuery("failed to create keyspace", createKsQuery, err)
			}

			if cassandraEngine == CassandraEngineAmazonKeyspaces {
				if checkKeyspaceErr := verifyKeyspaceExists(cqlSession, keyspace); checkKeyspaceErr != nil {
					return nil, cassandraEngine, checkKeyspaceErr
				}
			}

			if err := cqlSession.Query(createKsQuery).Exec(); err != nil {
				return nil, cassandraEngine, WrapDbErrorWithQuery("failed to create keyspace", createKsQuery, err)
			}

			// Create WF tables if needed
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.BatchHistoryEvent{}), wfmodel.TableNameBatchHistory); err != nil {
				return nil, cassandraEngine, err
			}
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.NodeHistoryEvent{}), wfmodel.TableNameNodeHistory); err != nil {
				return nil, cassandraEngine, err
			}
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.RunHistoryEvent{}), wfmodel.TableNameRunHistory); err != nil {
				return nil, cassandraEngine, err
			}
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.RunProperties{}), wfmodel.TableNameRunAffectedNodes); err != nil {
				return nil, cassandraEngine, err
			}
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.RunCounter{}), wfmodel.TableNameRunCounter); err != nil {
				return nil, cassandraEngine, err
			}

			if cassandraEngine == CassandraEngineAmazonKeyspaces {
				if checkTableErr := VerifyAmazonKeyspacesTablesReady(cqlSession, keyspace, []string{
					wfmodel.TableNameBatchHistory,
					wfmodel.TableNameNodeHistory,
					wfmodel.TableNameRunHistory,
					wfmodel.TableNameRunAffectedNodes,
					wfmodel.TableNameRunCounter}); checkTableErr != nil {
					return nil, cassandraEngine, checkTableErr
				}
			}

			qb := cql.QueryBuilder{}
			qb.
				Keyspace(keyspace).
				Write("ks", keyspace).
				Write("last_run", 0)
			q := qb.InsertUnpreparedQuery(wfmodel.TableNameRunCounter, cql.IgnoreIfExists) // If not exists. Insert only once.
			err = cqlSession.Query(q).Exec()
			if err != nil {
				return nil, cassandraEngine, WrapDbErrorWithQuery("cannot initialize run counter", q, err)
			}
		}
	}
	return cqlSession, cassandraEngine, nil
}
