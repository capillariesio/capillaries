package gocqlmem

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/shopspring/decimal"
	"gopkg.in/inf.v0"
)

type gocqlmemQuery struct {
	stmt    string
	values  []interface{} // Where do we use these?
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

func internalValueToClientType(val any, typ gocql.Type) (any, error) {
	switch typedInternalVal := val.(type) {
	case int64:
		switch typ {
		case gocql.TypeTinyInt:
			return int8(typedInternalVal), nil
		case gocql.TypeSmallInt:
			return int16(typedInternalVal), nil
		case gocql.TypeInt, gocql.TypeDate:
			return int32(typedInternalVal), nil
		case gocql.TypeBigInt, gocql.TypeVarint, gocql.TypeCounter, gocql.TypeTime:
			return typedInternalVal, nil
		}

	case float64:
		switch typ {
		case gocql.TypeFloat:
			return float32(typedInternalVal), nil
		case gocql.TypeDouble:
			return typedInternalVal, nil
		}

	case decimal.Decimal:
		switch typ {
		case gocql.TypeDecimal:
			s := typedInternalVal.String()
			infDecVal, ok := new(inf.Dec).SetString(s)
			if !ok {
				return nil, fmt.Errorf("cannot convert decimal %v(%T) to inf.Dec from string %s", typedInternalVal, typedInternalVal, s)
			}
			return *infDecVal, nil
		}
	case []byte:
		switch typ {
		case gocql.TypeBlob:
			return typedInternalVal, nil
		case gocql.TypeUUID, gocql.TypeTimeUUID:
			uuid, err := gocql.UUIDFromBytes(typedInternalVal)
			if err != nil {
				return nil, fmt.Errorf("cannot []byte %v(%T) to UUID/TimeUUID: %s", typedInternalVal, typedInternalVal, err.Error())
			}
			return uuid, nil
		}

	}

	// Give up and pray
	return val, nil
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

func (q *gocqlmemQuery) CustomPayload(customPayload map[string][]byte) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) Trace(trace gocql.Tracer) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) Observer(observer gocql.QueryObserver) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) PageSize(n int) Query {
	q.pageSize = n
	return q
}

func (q *gocqlmemQuery) DefaultTimestamp(enable bool) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) WithTimestamp(timestamp int64) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) RoutingKey(routingKey []byte) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) Keyspace() string {
	return q.keyspace
}
func (q *gocqlmemQuery) Prefetch(p float64) Query {
	// TODO: implement
	return q
}
func (q *gocqlmemQuery) RetryPolicy(r gocql.RetryPolicy) Query {
	// TODO: implement
	return q
}

func (q *gocqlmemQuery) SetSpeculativeExecutionPolicy(sp gocql.SpeculativeExecutionPolicy) Query {
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
func (q *gocqlmemQuery) Bind(v ...interface{}) Query {
	// TODO: implement
	return q
}
func (q *gocqlmemQuery) SerialConsistency(cons gocql.Consistency) Query {
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

func (q *gocqlmemQuery) ExecContext(ctx context.Context) error {
	// TODO: implement
	return fmt.Errorf("not implemented")
}

func (q *gocqlmemQuery) Iter() Iter {
	cmds, err := ParseCommands(q.stmt, q.values)
	if err != nil {
		return NewGocqlmemIterWithError(err)
	}
	if len(cmds) != 1 {
		return NewGocqlmemIterWithError(fmt.Errorf("exactly one CQL cmd expected, got: %s", q.stmt))
	}

	switch cmd := cmds[0].(type) {
	case *CommandCreateKeyspace:
		return NewGocqlmemIterWithError(q.session.createKeyspace(cmd))
	case *CommandUseKeyspace:
		return NewGocqlmemIterWithKeyspace(cmd.KeyspaceName)
	case *CommandDropKeyspace:
		return NewGocqlmemIterWithError(q.session.dropKeyspace(cmd))
	case *CommandCreateTable:
		return NewGocqlmemIterWithError(q.session.createTable(cmd))
	case *CommandTruncateTable:
		return NewGocqlmemIterWithError(q.session.truncateTable(cmd))
	case *CommandDropTable:
		return NewGocqlmemIterWithError(q.session.dropTable(cmd))
	case *CommandInsert:
		isApplied, existingColumnInfos, existingValues, err := q.session.execInsert(cmd)
		if err != nil {
			return NewGocqlmemIterWithError(err)
		}

		if err = adjustInternalValuesToClientTypesAccordingToColumnInfos(existingValues, existingColumnInfos); err != nil {
			return NewGocqlmemIterWithError(err)
		}

		existingColumnInfos, existingValues = addAppliedToRetrievedData(cmd.GetCtxKeyspace(), cmd.TableName, existingColumnInfos, existingValues, isApplied)
		return NewGocqlmemIterWithData(cmd.GetCtxKeyspace(), cmd.TableName, existingColumnInfos, existingValues)

	case *CommandSelect:
		var lastSelectedRowIdx int32
		if len(q.pageState) == 0 {
			lastSelectedRowIdx = -1
		} else {
			// This is our implementation of pagestate: we store the idx of the last selected row idx
			err := binary.Read(bytes.NewReader(q.pageState), binary.LittleEndian, &lastSelectedRowIdx)
			if err != nil {
				return NewGocqlmemIterWithError(fmt.Errorf("cannot convert page state %v to int: %s", q.pageState, err.Error()))
			}
		}
		names, values, typeInfos, newLastSelectedRowIdx, err := q.session.execSelect(cmd, int(lastSelectedRowIdx), q.pageSize, q.values)
		if err != nil {
			return NewGocqlmemIterWithError(err)
		}

		if err = adjustInternalValuesToClientTypesAccordingToTypeInfos(values, typeInfos); err != nil {
			return NewGocqlmemIterWithError(err)
		}

		// This is our implementation of pagestate: we store the idx of the last selected row idx
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.LittleEndian, int32(newLastSelectedRowIdx))
		if err != nil {
			return NewGocqlmemIterWithError(fmt.Errorf("cannot convert int %d to byte slice: %s", lastSelectedRowIdx, err.Error()))
		}

		return NewGocqlmemIterWithDataAndPagingState(cmd.GetCtxKeyspace(), cmd.TableName,
			namesAndTypeInfosTocolumnInfos(cmd.GetCtxKeyspace(), cmd.TableName, names, typeInfos),
			values,
			buf.Bytes())

	case *CommandUpdate:
		isApplied, existingColumnInfos, existingValues, err := q.session.execUpdate(cmd, q.values)
		if err != nil {
			return NewGocqlmemIterWithError(err)
		}

		if err = adjustInternalValuesToClientTypesAccordingToColumnInfos(existingValues, existingColumnInfos); err != nil {
			return NewGocqlmemIterWithError(err)
		}

		existingColumnInfos, existingValues = addAppliedToRetrievedData(cmd.GetCtxKeyspace(), cmd.TableName, existingColumnInfos, existingValues, isApplied)
		return NewGocqlmemIterWithData(cmd.GetCtxKeyspace(), cmd.TableName, existingColumnInfos, existingValues)

	case *CommandDelete:
		isApplied, err := q.session.execDelete(cmd, q.values)
		if err != nil {
			return NewGocqlmemIterWithError(err)
		}
		return NewGocqlmemIterWithData(cmd.GetCtxKeyspace(), "",
			[]gocql.ColumnInfo{
				{
					Keyspace: cmd.GetCtxKeyspace(),
					Table:    cmd.TableName,
					Name:     "[applied]",
					TypeInfo: newScalarType(gocql.TypeBoolean),
				}},
			[][]any{{isApplied}})

	default:
		return NewGocqlmemIterWithError(fmt.Errorf("Iter() does not support cmd %v", cmd))
	}
}

func (q *gocqlmemQuery) IterContext(ctx context.Context) Iter {
	// TODO: implement
	return q.Iter()
}

func (q *gocqlmemQuery) MapScan(m map[string]interface{}) error {
	iter := q.Iter()
	if err := iter.Err(); err != nil {
		return err
	}
	if !iter.MapScan(m) {
		return iter.Err()
	}
	return iter.Close()
}

func (q *gocqlmemQuery) MapScanContext(ctx context.Context, m map[string]interface{}) error {
	// TODO: implement
	return q.MapScan(m)
}

func (q *gocqlmemQuery) Scan(dest ...interface{}) error {
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

func (q *gocqlmemQuery) ScanContext(ctx context.Context, dest ...interface{}) error {
	// TODO: implement
	return q.Scan(dest)
}

func (q *gocqlmemQuery) ScanCAS(dest ...interface{}) (applied bool, err error) {
	iter := q.Iter()
	if err := iter.Err(); err != nil {
		return false, err
	}
	if iter.NumRows() == 0 {
		return false, gocql.ErrNotFound
	}
	if len(iter.Columns()) > 1 {
		dest = append([]interface{}{&applied}, dest...)
		iter.Scan(dest...)
	} else {
		iter.Scan(&applied)
	}
	return applied, iter.Close()
}

func (q *gocqlmemQuery) ScanCASContext(ctx context.Context, dest ...interface{}) (applied bool, err error) {
	// TODO: implement
	return q.ScanCAS(dest)
}

// INSERT, UPDATE, DELETE
func (q *gocqlmemQuery) MapScanCAS(dest map[string]interface{}) (applied bool, err error) {
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
	applied = dest["[applied]"].(bool)
	delete(dest, "[applied]")

	return applied, iter.Close()
}

func (q *gocqlmemQuery) MapScanCASContext(ctx context.Context, dest map[string]interface{}) (applied bool, err error) {
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
func (q *gocqlmemQuery) WithNowInSeconds(now int) Query {
	// TODO: implement
	return q
}
