package gocqlmem

import (
	"fmt"
	"strings"
	"testing"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/stretchr/testify/assert"
)

func TestTableSelectOrderBy(t *testing.T) {
	table := tableStore{
		columnDefs: []*columnDef{
			{"col1", PrimaryKeyPartition, gocql.TypeText, ClusteringOrderAsc},
			{"col2", PrimaryKeyClustering, gocql.TypeBigInt, ClusteringOrderDesc},
			{"col3", PrimaryKeyNone, gocql.TypeBoolean, ClusteringOrderNone},
		},
		columnValues: [][]any{
			{"a", "a", "b", "c"},
			{3, 2, 1, 1},
		},
		columnDefMap: map[string]int{"col1": 0, "col2": 1},
	}

	seq, err := table.getRowSequenceFromColumnDefAndSelectOrderBy([]*OrderByField{
		{"col1", ClusteringOrderAsc},
		{"col2", ClusteringOrderDesc},
	})
	assert.Nil(t, err)
	assert.Equal(t, []int{0, 1, 2, 3}, seq)

	seq, err = table.getRowSequenceFromColumnDefAndSelectOrderBy([]*OrderByField{
		{"col1", ClusteringOrderDesc},
		{"col2", ClusteringOrderAsc},
	})
	assert.Nil(t, err)
	assert.Equal(t, []int{3, 2, 1, 0}, seq)

	seq, err = table.getRowSequenceFromColumnDefAndSelectOrderBy([]*OrderByField{
		{"col1", ClusteringOrderAsc},
		{"col2", ClusteringOrderAsc},
	})
	assert.Nil(t, err)
	assert.Equal(t, []int{1, 0, 2, 3}, seq)

	seq, err = table.getRowSequenceFromColumnDefAndSelectOrderBy([]*OrderByField{
		{"col1", ClusteringOrderDesc},
		{"col2", ClusteringOrderDesc},
	})
	assert.Nil(t, err)
	assert.Equal(t, []int{3, 2, 0, 1}, seq)

	seq, err = table.getRowSequenceFromColumnDefAndSelectOrderBy([]*OrderByField{})
	assert.Nil(t, err)
	assert.Equal(t, []int{0, 1, 2, 3}, seq)

	seq, err = table.getRowSequenceFromColumnDefAndSelectOrderBy([]*OrderByField{
		{"col1", ClusteringOrderDesc},
	})
	assert.Nil(t, err)
	assert.Equal(t, []int{3, 2, 0, 1}, seq)

	seq, err = table.getRowSequenceFromColumnDefAndSelectOrderBy([]*OrderByField{
		{"col2", ClusteringOrderAsc},
	})
	assert.Nil(t, err)
	assert.Equal(t, []int{3, 2, 1, 0}, seq)

	seq, err = table.getRowSequenceFromColumnDefAndSelectOrderBy([]*OrderByField{
		{"col3", ClusteringOrderAsc},
		{"col2", ClusteringOrderAsc},
	})
	assert.Contains(t, err.Error(), "cannot process ORDER BY col3, this field is not a clustering key")
}

func TestRowIndexFromColumnDefAndInsert(t *testing.T) {
	var idx int
	var existingIdx int
	var isExists bool
	var err error

	var columnDefs []*columnDef
	var columnValues [][]any

	// ASC ASC

	columnDefs = []*columnDef{
		{"col1", PrimaryKeyPartition, gocql.TypeText, ClusteringOrderAsc},
		{"col2", PrimaryKeyClustering, gocql.TypeBigInt, ClusteringOrderAsc},
		{"col3", PrimaryKeyNone, gocql.TypeBoolean, ClusteringOrderNone},
	}
	columnValues = [][]any{
		{"a", "a", "c", "d"},
		{int64(0), int64(1), int64(3), int64(3)},
	}

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "a",
		"col2": int64(10),
		"col3": true})
	assert.Nil(t, err)
	assert.False(t, isExists)
	assert.Equal(t, -1, existingIdx)
	assert.Equal(t, 2, idx)

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "a",
		"col2": int64(0),
		"col3": true})
	assert.Nil(t, err)
	assert.True(t, isExists)
	assert.Equal(t, 0, existingIdx)
	assert.Equal(t, 1, idx)

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "e",
		"col2": int64(10000),
		"col3": true})
	assert.Nil(t, err)
	assert.False(t, isExists)
	assert.Equal(t, -1, existingIdx)
	assert.Equal(t, 4, idx)

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "A",
		"col2": int64(0),
		"col3": true})
	assert.Nil(t, err)
	assert.False(t, isExists)
	assert.Equal(t, -1, existingIdx)
	assert.Equal(t, 0, idx)

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "b",
		"col2": int64(10000),
		"col3": true})
	assert.Nil(t, err)
	assert.False(t, isExists)
	assert.Equal(t, -1, existingIdx)
	assert.Equal(t, 2, idx)

	// ASC DESC

	columnDefs = []*columnDef{
		{"col1", PrimaryKeyPartition, gocql.TypeText, ClusteringOrderAsc},
		{"col2", PrimaryKeyClustering, gocql.TypeBigInt, ClusteringOrderDesc},
		{"col3", PrimaryKeyNone, gocql.TypeBoolean, ClusteringOrderNone},
	}
	columnValues = [][]any{
		{"a", "a", "c", "d"},
		{int64(3), int64(3), int64(1), int64(0)},
	}

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "a",
		"col2": int64(10),
		"col3": true})
	assert.Nil(t, err)
	assert.False(t, isExists)
	assert.Equal(t, -1, existingIdx)
	assert.Equal(t, 0, idx)

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "a",
		"col2": int64(3),
		"col3": true})
	assert.Nil(t, err)
	assert.True(t, isExists)
	assert.Equal(t, 1, existingIdx)
	assert.Equal(t, 2, idx)

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "c",
		"col2": int64(10000),
		"col3": true})
	assert.Nil(t, err)
	assert.False(t, isExists)
	assert.Equal(t, -1, existingIdx)
	assert.Equal(t, 2, idx)

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "c",
		"col2": int64(1),
		"col3": true})
	assert.Nil(t, err)
	assert.True(t, isExists)
	assert.Equal(t, 2, existingIdx)
	assert.Equal(t, 3, idx)

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "d",
		"col2": int64(-1),
		"col3": true})
	assert.Nil(t, err)
	assert.False(t, isExists)
	assert.Equal(t, -1, existingIdx)
	assert.Equal(t, 4, idx)

	// ASC DESC empty

	columnDefs = []*columnDef{
		{"col1", PrimaryKeyPartition, gocql.TypeText, ClusteringOrderAsc},
		{"col2", PrimaryKeyClustering, gocql.TypeBigInt, ClusteringOrderDesc},
		{"col3", PrimaryKeyNone, gocql.TypeBoolean, ClusteringOrderNone},
	}
	columnValues = [][]any{
		{},
		{},
	}

	idx, existingIdx, isExists, err = getRowIndexFromColumnDefAndInsert(columnValues, columnDefs, map[string]any{
		"col1": "a",
		"col2": int64(1),
		"col3": true})
	assert.Nil(t, err)
	assert.False(t, isExists)
	assert.Equal(t, -1, existingIdx)
	assert.Equal(t, 0, idx)
}

func TestTableInsert(t *testing.T) {
	table := tableStore{
		columnDefs: []*columnDef{
			{"col1", PrimaryKeyPartition, gocql.TypeText, ClusteringOrderAsc},
			{"col2", PrimaryKeyClustering, gocql.TypeBigInt, ClusteringOrderDesc},
			{"col3", PrimaryKeyNone, gocql.TypeBoolean, ClusteringOrderNone},
		},
		columnValues: [][]any{[]any{}, []any{}, []any{}},
		columnDefMap: map[string]int{"col1": 0, "col2": 1, "col3": 2},
	}

	var cmd CommandInsert
	var isApplied bool
	var existingColumnInfos []gocql.ColumnInfo
	var existingValues [][]any
	var err error

	cmd = CommandInsert{
		ColumnNames:  []string{"col1", "col2", "col3"},
		ColumnValues: []any{"a", 1, nil},
		IfNotExists:  false,
	}

	isApplied, existingColumnInfos, existingValues, err = table.execInsert(&cmd)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	assert.Nil(t, existingColumnInfos)
	assert.Nil(t, existingValues)
	assert.Equal(t, "a", table.columnValues[0][0])
	assert.Equal(t, int64(1), table.columnValues[1][0])
	assert.Equal(t, nil, table.columnValues[2][0])

	cmd = CommandInsert{
		ColumnNames:  []string{"col1", "col2", "col3"},
		ColumnValues: []any{"a", 2, true},
		IfNotExists:  false,
	}
	isApplied, existingColumnInfos, existingValues, err = table.execInsert(&cmd)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	assert.Equal(t, "a", table.columnValues[0][0])
	assert.Equal(t, int64(2), table.columnValues[1][0])
	assert.Equal(t, true, table.columnValues[2][0])
	assert.Equal(t, "a", table.columnValues[0][1])
	assert.Equal(t, int64(1), table.columnValues[1][1])
	assert.Equal(t, nil, table.columnValues[2][1])

	cmd = CommandInsert{
		ColumnNames:  []string{"col1", "col2", "col3"},
		ColumnValues: []any{"b", 0, true},
		IfNotExists:  false,
	}
	isApplied, existingColumnInfos, existingValues, err = table.execInsert(&cmd)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	assert.Equal(t, "a", table.columnValues[0][0])
	assert.Equal(t, int64(2), table.columnValues[1][0])
	assert.Equal(t, true, table.columnValues[2][0])
	assert.Equal(t, "a", table.columnValues[0][1])
	assert.Equal(t, int64(1), table.columnValues[1][1])
	assert.Equal(t, nil, table.columnValues[2][1])
	assert.Equal(t, "b", table.columnValues[0][2])
	assert.Equal(t, int64(0), table.columnValues[1][2])
	assert.Equal(t, true, table.columnValues[2][2])

	cmd.IfNotExists = true
	isApplied, existingColumnInfos, existingValues, err = table.execInsert(&cmd)
	assert.Nil(t, err)
	assert.False(t, isApplied)
	assert.Equal(t, 3, len(existingColumnInfos))
	assert.Equal(t, "col1", existingColumnInfos[0].Name)
	assert.Equal(t, "col2", existingColumnInfos[1].Name)
	assert.Equal(t, "col3", existingColumnInfos[2].Name)
	assert.Equal(t, 1, len(existingValues))
	assert.Equal(t, 3, len(existingValues[0]))
	assert.Equal(t, "b", existingValues[0][0])
	assert.Equal(t, int64(0), existingValues[0][1])
	assert.Equal(t, true, existingValues[0][2])

	cmd.IfNotExists = false
	isApplied, existingColumnInfos, existingValues, err = table.execInsert(&cmd)
	assert.Contains(t, err.Error(), "cannot upsert duplicate map[col1:b col2:0 col3:true]")
	assert.False(t, isApplied)
	assert.Equal(t, 3, len(existingColumnInfos))
	assert.Equal(t, "col1", existingColumnInfos[0].Name)
	assert.Equal(t, "col2", existingColumnInfos[1].Name)
	assert.Equal(t, "col3", existingColumnInfos[2].Name)
	assert.Equal(t, 1, len(existingValues))
	assert.Equal(t, 3, len(existingValues[0]))
	assert.Equal(t, "b", existingValues[0][0])
	assert.Equal(t, int64(0), existingValues[0][1])
	assert.Equal(t, true, existingValues[0][2])
}

func TestTableSelect(t *testing.T) {
	table := tableStore{
		columnDefs: []*columnDef{
			{"col1", PrimaryKeyPartition, gocql.TypeText, ClusteringOrderAsc},
			{"col2", PrimaryKeyClustering, gocql.TypeBigInt, ClusteringOrderDesc},
		},
		columnValues: [][]any{
			{"a", "a", "c", "d"},
			{int64(0), int64(1), int64(3), int64(3)},
		},
		columnDefMap: map[string]int{"col1": 0, "col2": 1, "col3": 2},
	}
	var cmds []Command
	var cmd *CommandSelect
	var err error
	var ok bool
	var names []string
	var values [][]any

	cmds, err = ParseCommands(`SELECT t.col1+'a' AS c1, cast(col2*5 as text) as c2 FROM ks1.t WHERE t.col1 = 'a'`)
	assert.Nil(t, err)
	cmd, ok = cmds[0].(*CommandSelect)
	assert.True(t, ok)
	names, values, _, _, err = table.execSelect(cmd, -1, -1)
	assert.Nil(t, err)
	assert.Equal(t, "c1,c2", strings.Join(names, ","))
	assert.Equal(t, 2, len(values))
	assert.Equal(t, "[aa 0]", fmt.Sprintf("%v", values[0]))
	assert.Equal(t, "[aa 5]", fmt.Sprintf("%v", values[1]))

	cmds, err = ParseCommands(`SELECT *, col2 FROM ks1.t WHERE t.col1 = 'a' OR col2 = 3`)
	assert.Nil(t, err)
	cmd, ok = cmds[0].(*CommandSelect)
	assert.True(t, ok)
	names, values, _, _, err = table.execSelect(cmd, -1, -1)
	assert.Nil(t, err)
	assert.Equal(t, "col1,col2,col2", strings.Join(names, ","))
	assert.Equal(t, 4, len(values))
	assert.Equal(t, "[a 0 0]", fmt.Sprintf("%v", values[0]))
	assert.Equal(t, "[a 1 1]", fmt.Sprintf("%v", values[1]))
	assert.Equal(t, "[c 3 3]", fmt.Sprintf("%v", values[2]))
	assert.Equal(t, "[d 3 3]", fmt.Sprintf("%v", values[3]))

	cmds, err = ParseCommands(`SELECT count(*) as c FROM ks1.t WHERE col1='c'`)
	assert.Nil(t, err)
	cmd, ok = cmds[0].(*CommandSelect)
	assert.True(t, ok)
	names, values, _, _, err = table.execSelect(cmd, -1, -1)
	assert.Nil(t, err)
	assert.Equal(t, "c", strings.Join(names, ","))
	assert.Equal(t, 1, len(values))
	assert.Equal(t, "[1]", fmt.Sprintf("%v", values[0]))
}

func TestTableUpdate(t *testing.T) {
	table := tableStore{
		columnDefs: []*columnDef{
			{"col1", PrimaryKeyPartition, gocql.TypeText, ClusteringOrderAsc},
			{"col2", PrimaryKeyClustering, gocql.TypeBigInt, ClusteringOrderDesc},
			{"col3", PrimaryKeyNone, gocql.TypeBigInt, ClusteringOrderNone},
		},
		columnValues: [][]any{
			{"a", "a", "c", "d"},
			{int64(0), int64(1), int64(3), int64(3)},
			{int64(100), int64(101), int64(103), int64(103)},
		},
		columnDefMap: map[string]int{"col1": 0, "col2": 1, "col3": 2},
	}

	var cmds []Command
	var cmd *CommandUpdate
	var isApplied bool
	var existingColumnInfos []gocql.ColumnInfo
	var existingValues [][]any
	var err error
	var ok bool

	cmds, err = ParseCommands(`UPDATE ks1.t SET col3=1001 WHERE t.col1 = 'a'`)
	assert.Nil(t, err)
	cmd, ok = cmds[0].(*CommandUpdate)
	assert.True(t, ok)

	isApplied, existingColumnInfos, existingValues, err = table.execUpdate(cmd)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	assert.Nil(t, existingColumnInfos)
	assert.Nil(t, existingValues)

	assert.Equal(t, "a", table.columnValues[0][0])
	assert.Equal(t, int64(0), table.columnValues[1][0])
	assert.Equal(t, int64(1001), table.columnValues[2][0])

	assert.Equal(t, "a", table.columnValues[0][1])
	assert.Equal(t, int64(1), table.columnValues[1][1])
	assert.Equal(t, int64(1001), table.columnValues[2][1])

	assert.Equal(t, "c", table.columnValues[0][2])
	assert.Equal(t, int64(3), table.columnValues[1][2])
	assert.Equal(t, int64(103), table.columnValues[2][2])

	assert.Equal(t, "d", table.columnValues[0][3])
	assert.Equal(t, int64(3), table.columnValues[1][3])
	assert.Equal(t, int64(103), table.columnValues[2][3])

	// UPSERT
	cmds, err = ParseCommands(`UPDATE ks1.t SET col3=1002 WHERE col1 = 'a' and t.col2 = 100`)
	assert.Nil(t, err)
	cmd, ok = cmds[0].(*CommandUpdate)
	assert.True(t, ok)

	isApplied, existingColumnInfos, existingValues, err = table.execUpdate(cmd)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	assert.Nil(t, existingColumnInfos)
	assert.Nil(t, existingValues)

	// Upserted
	assert.Equal(t, "a", table.columnValues[0][0])
	assert.Equal(t, int64(100), table.columnValues[1][0])
	assert.Equal(t, int64(1002), table.columnValues[2][0])

	// Old
	assert.Equal(t, "a", table.columnValues[0][1])
	assert.Equal(t, int64(0), table.columnValues[1][1])
	assert.Equal(t, int64(1001), table.columnValues[2][1])

	assert.Equal(t, "a", table.columnValues[0][2])
	assert.Equal(t, int64(1), table.columnValues[1][2])
	assert.Equal(t, int64(1001), table.columnValues[2][2])

	assert.Equal(t, "c", table.columnValues[0][3])
	assert.Equal(t, int64(3), table.columnValues[1][3])
	assert.Equal(t, int64(103), table.columnValues[2][3])

	assert.Equal(t, "d", table.columnValues[0][4])
	assert.Equal(t, int64(3), table.columnValues[1][4])
	assert.Equal(t, int64(103), table.columnValues[2][4])
}

func TestTableDelete(t *testing.T) {
	table := tableStore{
		columnDefs: []*columnDef{
			{"col1", PrimaryKeyPartition, gocql.TypeText, ClusteringOrderAsc},
			{"col2", PrimaryKeyClustering, gocql.TypeBigInt, ClusteringOrderDesc},
			{"col3", PrimaryKeyNone, gocql.TypeInt, ClusteringOrderNone},
		},
		columnValues: [][]any{
			{"a", "a", "c", "d"},
			{int64(0), int64(1), int64(3), int64(3)},
			{int64(1), int64(2), int64(3), int64(4)},
		},
		columnDefMap: map[string]int{"col1": 0, "col2": 1, "col3": 2},
	}
	var cmds []Command
	var cmd *CommandDelete
	var err error
	var ok bool
	var isApplied bool

	cmds, err = ParseCommands(`DELETE col3 FROM ks1.t WHERE t.col1 = 'a'`)
	assert.Nil(t, err)
	cmd, ok = cmds[0].(*CommandDelete)
	assert.True(t, ok)
	isApplied, err = table.execDelete(cmd)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	assert.Nil(t, table.columnValues[2][0])
	assert.Nil(t, table.columnValues[2][1])

	cmds, err = ParseCommands(`DELETE FROM ks1.t WHERE t.col1 = 'a'`)
	assert.Nil(t, err)
	cmd, ok = cmds[0].(*CommandDelete)
	assert.True(t, ok)
	isApplied, err = table.execDelete(cmd)
	assert.Nil(t, err)
	assert.True(t, isApplied)
	assert.Equal(t, "c", table.columnValues[0][0])
	assert.Equal(t, "d", table.columnValues[0][1])
}
