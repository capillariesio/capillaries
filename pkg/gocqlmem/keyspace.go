package gocqlmem

import (
	"fmt"
	"sync"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

type Keyspace struct {
	TableMap        map[string]*tableStore
	WithReplication []*KeyValuePair
	Lock            sync.RWMutex
}

func newKeyspace() *Keyspace {
	return &Keyspace{
		TableMap:        map[string]*tableStore{},
		WithReplication: make([]*KeyValuePair, 0),
	}
}

func (ks *Keyspace) createTable(cmd *CommandCreateTable) error {
	ks.Lock.Lock()
	defer ks.Lock.Unlock()

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
	return nil
}

func (ks *Keyspace) truncateTable(cmd *CommandTruncateTable) error {
	ks.Lock.RLock()
	defer ks.Lock.RUnlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists {
		return fmt.Errorf("cannot truncate table %s, it was not found", cmd.TableName)
	}

	return t.execTruncate(cmd)
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

func (ks *Keyspace) execSelect(cmd *CommandSelect, lastSelectedRowIdx int, maxRows int) ([]string, [][]any, []gocql.TypeInfo, int, error) {
	ks.Lock.RLock()
	defer ks.Lock.RUnlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists {
		return []string{}, [][]any{}, []gocql.TypeInfo{}, -1, fmt.Errorf("cannot select from  table %s, it was not found in the keyspace %s", cmd.TableName, cmd.GetCtxKeyspace())
	}
	return t.execSelect(cmd, lastSelectedRowIdx, maxRows)
}

func (ks *Keyspace) execUpdate(cmd *CommandUpdate) (bool, []gocql.ColumnInfo, [][]any, error) {
	ks.Lock.RLock()
	defer ks.Lock.RUnlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists {
		return false, nil, nil, fmt.Errorf("cannot update table %s, it was not found in the keyspace %s", cmd.TableName, cmd.GetCtxKeyspace())
	}
	return t.execUpdate(cmd)
}

func (ks *Keyspace) execDelete(cmd *CommandDelete) (bool, error) {
	ks.Lock.RLock()
	defer ks.Lock.RUnlock()

	t, alreadyExists := ks.TableMap[cmd.TableName]
	if !alreadyExists {
		return false, fmt.Errorf("cannot delete from table %s, it was not found in the keyspace %s", cmd.TableName, cmd.GetCtxKeyspace())
	}

	return t.execDelete(cmd)
}
