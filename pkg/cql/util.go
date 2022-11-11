package cql

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
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

func NewSession(envConfig *env.EnvConfig, keyspace string, createKeyspace CreateKeyspaceEnumType) (*gocql.Session, error) {
	dataCluster := gocql.NewCluster(envConfig.Cassandra.Hosts...)
	dataCluster.Port = envConfig.Cassandra.Port
	dataCluster.Authenticator = gocql.PasswordAuthenticator{Username: envConfig.Cassandra.Username, Password: envConfig.Cassandra.Password}
	dataCluster.NumConns = envConfig.Cassandra.NumConns
	dataCluster.Timeout = time.Duration(envConfig.Cassandra.Timeout * int(time.Millisecond))
	dataCluster.ConnectTimeout = time.Duration(envConfig.Cassandra.ConnectTimeout * int(time.Millisecond))
	if envConfig.Cassandra.SslOpts != nil {
		dataCluster.SslOpts = &gocql.SslOptions{
			EnableHostVerification: envConfig.Cassandra.SslOpts.EnableHostVerification,
			CaPath:                 envConfig.Cassandra.SslOpts.CaPath,
			CertPath:               envConfig.Cassandra.SslOpts.CaPath,
			KeyPath:                envConfig.Cassandra.SslOpts.KeyPath}
	}
	cqlSession, err := dataCluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to data cluster %v, keyspace [%s]: %s", envConfig.Cassandra.Hosts, keyspace, err.Error())
	}
	// Create keyspace if needed
	if len(keyspace) > 0 {
		dataCluster.Keyspace = keyspace

		if createKeyspace == CreateKeyspaceOnConnect {
			createKsQuery := fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = %s", keyspace, envConfig.Cassandra.KeyspaceReplicationConfig)
			if err := cqlSession.Query(createKsQuery).Exec(); err != nil {
				return nil, WrapDbErrorWithQuery("failed to create keyspace", createKsQuery, err)
			}

			// Create WF tables if needed
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.BatchHistoryEvent{}), wfmodel.TableNameBatchHistory); err != nil {
				return nil, err
			}
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.NodeHistoryEvent{}), wfmodel.TableNameNodeHistory); err != nil {
				return nil, err
			}
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.RunHistoryEvent{}), wfmodel.TableNameRunHistory); err != nil {
				return nil, err
			}
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.RunProperties{}), wfmodel.TableNameRunAffectedNodes); err != nil {
				return nil, err
			}
			if err = createWfTable(cqlSession, keyspace, reflect.TypeOf(wfmodel.RunCounter{}), wfmodel.TableNameRunCounter); err != nil {
				return nil, err
			}

			qb := QueryBuilder{}
			qb.
				Keyspace(keyspace).
				Write("ks", keyspace).
				Write("last_run", 0)
			q := qb.Insert(wfmodel.TableNameRunCounter, IgnoreIfExists) // If not exists. Insert only once.
			err = cqlSession.Query(q).Exec()
			if err != nil {
				return nil, WrapDbErrorWithQuery("cannot initialize run counter", q, err)
			}
		}
	}
	return cqlSession, nil
}
