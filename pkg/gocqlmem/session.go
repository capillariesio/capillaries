package gocqlmem

import (
	"context"
	"fmt"
	"sync"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

type gocqlmemSession struct {
	keyspaceMap map[string]*Keyspace
	lock        sync.RWMutex
	isClosed    bool
}

func NewGocqlmemSession() Session {
	return &gocqlmemSession{
		keyspaceMap: map[string]*Keyspace{},
	}
}

func (s *gocqlmemSession) createKeyspace(cmd *CommandCreateKeyspace) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, alreadyExists := s.keyspaceMap[cmd.KeyspaceName]
	if alreadyExists && cmd.IfNotExists {
		return nil
	}
	if alreadyExists && !cmd.IfNotExists {
		return fmt.Errorf("cannot create keyspace %s, it already exists and no IF NOT EXISTS were specified", cmd.KeyspaceName)
	}
	s.keyspaceMap[cmd.KeyspaceName] = newKeyspace()
	return nil
}

func (s *gocqlmemSession) dropKeyspace(cmd *CommandDropKeyspace) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	ks, alreadyExists := s.keyspaceMap[cmd.KeyspaceName]
	if !alreadyExists && cmd.IfExists {
		return nil
	}
	if !alreadyExists && !cmd.IfExists {
		return fmt.Errorf("cannot drop keyspace %s, it was not found and no IF EXISTS were specified", cmd.KeyspaceName)
	}

	ks.Lock.Lock()
	delete(s.keyspaceMap, cmd.KeyspaceName)
	ks.Lock.Unlock()

	return nil
}

func (s *gocqlmemSession) createTable(cmd *CommandCreateTable) error {
	s.lock.RLock()

	ks, ksExists := s.keyspaceMap[cmd.GetCtxKeyspace()]
	s.lock.RUnlock()
	if !ksExists {
		return fmt.Errorf("keyspace %s does not exist", cmd.GetCtxKeyspace())
	}

	return ks.createTable(cmd)
}

func (s *gocqlmemSession) truncateTable(cmd *CommandTruncateTable) error {
	s.lock.RLock()

	ks, ksExists := s.keyspaceMap[cmd.GetCtxKeyspace()]
	s.lock.RUnlock()
	if !ksExists {
		return fmt.Errorf("keyspace %s does not exist", cmd.GetCtxKeyspace())
	}

	return ks.truncateTable(cmd)
}

func (s *gocqlmemSession) dropTable(cmd *CommandDropTable) error {
	s.lock.RLock()

	ks, ksExists := s.keyspaceMap[cmd.GetCtxKeyspace()]
	s.lock.RUnlock()
	if !ksExists {
		return fmt.Errorf("keyspace %s does not exist", cmd.GetCtxKeyspace())
	}

	return ks.dropTable(cmd)
}

func (s *gocqlmemSession) execInsert(cmd *CommandInsert) (bool, []gocql.ColumnInfo, [][]any, error) {
	s.lock.RLock()

	ks, ksExists := s.keyspaceMap[cmd.GetCtxKeyspace()]
	s.lock.RUnlock()
	if !ksExists {
		return false, nil, nil, fmt.Errorf("keyspace %s does not exist", cmd.GetCtxKeyspace())
	}

	return ks.execInsert(cmd)
}

func (s *gocqlmemSession) execSelect(cmd *CommandSelect, lastSelectedRowIdx int, maxRows int, preparedQueryParams []interface{}) ([]string, [][]any, []gocql.TypeInfo, int, error) {
	s.lock.RLock()

	ks, ksExists := s.keyspaceMap[cmd.GetCtxKeyspace()]
	s.lock.RUnlock()
	if !ksExists {
		return []string{}, [][]any{}, []gocql.TypeInfo{}, -1, fmt.Errorf("keyspace %s does not exist", cmd.GetCtxKeyspace())
	}

	return ks.execSelect(cmd, lastSelectedRowIdx, maxRows, preparedQueryParams)
}

func (s *gocqlmemSession) execUpdate(cmd *CommandUpdate, preparedQueryParams []interface{}) (bool, []gocql.ColumnInfo, [][]any, error) {
	s.lock.RLock()

	ks, ksExists := s.keyspaceMap[cmd.GetCtxKeyspace()]
	s.lock.RUnlock()
	if !ksExists {
		return false, nil, nil, fmt.Errorf("keyspace %s does not exist", cmd.GetCtxKeyspace())
	}

	return ks.execUpdate(cmd, preparedQueryParams)
}

func (s *gocqlmemSession) execDelete(cmd *CommandDelete, preparedQueryParams []interface{}) (bool, error) {
	s.lock.RLock()

	ks, ksExists := s.keyspaceMap[cmd.GetCtxKeyspace()]
	s.lock.RUnlock()
	if !ksExists {
		return false, fmt.Errorf("keyspace %s does not exist", cmd.GetCtxKeyspace())
	}

	return ks.execDelete(cmd, preparedQueryParams)
}

// Session interface

func (s *gocqlmemSession) AwaitSchemaAgreement(ctx context.Context) error {
	return nil
}

func (s *gocqlmemSession) Query(stmt string, values ...interface{}) Query {
	return &gocqlmemQuery{
		session: s,
		stmt:    stmt,
		values:  values,
	}
}
func (s *gocqlmemSession) Bind(stmt string, b func(q *gocql.QueryInfo) ([]interface{}, error)) Query {
	// TODO: implement
	return nil
}
func (s *gocqlmemSession) Close() {
	s.isClosed = true
}
func (s *gocqlmemSession) Closed() bool {
	return s.isClosed
}
func (s *gocqlmemSession) KeyspaceMetadata(keyspace string) (*gocql.KeyspaceMetadata, error) {
	// TODO: implement
	return nil, fmt.Errorf("not implemented")
}
func (s *gocqlmemSession) Batch(typ gocql.BatchType) *gocql.Batch {
	// TODO: implement
	return nil
}
func (s *gocqlmemSession) GetHosts() []*gocql.HostInfo {
	return []*gocql.HostInfo{}
}
