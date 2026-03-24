package gocqlmem

import (
	"fmt"
	"sync"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

type Keyspace struct {
	TableMap         map[string]*tableStore
	WithReplication  []*KeyValuePair
	Lock             sync.RWMutex
	TableMetadataMap map[string]*gocql.TableMetadata
}

func newKeyspace() *Keyspace {
	return &Keyspace{
		TableMap:         map[string]*tableStore{},
		WithReplication:  make([]*KeyValuePair, 0),
		TableMetadataMap: map[string]*gocql.TableMetadata{},
	}
}

func (ks *Keyspace) createTable(cmd *CommandCreateTable) error {
	ks.Lock.Lock()
	defer ks.Lock.Unlock()

	// Table store
	_, alreadyExists := ks.TableMap[cmd.TableName]
	if alreadyExists && cmd.IfNotExists {
		return nil
	}
	if alreadyExists && !cmd.IfNotExists {
		return fmt.Errorf("cannot create table %s, it already exists and no IF NOT EXISTS were specified", cmd.TableName)
	}
	newTable, err := newTable(cmd)
	if err != nil {
		return fmt.Errorf("cannot create table %s: %s", cmd.TableName, err.Error())
	}
	ks.TableMap[cmd.TableName] = newTable

	// Table metadata
	ks.TableMetadataMap[cmd.TableName] = &gocql.TableMetadata{
		Keyspace: cmd.GetCtxKeyspace(),
		Name:     cmd.TableName,
	}

	return nil
}

func (ks *Keyspace) truncateTable(cmd *CommandTruncateTable) error {
	ks.Lock.RLock()
	defer ks.Lock.RUnlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists {
		return fmt.Errorf("cannot truncate table %s, it was not found", cmd.TableName)
	}

	return t.execTruncate()
}

func (ks *Keyspace) dropTable(cmd *CommandDropTable) error {
	ks.Lock.Lock()
	defer ks.Lock.Unlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists && cmd.IfExists {
		return nil
	}
	if !alreadyExists && !cmd.IfExists {
		return fmt.Errorf("cannot drop table %s, it was not found and no IF EXISTS were specified", cmd.TableName)
	}

	t.lock.Lock()
	delete(ks.TableMap, cmd.TableName)
	t.lock.Unlock()

	return nil
}

func (ks *Keyspace) execInsert(cmd *CommandInsert) (bool, []gocql.ColumnInfo, [][]any, error) {
	ks.Lock.RLock()
	defer ks.Lock.RUnlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists {
		return false, nil, nil, fmt.Errorf("cannot insert into table %s, it was not found in the keyspace %s", cmd.TableName, cmd.GetCtxKeyspace())
	}
	return t.execInsert(cmd)
}

func (ks *Keyspace) execSelect(cmd *CommandSelect, lastSelectedRowIdx int, maxRows int, preparedQueryParams []any) ([]string, [][]any, []gocql.TypeInfo, int, error) {
	ks.Lock.RLock()
	defer ks.Lock.RUnlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists {
		return []string{}, [][]any{}, []gocql.TypeInfo{}, -1, fmt.Errorf("cannot select from  table %s, it was not found in the keyspace %s", cmd.TableName, cmd.GetCtxKeyspace())
	}
	return t.execSelect(cmd, lastSelectedRowIdx, maxRows, preparedQueryParams)
}

func (ks *Keyspace) execUpdate(cmd *CommandUpdate, preparedQueryParams []any) (bool, []gocql.ColumnInfo, [][]any, error) {
	ks.Lock.RLock()
	defer ks.Lock.RUnlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists {
		return false, nil, nil, fmt.Errorf("cannot update table %s, it was not found in the keyspace %s", cmd.TableName, cmd.GetCtxKeyspace())
	}
	return t.execUpdate(cmd, preparedQueryParams)
}

func (ks *Keyspace) execDelete(cmd *CommandDelete, preparedQueryParams []any) (bool, error) {
	ks.Lock.RLock()
	defer ks.Lock.RUnlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists {
		return false, fmt.Errorf("cannot delete from table %s, it was not found in the keyspace %s", cmd.TableName, cmd.GetCtxKeyspace())
	}

	return t.execDelete(cmd, preparedQueryParams)
}
