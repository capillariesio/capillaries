package gocqlmem

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

type gocqlmemQuery struct {
	stmt    string
	values  []any // Where do we use these?
	session *gocqlmemSession
	// disableAutoPage     bool
	// disableSkipMetadata bool

	// Required by Query interface
	consistency gocql.Consistency
	idempotent  bool
	hostId      string
	keyspace    string
	pageSize    int
	pageState   []byte
}

func addAppliedToRetrievedData(ks string, tableName string, existingColumnInfos []gocql.ColumnInfo, existingValues [][]any, isApplied bool) ([]gocql.ColumnInfo, [][]any) {
	if existingColumnInfos == nil {
		existingColumnInfos = []gocql.ColumnInfo{}
	}
	existingColumnInfos = append(existingColumnInfos, gocql.ColumnInfo{
		Keyspace: ks,
		Table:    tableName,
		Name:     "[applied]",
		TypeInfo: newScalarType(gocql.TypeBoolean),
	})

	if existingValues == nil {
		existingValues = [][]any{}
	}
	if len(existingValues) == 0 {
		existingValues = append(existingValues, []any{})
	}
	existingValues[0] = append(existingValues[0], isApplied)
	return existingColumnInfos, existingValues
}

func adjustInternalValuesToClientTypesAccordingToTypeInfos(values [][]any, typeInfos []gocql.TypeInfo) error {
	for rowIdx := range len(values) {
		for colIdx := range len(typeInfos) {
			if typeInfos[colIdx] != nil {
				var err error
				if values[rowIdx][colIdx], err = internalValueToClientType(values[rowIdx][colIdx], typeInfos[colIdx].Type()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func adjustInternalValuesToClientTypesAccordingToColumnInfos(values [][]any, columnInfos []gocql.ColumnInfo) error {
	for rowIdx := range len(values) {
		for colIdx := range len(columnInfos) {
			if columnInfos[colIdx].TypeInfo != nil {
				var err error
				if values[rowIdx][colIdx], err = internalValueToClientType(values[rowIdx][colIdx], columnInfos[colIdx].TypeInfo.Type()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Query interface

// Why does Query have to mimic Iter's Scan, MapScan, ScanCAS???

func (q *gocqlmemQuery) Consistency(c gocql.Consistency) Query {
	q.consistency = c
	return q
}

func (q *gocqlmemQuery) GetConsistency() gocql.Consistency {
	return q.consistency
}

func (q *gocqlmemQuery) CustomPayload(_ map[string][]byte) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) Trace(_ gocql.Tracer) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) Observer(_ gocql.QueryObserver) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) PageSize(n int) Query {
	q.pageSize = n
	return q
}

func (q *gocqlmemQuery) DefaultTimestamp(_ bool) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) WithTimestamp(_ int64) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) RoutingKey(_ []byte) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) Keyspace() string {
	return q.keyspace
}
func (q *gocqlmemQuery) Prefetch(_ float64) Query {
	// TODO: implement
	return q
}
func (q *gocqlmemQuery) RetryPolicy(_ gocql.RetryPolicy) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) SetSpeculativeExecutionPolicy(_ gocql.SpeculativeExecutionPolicy) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) IsIdempotent() bool {
	return q.idempotent
}

func (q *gocqlmemQuery) Idempotent(value bool) Query {
	q.idempotent = value
	return q
}
func (q *gocqlmemQuery) Bind(_ ...any) Query {
	// TODO: implement
	return q
}
func (q *gocqlmemQuery) SerialConsistency(_ gocql.Consistency) Query {
	// TODO: implement
	return q
}
func (q *gocqlmemQuery) PageState(state []byte) Query {
	q.pageState = state
	return q
}

func (q *gocqlmemQuery) NoSkipMetadata() Query {
	// TODO: implement
	return q
}

// CREATE KEYSPACE
func (q *gocqlmemQuery) Exec() error {
	return q.Iter().Close()
}

func (q *gocqlmemQuery) ExecContext(_ context.Context) error {
	// TODO: implement
	return errors.New("not implemented")
}

func (q *gocqlmemQuery) Iter() Iter {
	cmds, err := ParseCommands(q.stmt, q.values)
	if err != nil {
		return newGocqlmemIterWithError(err)
	}
	if len(cmds) != 1 {
		return newGocqlmemIterWithError(fmt.Errorf("exactly one CQL cmd expected, got: %s", q.stmt))
	}

	switch cmd := cmds[0].(type) {
	case *CommandCreateKeyspace:
		return newGocqlmemIterWithError(q.session.createKeyspace(cmd))
	case *CommandUseKeyspace:
		return newGocqlmemIterWithKeyspace(cmd.KeyspaceName)
	case *CommandDropKeyspace:
		return newGocqlmemIterWithError(q.session.dropKeyspace(cmd))
	case *CommandCreateTable:
		return newGocqlmemIterWithError(q.session.createTable(cmd))
	case *CommandTruncateTable:
		return newGocqlmemIterWithError(q.session.truncateTable(cmd))
	case *CommandDropTable:
		return newGocqlmemIterWithError(q.session.dropTable(cmd))
	case *CommandInsert:
		isApplied, existingColumnInfos, existingValues, err := q.session.execInsert(cmd)
		if err != nil {
			return newGocqlmemIterWithError(err)
		}

		if err = adjustInternalValuesToClientTypesAccordingToColumnInfos(existingValues, existingColumnInfos); err != nil {
			return newGocqlmemIterWithError(err)
		}

		existingColumnInfos, existingValues = addAppliedToRetrievedData(cmd.GetCtxKeyspace(), cmd.TableName, existingColumnInfos, existingValues, isApplied)
		return newGocqlmemIterWithData(cmd.GetCtxKeyspace(), cmd.TableName, existingColumnInfos, existingValues)

	case *CommandSelect:
		var lastSelectedRowIdx int32
		if len(q.pageState) == 0 {
			lastSelectedRowIdx = -1
		} else {
			// This is our implementation of pagestate: we store the idx of the last selected row idx
			err := binary.Read(bytes.NewReader(q.pageState), binary.LittleEndian, &lastSelectedRowIdx)
			if err != nil {
				return newGocqlmemIterWithError(fmt.Errorf("cannot convert page state %v to int: %s", q.pageState, err.Error()))
			}
		}
		names, values, typeInfos, newLastSelectedRowIdx, err := q.session.execSelect(cmd, int(lastSelectedRowIdx), q.pageSize, q.values)
		if err != nil {
			return newGocqlmemIterWithError(err)
		}

		if err = adjustInternalValuesToClientTypesAccordingToTypeInfos(values, typeInfos); err != nil {
			return newGocqlmemIterWithError(err)
		}

		// This is our implementation of pagestate: we store the idx of the last selected row idx
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.LittleEndian, int32(newLastSelectedRowIdx))
		if err != nil {
			return newGocqlmemIterWithError(fmt.Errorf("cannot convert int %d to byte slice: %s", lastSelectedRowIdx, err.Error()))
		}

		return newGocqlmemIterWithDataAndPagingState(cmd.GetCtxKeyspace(), cmd.TableName,
			namesAndTypeInfosTocolumnInfos(cmd.GetCtxKeyspace(), cmd.TableName, names, typeInfos),
			values,
			buf.Bytes())

	case *CommandUpdate:
		isApplied, existingColumnInfos, existingValues, err := q.session.execUpdate(cmd, q.values)
		if err != nil {
			return newGocqlmemIterWithError(err)
		}

		if err = adjustInternalValuesToClientTypesAccordingToColumnInfos(existingValues, existingColumnInfos); err != nil {
			return newGocqlmemIterWithError(err)
		}

		existingColumnInfos, existingValues = addAppliedToRetrievedData(cmd.GetCtxKeyspace(), cmd.TableName, existingColumnInfos, existingValues, isApplied)
		return newGocqlmemIterWithData(cmd.GetCtxKeyspace(), cmd.TableName, existingColumnInfos, existingValues)

	case *CommandDelete:
		isApplied, err := q.session.execDelete(cmd, q.values)
		if err != nil {
			return newGocqlmemIterWithError(err)
		}
		return newGocqlmemIterWithData(cmd.GetCtxKeyspace(), "",
			[]gocql.ColumnInfo{
				{
					Keyspace: cmd.GetCtxKeyspace(),
					Table:    cmd.TableName,
					Name:     "[applied]",
					TypeInfo: newScalarType(gocql.TypeBoolean),
				}},
			[][]any{{isApplied}})

	default:
		return newGocqlmemIterWithError(fmt.Errorf("Iter() does not support cmd %v", cmd))
	}
}

func (q *gocqlmemQuery) IterContext(_ context.Context) Iter {
	// TODO: implement
	return q.Iter()
}

func (q *gocqlmemQuery) MapScan(m map[string]any) error {
	iter := q.Iter()
	if err := iter.Err(); err != nil {
		return err
	}
	if !iter.MapScan(m) {
		return iter.Err()
	}
	return iter.Close()
}

func (q *gocqlmemQuery) MapScanContext(_ context.Context, m map[string]any) error {
	// TODO: implement
	return q.MapScan(m)
}

func (q *gocqlmemQuery) Scan(dest ...any) error {
	iter := q.Iter()
	if err := iter.Err(); err != nil {
		return err
	}
	if iter.NumRows() == 0 {
		return gocql.ErrNotFound
	}
	if !iter.Scan(dest) {
		return iter.Err()
	}
	return iter.Close()
}

func (q *gocqlmemQuery) ScanContext(_ context.Context, dest ...any) error {
	// TODO: implement
	return q.Scan(dest)
}

func (q *gocqlmemQuery) ScanCAS(dest ...any) (applied bool, err error) {
	iter := q.Iter()
	if err := iter.Err(); err != nil {
		return false, err
	}
	if iter.NumRows() == 0 {
		return false, gocql.ErrNotFound
	}
	if len(iter.Columns()) > 1 {
		dest = append([]any{&applied}, dest...)
		iter.Scan(dest...)
	} else {
		iter.Scan(&applied)
	}
	return applied, iter.Close()
}

func (q *gocqlmemQuery) ScanCASContext(_ context.Context, dest ...any) (applied bool, err error) {
	// TODO: implement
	return q.ScanCAS(dest)
}

// INSERT, UPDATE, DELETE
func (q *gocqlmemQuery) MapScanCAS(dest map[string]any) (bool, error) {
	iter := q.Iter()
	if err := iter.Err(); err != nil {
		return false, err
	}
	if iter.NumRows() == 0 {
		return false, gocql.ErrNotFound
	}
	if !iter.MapScan(dest) {
		return false, iter.Err()
	}
	applied, ok := dest["[applied]"].(bool)
	if !ok {
		return false, errors.New("cannot read bool [applied]")
	}
	delete(dest, "[applied]")

	return applied, iter.Close()
}

func (q *gocqlmemQuery) MapScanCASContext(_ context.Context, dest map[string]any) (applied bool, err error) {
	// TODO: implement
	return q.MapScanCAS(dest)
}

func (q *gocqlmemQuery) SetHostID(hostID string) Query {
	q.hostId = hostID
	return q
}

func (q *gocqlmemQuery) GetHostID() string {
	return q.hostId
}

func (q *gocqlmemQuery) SetKeyspace(keyspace string) Query {
	q.keyspace = keyspace
	return q
}
func (q *gocqlmemQuery) WithNowInSeconds(_ int) Query {
	// TODO: implement
	return q
}
