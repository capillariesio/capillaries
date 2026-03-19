package gocqlmem

import (
	"bytes"
	"cmp"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type PrimaryKeyType int

const (
	PrimaryKeyPartition PrimaryKeyType = iota
	PrimaryKeyClustering
	PrimaryKeyNone
)

type columnDef struct {
	name            string
	primaryKey      PrimaryKeyType
	columnType      gocql.Type
	clusteringOrder ClusteringOrderType
}

type tableStore struct {
	columnDefs   []*columnDef // Partition,clustering,other
	columnValues [][]any      // Partition,clustering,other
	columnTokens [][]int64    // Partition keys only
	columnDefMap map[string]int
	lock         sync.RWMutex
}

func createColDef(name string, mapColType map[string]gocql.Type, primaryKeyType PrimaryKeyType, mapColClusteringOrder map[string]ClusteringOrderType) (*columnDef, error) {
	dataType, ok := mapColType[name]
	if !ok {
		return nil, fmt.Errorf("cannot find definition for column %s", name)
	}
	clusteringOrder, ok := mapColClusteringOrder[name]
	if !ok {
		clusteringOrder = ClusteringOrderNone
	}
	// For partition keys force ASC, we need it for our internal purposes when we walk through values
	if primaryKeyType == PrimaryKeyPartition {
		clusteringOrder = ClusteringOrderAsc
	}
	return &columnDef{
		name:            name,
		primaryKey:      primaryKeyType,
		columnType:      dataType,
		clusteringOrder: clusteringOrder,
	}, nil
}

func newTable(cmd *CommandCreateTable) (*tableStore, error) {
	t := tableStore{
		columnDefs:   make([]*columnDef, len(cmd.ColumnDefs)),
		columnValues: make([][]any, len(cmd.ColumnDefs)),
		columnTokens: make([][]int64, len(cmd.PartitionKeyColumns)),
		columnDefMap: map[string]int{},
	}

	mapColType := map[string]gocql.Type{}
	var err error
	for _, createTableColDef := range cmd.ColumnDefs {
		mapColType[createTableColDef.Name] = createTableColDef.ColumnType
	}

	mapColClusteringOrder := map[string]ClusteringOrderType{}
	for _, orderByField := range cmd.ClusteringOrderBy {
		// Check definition is present
		if _, ok := mapColType[orderByField.FieldName]; !ok {
			return nil, fmt.Errorf("cannot find definition for clustering order column %s", orderByField.FieldName)
		}
		// Check it's a clustering column
		var isClustering bool
		for _, name := range cmd.ClusteringKeyColumns {
			if orderByField.FieldName == name {
				isClustering = true
				break
			}
		}
		if !isClustering {
			return nil, fmt.Errorf("clustering order column %s is not in the list of clustering keys %v", orderByField.FieldName, cmd.ClusteringKeyColumns)
		}
		// Save ASC/DESC to temp map
		mapColClusteringOrder[orderByField.FieldName] = orderByField.ClusteringOrder
	}

	colDefIdx := 0
	t.columnDefMap = map[string]int{}
	// Partition columns first
	for _, name := range cmd.PartitionKeyColumns {
		if t.columnDefs[colDefIdx], err = createColDef(name, mapColType, PrimaryKeyPartition, mapColClusteringOrder); err != nil {
			return nil, err
		}
		t.columnDefMap[name] = colDefIdx
		colDefIdx++
	}
	// Clustering columns next
	for _, name := range cmd.ClusteringKeyColumns {
		if t.columnDefs[colDefIdx], err = createColDef(name, mapColType, PrimaryKeyClustering, mapColClusteringOrder); err != nil {
			return nil, err
		}
		t.columnDefMap[name] = colDefIdx
		colDefIdx++
	}
	// All other columns next, in the order of appearance in the CREATE TABLE cmd
	for _, createTableColDef := range cmd.ColumnDefs {
		if _, ok := t.columnDefMap[createTableColDef.Name]; !ok {
			if t.columnDefs[colDefIdx], err = createColDef(createTableColDef.Name, mapColType, PrimaryKeyNone, mapColClusteringOrder); err != nil {
				return nil, err
			}
			t.columnDefMap[createTableColDef.Name] = colDefIdx
			colDefIdx++
		}
	}

	return &t, nil
}

func (t *tableStore) getClusteringKeyOrderByName(name string) ClusteringOrderType {
	for _, colDef := range t.columnDefs {
		if name == colDef.name {
			return colDef.clusteringOrder
		}
	}
	return ClusteringOrderNone
}

func getNumericValueSign(v any) (string, any, error) {
	switch typedVal := v.(type) {
	case int64:
		if typedVal >= 0 {
			return "0", typedVal, nil
		}
		return "-", -typedVal, nil

	case float64:
		if typedVal >= 0 {
			return "0", typedVal, nil
		}
		return "-", -typedVal, nil

	case decimal.Decimal:
		if typedVal.Sign() >= 0 {
			return "0", typedVal, nil
		}
		return "-", typedVal.Neg(), nil

	default:
		return "", nil, fmt.Errorf("type %T(%v) not supported", typedVal, typedVal)
	}
}

const BeginningOfTimeMicro = int64(-62135596800000000) // time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC).UnixMicro()

func (t *tableStore) buildKey(orderByFieldsFromSelect []*OrderByField, rowIdx int) (string, error) {
	var keyBuffer bytes.Buffer
	tfm := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	flipReplacer := strings.NewReplacer("0", "9", "1", "8", "2", "7", "3", "6", "4", "5", "5", "4", "6", "3", "7", "2", "8", "1", "9", "0")

	for _, keyField := range orderByFieldsFromSelect {
		var stringValue string
		var strippedFieldName string
		var isToken bool
		if strings.HasPrefix(keyField.FieldName, "token(") {
			strippedFieldName = strings.Replace(strings.Replace(keyField.FieldName, "token(", "", -1), ")", "", -1)
			isToken = true
		} else {
			strippedFieldName = keyField.FieldName
		}

		fieldIdx, ok := t.columnDefMap[strippedFieldName]
		if !ok {
			return "", fmt.Errorf("cannot find field %s", strippedFieldName)
		}
		val := t.columnValues[fieldIdx][rowIdx]

		if isToken {
			tokenVal, err := callToken([]any{val})
			if err != nil {
				return "", fmt.Errorf("cannot calculate token of %v", val)
			}
			tokenValInt64, ok := tokenVal.(int64)
			if !ok {
				return "", fmt.Errorf("unexpectedly, token val in not int64: %T(%v)", tokenVal, tokenVal)
			}
			sign, absVal, err := getNumericValueSign(tokenValInt64)
			if err != nil {
				return "", err
			}
			stringValue = fmt.Sprintf("%s%020d", sign, absVal)
			// If this is a negative value, flip every digit
			if sign == "-" {
				stringValue = flipReplacer.Replace(stringValue)
			}
		} else {
			switch typedVal := val.(type) {
			case int64:
				sign, absVal, err := getNumericValueSign(typedVal)
				if err != nil {
					return "", err
				}
				stringValue = fmt.Sprintf("%s%020d", sign, absVal)
				// If this is a negative value, flip every digit
				if sign == "-" {
					stringValue = flipReplacer.Replace(stringValue)
				}

			case float64:
				// We should support numbers as big as 10^32 and with 32 digits afetr decimal point
				sign, absVal, err := getNumericValueSign(typedVal)
				if err != nil {
					return "", err
				}
				stringValue = strings.ReplaceAll(fmt.Sprintf("%s%66s", sign, fmt.Sprintf("%.32f", absVal)), " ", "0")
				// If this is a negative value, flip every digit
				if sign == "-" {
					stringValue = flipReplacer.Replace(stringValue)
				}

			case decimal.Decimal:
				sign, absVal, err := getNumericValueSign(typedVal)
				if err != nil {
					return "", err
				}
				decVal, ok := absVal.(decimal.Decimal)
				if !ok {
					return "", fmt.Errorf("unexpectedly cannot convert value %v to type decimal2", typedVal)
				}
				floatVal, _ := decVal.Float64()
				stringValue = strings.ReplaceAll(fmt.Sprintf("%s%66s", sign, fmt.Sprintf("%.32f", floatVal)), " ", "0")
				// If this is a negative value, flip every digit
				if sign == "-" {
					stringValue = flipReplacer.Replace(stringValue)
				}

			case time.Time:
				// We support time differences up to microsecond. Not nanosecond! Cassandra supports only milliseconds. Millis are our lingua franca.
				stringValue = fmt.Sprintf("%020d", typedVal.UnixMicro()-BeginningOfTimeMicro)

			case string:
				// Normalize the string
				transformedString, _, _ := transform.String(tfm, typedVal)
				// Take only first 64 (or whatever we have in StringLen) characters
				// use "%-64s" sprint format to pad with spaces on the right
				formatString := fmt.Sprintf("%s-%ds", "%", 64)
				stringValue = fmt.Sprintf(formatString, transformedString)[:64]
				if keyField.CaseSensitivity == ClusteringOrderIgnoreCase {
					stringValue = strings.ToUpper(stringValue)
				}

			case bool:
				if typedVal {
					stringValue = "T" // "F" < "T"
				} else {
					stringValue = "F"
				}

			default:
				return "", fmt.Errorf("cannot build key, unsupported field data type %T(%v)", typedVal, typedVal)
			}
		}

		if keyField.ClusteringOrder == ClusteringOrderDesc {
			stringBytes := []byte(stringValue)
			for i, b := range stringBytes {
				stringBytes[i] = 0xFF - b
			}
			stringValue = hex.EncodeToString(stringBytes)
		}

		keyBuffer.WriteString(stringValue)
	}

	return keyBuffer.String(), nil
}

type clusteringKeyEntry struct {
	Idx int
	Key string
}

type sortNumericKeyEntry struct {
	Idx int
	Key int64
}

func (t *tableStore) getRowSequenceFromColumnDefAndPartitionFieldToken(partitionKeyFieldIdx int) ([]int, error) {
	totalRows := len(t.columnValues[0])
	tempClusteringKey := make([]sortNumericKeyEntry, totalRows)
	for i := range totalRows {
		tokenVal, err := callToken([]any{t.columnValues[partitionKeyFieldIdx][i]})
		if err != nil {
			return nil, fmt.Errorf("cannot calculate token of %v", t.columnValues[partitionKeyFieldIdx][i])
		}
		tokenValInt64, ok := tokenVal.(int64)
		if !ok {
			return nil, fmt.Errorf("unexpectedly, token val in not int64: %T(%v)", tokenVal, tokenVal)
		}
		tempClusteringKey[i] = sortNumericKeyEntry{Idx: i, Key: tokenValInt64}
	}
	slices.SortFunc(tempClusteringKey, func(e1, e2 sortNumericKeyEntry) int {
		return cmp.Compare(e1.Key, e2.Key)
	})

	result := make([]int, len(tempClusteringKey))
	for i := range len(tempClusteringKey) {
		result[i] = tempClusteringKey[i].Idx
	}
	return result, nil
}

func (t *tableStore) getRowSequenceByKey(orderByFieldsFromSelect []*OrderByField) ([]int, error) {
	totalRows := len(t.columnValues[0])
	tempClusteringKey := make([]clusteringKeyEntry, totalRows)
	for rowIdx := range totalRows {
		key, err := t.buildKey(orderByFieldsFromSelect, rowIdx)
		if err != nil {
			return nil, err
		}
		tempClusteringKey[rowIdx] = clusteringKeyEntry{Idx: rowIdx, Key: key}
	}
	slices.SortFunc(tempClusteringKey, func(e1, e2 clusteringKeyEntry) int {
		return cmp.Compare(e1.Key, e2.Key)
	})

	result := make([]int, len(tempClusteringKey))
	for i := range len(tempClusteringKey) {
		result[i] = tempClusteringKey[i].Idx
	}
	return result, nil
}

func (t *tableStore) getRowSequenceFromColumnDefAndSelectOrderBy(orderByFieldsFromSelect []*OrderByField) ([]int, error) {
	totalRows := len(t.columnValues[0])
	if len(orderByFieldsFromSelect) == 0 {
		result := make([]int, totalRows)
		for i := range totalRows {
			result[i] = i
		}
		return result, nil
	}

	tempClusteringKey := make([]clusteringKeyEntry, totalRows)
	for _, orderByFieldFromSelect := range orderByFieldsFromSelect {
		// Each field from SELECT ORDER by must be a clustering key
		tableClusteringOrder := t.getClusteringKeyOrderByName(orderByFieldFromSelect.FieldName)
		if tableClusteringOrder == ClusteringOrderNone {
			return nil, fmt.Errorf("cannot process ORDER BY %s, this field is not a clustering key", orderByFieldFromSelect.FieldName)
		}
		colIdx := t.columnDefMap[orderByFieldFromSelect.FieldName]
		var lastVal any
		var lastTempClusteringKeySegment int
		for i := range totalRows {
			if lastVal == nil {
				lastVal = t.columnValues[colIdx][i]
				if tableClusteringOrder == orderByFieldFromSelect.ClusteringOrder {
					lastTempClusteringKeySegment = 0
				} else {
					lastTempClusteringKeySegment = math.MaxInt32
				}
			} else {
				if t.columnValues[colIdx][i] != lastVal {
					if tableClusteringOrder == orderByFieldFromSelect.ClusteringOrder {
						lastTempClusteringKeySegment++
					} else {
						lastTempClusteringKeySegment--
					}
				}
			}
			if colIdx == 0 {
				tempClusteringKey[i] = clusteringKeyEntry{Idx: i, Key: fmt.Sprintf("0x%08X", lastTempClusteringKeySegment)}
			} else {
				tempClusteringKey[i] = clusteringKeyEntry{Idx: i, Key: fmt.Sprintf("%s0x%08X", tempClusteringKey[i].Key, lastTempClusteringKeySegment)}
			}
		}
	}

	slices.SortFunc(tempClusteringKey, func(e1, e2 clusteringKeyEntry) int {
		return cmp.Compare(e1.Key, e2.Key)
	})

	result := make([]int, len(tempClusteringKey))
	for i := range len(tempClusteringKey) {
		result[i] = tempClusteringKey[i].Idx
	}
	return result, nil
}

/*
func convertLexemToInternalType(lexem *Lexem, cqlType gocql.Type) (any, error) {
	if lexem.T == LexemNull {
		return nil, nil
	}
	switch cqlType {
	case gocql.TypeBigInt, gocql.TypeInt, gocql.TypeTinyInt, gocql.TypeSmallInt, gocql.TypeVarint:
		if lexem.T == LexemIdent || lexem.T == LexemPointedIdent {
			constVal, ok := GocqlmemEvalConstants[lexem.V]
			if !ok {
				return 0, fmt.Errorf("cannot convert %v to integer, unknown constant", lexem.V)
			}
			intVal, ok := constVal.(int64)
			if !ok {
				return 0, fmt.Errorf("cannot convert %v to integer, the constant is not of int64 type", lexem.V)
			}
			return intVal, nil

		} else if lexem.T != LexemNumberLiteral {
			return 0, fmt.Errorf("cannot convert %v to integer, lexem type %d not supported", lexem.V, lexem.T)
		}
		val, err := strconv.ParseInt(lexem.V, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %v to integer: %s", lexem.V, err.Error())
		}
		return val, nil

	case gocql.TypeDouble, gocql.TypeFloat:
		if lexem.T == LexemIdent || lexem.T == LexemPointedIdent {
			constVal, ok := GocqlmemEvalConstants[lexem.V]
			if !ok {
				return 0, fmt.Errorf("cannot convert %v to float, unknown constant", lexem.V)
			}
			floatVal, ok := constVal.(float64)
			if !ok {
				return 0, fmt.Errorf("cannot convert %v to float, the constant is not of float64 type", lexem.V)
			}
			return floatVal, nil

		} else if lexem.T != LexemNumberLiteral {
			return 0, fmt.Errorf("cannot convert %v to float, lexem type %d not supported", lexem.V, lexem.T)
		}
		val, err := strconv.ParseFloat(lexem.V, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %v to float: %s", lexem.V, err.Error())
		}
		return float64(val), nil

	case gocql.TypeDecimal:
		if lexem.T != LexemNumberLiteral {
			return 0, fmt.Errorf("cannot convert %v to decimal, lexem type %d not supported", lexem.V, lexem.T)
		}

		d, ok := new(inf.Dec).SetString(lexem.V)
		if !ok {
			return 0, fmt.Errorf("cannot convert %v to decimal", lexem.V)
		}
		return d, nil

	case gocql.TypeText, gocql.TypeVarchar:
		if lexem.T != LexemStringLiteral {
			return 0, fmt.Errorf("cannot convert %v to string, lexem type %d not supported", lexem.V, lexem.T)
		}
		return lexem.V, nil

	case gocql.TypeBoolean:
		if lexem.T != LexemBoolLiteral {
			return 0, fmt.Errorf("cannot convert %v to bool, lexem type %d not supported", lexem.V, lexem.T)
		}
		return lexem.V == "TRUE", nil

	default:
		return 0, fmt.Errorf("unknown column type %v", cqlType)
	}
}
*/

func getRowIndexFromColumnDefAndInsert(columnValues [][]any, columnDefs []*columnDef, insertedColumnValues map[string]any) (int, int, bool, error) {
	topIdx := 0                       // Top candidate for replacement
	bottomIdx := len(columnValues[0]) // One step below the last candidate
	var isExists bool
	existingIdxCandidate := -1
	for tableColIdx, tableColDef := range columnDefs {
		insertedColVal := insertedColumnValues[tableColDef.name]
		newStartIdx := -1
		newEndIdx := -1
		curIdx := topIdx
		for curIdx < bottomIdx {
			curVal := columnValues[tableColIdx][curIdx]
			compareResult, err := compareInternalKnownType(curVal, insertedColVal, tableColDef.columnType)
			if err != nil {
				return -1, -1, false, fmt.Errorf("cannot compare existing %v to inserted %v", curVal, insertedColVal)
			}
			if tableColDef.clusteringOrder == ClusteringOrderDesc {
				compareResult *= -1
			}
			if compareResult == 0 {
				if newStartIdx == -1 {
					newStartIdx = curIdx
				}
				newEndIdx = curIdx + 1
				existingIdxCandidate = curIdx
			} else if compareResult == 1 {
				// curVal > insertedColVal (ASC) or curVal < insertedColVal (DESC), time to end looking
				if newStartIdx == -1 {
					return curIdx, -1, false, nil
				}
				newEndIdx = curIdx
				break
			}
			curIdx++
		}
		if newStartIdx == -1 {
			// No equal or greater (ASC) or smaller (DESC) values found in this column,
			// ready to insert in the previously harvested range startIdx,endIdx
			if tableColDef.clusteringOrder == ClusteringOrderAsc {
				return bottomIdx, -1, false, nil
			}
			return bottomIdx, -1, false, nil
		}
		// Now we have a range of size at least to, say:
		// newStartIdx = 5 (with value eq to the inserted one)
		// newEndIdx = 6 (with value > to the inserted one, or beyond the range)

		// Proceed to the next key column with updated range
		topIdx = newStartIdx
		bottomIdx = newEndIdx

		// If there are no more key columns, insert here
		if tableColIdx == len(columnDefs)-1 || columnDefs[tableColIdx+1].primaryKey == PrimaryKeyNone {
			isExists = true
			break
		}
	}
	if !isExists {
		existingIdxCandidate = -1
	}
	return bottomIdx, existingIdxCandidate, isExists, nil
}

func (t *tableStore) execInternalUpsert(cmd *CommandInsert) (bool, []gocql.ColumnInfo, [][]any, error) {
	for _, tableColDef := range t.columnDefs {
		if tableColDef.primaryKey == PrimaryKeyPartition || tableColDef.primaryKey == PrimaryKeyClustering {
			var isPresent bool
			for _, colName := range cmd.ColumnNames {
				if colName == tableColDef.name {
					isPresent = true
					break
				}
			}
			if !isPresent {
				return false, nil, nil, fmt.Errorf("partition/clustering column key %s must be specified in the upsert command", tableColDef.name)
			}
		}
	}

	var err error
	var insertIdx int
	var isAlreadyExists bool
	var existingIdx int

	insertedColumnValues := map[string]any{}
	for i, name := range cmd.ColumnNames {
		if t.columnDefs[t.columnDefMap[name]].columnType == gocql.TypeCounter {
			return false, nil, nil, fmt.Errorf("cannot insert value %T(%v) into counter column %s, only updates are supported", cmd.ColumnValues[i], cmd.ColumnValues[i], name)
		}
		insertedColumnValues[name], err = sanitizeToInternalKnownType(cmd.ColumnValues[i], t.columnDefs[t.columnDefMap[name]].columnType)
		if err != nil {
			return false, nil, nil, fmt.Errorf("cannot cast column %d(%s) to internal type %v: %s", i, name, cmd.ColumnValues[i], err.Error())
		}
	}

	// Initialize counter columns with zeroes
	for _, colDef := range t.columnDefs {
		if colDef.columnType == gocql.TypeCounter {
			insertedColumnValues[colDef.name] = int64(0)
		}
	}

	if len(t.columnValues[0]) > 0 {
		insertIdx, existingIdx, isAlreadyExists, err = getRowIndexFromColumnDefAndInsert(t.columnValues, t.columnDefs, insertedColumnValues)
		if err != nil {
			return false, nil, nil, fmt.Errorf("cannot find upsert idx for %v: %s", insertedColumnValues, err.Error())
		}

		var existingColumnInfos []gocql.ColumnInfo
		var existingValues [][]any
		if isAlreadyExists {
			existingColumnInfos = make([]gocql.ColumnInfo, len(t.columnDefs))
			existingValues = make([][]any, 1)
			existingValues[0] = make([]any, len(t.columnDefs))
			for colIdx, tableColDef := range t.columnDefs {
				existingColumnInfos[colIdx] = gocql.ColumnInfo{
					Keyspace: cmd.GetCtxKeyspace(),
					Table:    cmd.TableName,
					Name:     tableColDef.name,
					TypeInfo: newScalarType(tableColDef.columnType),
				}
				existingValues[0][colIdx] = t.columnValues[colIdx][existingIdx]
			}
		}

		if isAlreadyExists && cmd.IfNotExists {
			return false, existingColumnInfos, existingValues, nil
		}

		if isAlreadyExists && !cmd.IfNotExists {
			return false, existingColumnInfos, existingValues, fmt.Errorf("cannot upsert duplicate %v", insertedColumnValues)
		}

		for tableColIdx, tableColDef := range t.columnDefs {
			val, ok := insertedColumnValues[tableColDef.name]
			if !ok {
				val = nil
			}
			t.columnValues[tableColIdx] = slices.Insert(t.columnValues[tableColIdx], insertIdx, val)
		}
		if cmd.TableName == "order_item_date_inner_00001" && insertedColumnValues["order_id"].(string) == "001d9673d0e150471d536c210ce20123" {
			fmt.Printf("internalwritingdata insert %v\n", insertedColumnValues)
		}
	} else {
		for tableColIdx, tableColDef := range t.columnDefs {
			val, ok := insertedColumnValues[tableColDef.name]
			if !ok {
				val = nil
			}
			t.columnValues[tableColIdx] = append(t.columnValues[tableColIdx], val)
		}

		if cmd.TableName == "order_item_date_inner_00001" && insertedColumnValues["order_id"].(string) == "001d9673d0e150471d536c210ce20123" {
			fmt.Printf("internalwritingdata append %v\n", insertedColumnValues)
		}

	}
	return true, nil, nil, nil
}

func (t *tableStore) execTruncate(cmd *CommandTruncateTable) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	for i := range len(t.columnValues) {
		t.columnValues[i] = t.columnValues[i][:0]
	}
	return nil
}

func (t *tableStore) execInsert(cmd *CommandInsert) (bool, []gocql.ColumnInfo, [][]any, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	//insertedColumnValues := map[string]any{}
	for i, name := range cmd.ColumnNames {
		if cmd.ColumnValues[i] == nil && (t.columnDefs[i].primaryKey == PrimaryKeyPartition || t.columnDefs[i].primaryKey == PrimaryKeyClustering) {
			return false, nil, nil, fmt.Errorf("cannot insert NULL into a partition/clustered key column %s", name)
		}
		/*
			eCtx := eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, nil)
			colVal, err := eCtx.Eval(cmd.ColumnValueExpAsts[i])
			if err != nil {
				return false, nil, nil, fmt.Errorf("cannot eval inserted column %d: %s", i, err.Error())
			}
			if colVal == nil && (t.columnDefs[t.columnDefMap[name]].primaryKey == PrimaryKeyPartition || t.columnDefs[t.columnDefMap[name]].primaryKey == PrimaryKeyClustering) {
				return false, nil, nil, fmt.Errorf("cannot insert NULL into a partition/clustered key column %s", name)
			}
		*/
		/*
			insertedColumnValues[name], err = convertLexemToInternalType(cmd.ColumnValues[i], t.columnDefs[t.columnDefMap[name]].columnType)
			if err != nil {
				return false, nil, nil, fmt.Errorf("cannot cast upserted column %d: %s", i, err.Error())
			}
			if insertedColumnValues[name] == nil && (t.columnDefs[t.columnDefMap[name]].primaryKey == PrimaryKeyPartition || t.columnDefs[t.columnDefMap[name]].primaryKey == PrimaryKeyClustering) {
				return false, nil, nil, fmt.Errorf("cannot upsert NULL into a partition/clustered key column %s", name)
			}
		*/
	}

	return t.execInternalUpsert(cmd)
}

// Looks for select expressions like "*" and "table1.*"
func isSelectAsterisk(tableName string, lexems []*Lexem) bool {
	if len(lexems) == 1 {
		if lexems[0].T == LexemAsterisk {
			return true
		}
		if lexems[0].T == LexemPointedAsterisk {
			parts := strings.Split(lexems[0].V, ".")
			if len(parts) == 2 && parts[0] == tableName {
				return true
			}
		}
	}
	return false
}

/*
func getResultNamesAndExpressions(tableName string, columnDefs []*columnDef, selectExpLexems [][]*Lexem, selectExpAsts []ast.Expr) ([]string, []ast.Expr, error) {
	resultNames := []string{}
	resultExps := []ast.Expr{}
	for selectItemIdx, selectLexems := range selectExpLexems {
		// Handle SELECT * and SELECT t.*
		if isSelectAsterisk(tableName, selectLexems) {
			for colIdx := range len(columnDefs) {
				resultExp, err := parser.ParseExpr(columnDefs[colIdx].name)
				if err != nil {
					return nil, nil, fmt.Errorf("dev error, cannot parse column name %s: %s", columnDefs[colIdx].name, err.Error())
				}
				resultNames = append(resultNames, columnDefs[colIdx].name)
				resultExps = append(resultExps, resultExp)
			}
			continue
		}
		// Handle count(*), count(t.*) count(field_name), count(null)
		if len(selectLexems) >= 4 && (selectLexems[0].V == "count" || selectLexems[0].V == "COUNT") && selectLexems[1].V == "(" {
			var lexemsUnderCount []*Lexem
			if selectLexems[len(selectLexems)-2].T == LexemAs {
				lexemsUnderCount = selectLexems[2 : len(selectLexems)-3]
			} else {
				lexemsUnderCount = selectLexems[2 : len(selectLexems)-1]
			}
			var resultExpString string
			if isSelectAsterisk(tableName, lexemsUnderCount) {
				resultNames = append(resultNames, fmt.Sprintf("count(%s)", selectLexems[2].V))
				resultExpString = "count()"
			} else {
				s, err := lexemsToStringForColumnExpression(lexemsUnderCount)
				if err != nil {
					return nil, nil, fmt.Errorf("cannot parse count(...): %s", err.Error())
				}
				resultNames = append(resultNames, fmt.Sprintf("count(%s)", s))
				resultExpString = fmt.Sprintf("count_if((%s) != NULL)", s) // GocqlmemEvalConstants will take care of NULL
			}
			resultExp, err := parser.ParseExpr(resultExpString)
			if err != nil {
				return nil, nil, fmt.Errorf("cannot parse count(...) expression [%s]: %s", resultExpString, err.Error())
			}
			// Override result column name if needed
			if selectLexems[len(selectLexems)-2].T == LexemAs {
				resultNames[len(resultNames)-1] = selectLexems[len(selectLexems)-1].V
			}
			resultExps = append(resultExps, resultExp)
			continue
		}
		// Handle everything else
		s, as, err := lexemsToStringForColumnExpression(selectLexems)
		if err != nil {
			return nil, nil, err
		}
		if as == "" {
			as = s
		}
		resultNames = append(resultNames, as)
		resultExps = append(resultExps, selectExpAsts[selectItemIdx])
	}
	return resultNames, resultExps, nil
}
*/
/*
func replaceAsteriskInColumnNames(tableName string, columnDefs []*columnDef, columnExpLexems [][]*Lexem, columnExpNames []string, columnExpAsts []ast.Expr) ([]string, []ast.Expr, error) {
	newColumnExpNames := []string{}
	newColumnExpAsts := []ast.Expr{}
	for colIdx, columnExpLexems := range columnExpLexems {
		// Handle SELECT * and SELECT t.*
		if isSelectAsterisk(tableName, columnExpLexems) {
			for colIdx := range len(columnDefs) {
				asteriskColumnExpAst, err := parser.ParseExpr(columnDefs[colIdx].name)
				if err != nil {
					return nil, nil, fmt.Errorf("dev error, cannot parse column name %s: %s", columnDefs[colIdx].name, err.Error())
				}
				newColumnExpNames = append(newColumnExpNames, columnDefs[colIdx].name)
				newColumnExpAsts = append(newColumnExpAsts, asteriskColumnExpAst)
			}
			continue
		}
		newColumnExpNames = append(newColumnExpNames, columnExpNames[colIdx])
		newColumnExpAsts = append(newColumnExpAsts, columnExpAsts[colIdx])
	}
	return newColumnExpNames, newColumnExpAsts, nil
}
*/

// Handle SELECT * and SELECT t.*
func replaceAsteriskInColumnName(tableName string, columnDefs []*columnDef, columnExpLexems []*Lexem) ([]string, []ast.Expr, error) {
	if !isSelectAsterisk(tableName, columnExpLexems) {
		return nil, nil, nil
	}
	newColumnExpNames := []string{}
	newColumnExpAsts := []ast.Expr{}
	for colIdx := range len(columnDefs) {
		asteriskColumnExpAst, err := parser.ParseExpr(columnDefs[colIdx].name)
		if err != nil {
			return nil, nil, fmt.Errorf("dev error, cannot parse column name %s: %s", columnDefs[colIdx].name, err.Error())
		}
		newColumnExpNames = append(newColumnExpNames, columnDefs[colIdx].name)
		newColumnExpAsts = append(newColumnExpAsts, asteriskColumnExpAst)
	}
	return newColumnExpNames, newColumnExpAsts, nil
}

/*
func populateFieldsUnderCount(tableName string, columnDefs []*columnDef, columnExpLexems [][]*Lexem, columnExpNames []string, columnExpAsts []ast.Expr) ([]string, []ast.Expr, error) {
	newColumnExpNames := []string{}
	newColumnExpAsts := []ast.Expr{}
	for colIdx, columnExpLexems := range columnExpLexems {
		// Handle count(*), count(t.*) count(field_name), count(null)
		if len(columnExpLexems) >= 4 && (columnExpLexems[0].V == "count" || columnExpLexems[0].V == "COUNT") && columnExpLexems[1].V == "(" {
			var lexemsUnderCount []*Lexem
			if columnExpLexems[len(columnExpLexems)-2].T == LexemAs {
				lexemsUnderCount = columnExpLexems[2 : len(columnExpLexems)-3]
			} else {
				lexemsUnderCount = columnExpLexems[2 : len(columnExpLexems)-1]
			}
			// Figure out new column name and a expression string
			var resultExpString string
			if isSelectAsterisk(tableName, lexemsUnderCount) {
				// count(*), count(t.*)
				newColumnExpNames = append(newColumnExpNames, fmt.Sprintf("count(%s)", columnExpLexems[2].V))
				resultExpString = "count()"
			} else {
				// count(field_name), count(null)
				newColumnExpNames = append(newColumnExpNames, fmt.Sprintf("count(%s)", lexemsToStringForColumnNames(lexemsUnderCount)))
				newColumnExpAst, err := lexemsToStringForColumnExpression(lexemsUnderCount)
				if err != nil {
					return nil, nil, fmt.Errorf("cannot parse count(...): %s", err.Error())
				}
				resultExpString = fmt.Sprintf("count_if((%s) != NULL)", newColumnExpAst) // GocqlmemEvalConstants will take care of NULL
			}
			// Parse newly crafted expression string and add it to the result
			resultExp, err := parser.ParseExpr(resultExpString)
			if err != nil {
				return nil, nil, fmt.Errorf("cannot parse count(...) expression [%s]: %s", resultExpString, err.Error())
			}
			newColumnExpAsts = append(newColumnExpAsts, resultExp)
			continue
		}
		newColumnExpNames = append(newColumnExpNames, columnExpNames[colIdx])
		newColumnExpAsts = append(newColumnExpAsts, columnExpAsts[colIdx])
	}
	return newColumnExpNames, newColumnExpAsts, nil
}
*/

// Handle count(*), count(t.*) count(field_name), count(null)
func populateFieldsUnderCount(tableName string, columnDefs []*columnDef, columnExpLexems []*Lexem) (string, ast.Expr, error) {
	if len(columnExpLexems) < 4 || (columnExpLexems[0].V != "count" && columnExpLexems[0].V != "COUNT") || columnExpLexems[1].V != "(" {
		return "", nil, nil
	}
	var lexemsUnderCount []*Lexem
	if columnExpLexems[len(columnExpLexems)-2].T == LexemAs {
		lexemsUnderCount = columnExpLexems[2 : len(columnExpLexems)-3]
	} else {
		lexemsUnderCount = columnExpLexems[2 : len(columnExpLexems)-1]
	}
	// Figure out new column name and a expression string
	var newColumnExpName string
	var newColumnExpAstString string
	if isSelectAsterisk(tableName, lexemsUnderCount) {
		// count(*), count(t.*)
		newColumnExpName = fmt.Sprintf("count(%s)", columnExpLexems[2].V)
		newColumnExpAstString = "count()"
	} else {
		// count(field_name), count(null)
		newColumnExpName = fmt.Sprintf("count(%s)", lexemsToStringForColumnNames(lexemsUnderCount))
		newColumnExpAst, err := lexemsToStringForColumnExpression(lexemsUnderCount)
		if err != nil {
			return "", nil, fmt.Errorf("cannot parse count(...): %s", err.Error())
		}
		newColumnExpAstString = fmt.Sprintf("count_if((%s) != NULL)", newColumnExpAst) // GocqlmemEvalConstants will take care of NULL
	}
	// Parse newly crafted expression string and add it to the result
	newColumnExpAst, err := parser.ParseExpr(newColumnExpAstString)
	if err != nil {
		return "", nil, fmt.Errorf("cannot parse count(...) expression [%s]: %s", newColumnExpAstString, err.Error())
	}

	// We may have lost AS, catch up here
	if columnExpLexems[len(columnExpLexems)-2].T == LexemAs {
		newColumnExpName = columnExpLexems[len(columnExpLexems)-1].V
	}

	return newColumnExpName, newColumnExpAst, nil
}

func findTokenPartitionFieldsInLexems(lexems []*Lexem, tableName string, columnDefs []*columnDef) []string {
	result := []string{}
	// Follow field order: partition keys always go first
	for _, colDef := range columnDefs {
		if colDef.primaryKey != PrimaryKeyPartition {
			continue
		}
		for lexemIdx, l := range lexems {
			if l.V != colDef.name && l.V != tableName+"."+colDef.name {
				continue
			}
			if lexemIdx < 2 || lexems[lexemIdx-1].V != "(" || lexems[lexemIdx-2].V != "token" {
				continue
			}
			// We have a token(<partitioning key field>) expression, return this field name
			finalFieldName := "token(" + colDef.name + ")"
			var isAlreadyThere bool
			for _, fieldName := range result {
				if finalFieldName == fieldName {
					isAlreadyThere = true
					break
				}
			}
			if !isAlreadyThere {
				result = append(result, "token("+colDef.name+")")
			}
		}
	}
	return result
}

/*
func findTokenPartitionFieldInLexems(lexems []*Lexem, tableName string, columnDefs []*columnDef) int {
	// Follow field order: partition keys always go first
	for colIdx, colDef := range columnDefs {
		if colDef.primaryKey != PrimaryKeyPartition {
			continue
		}
		for lexemIdx, l := range lexems {
			if l.V != colDef.name && l.V != tableName+"."+colDef.name {
				continue
			}
			if lexemIdx < 2 || lexems[lexemIdx-1].V != "(" || lexems[lexemIdx-2].V != "token" {
				continue
			}
			// We have a token(<partitioning key field>) expression, return this field name
			return colIdx
		}
	}
	return -1
}
*/

func (t *tableStore) execSelect(cmd *CommandSelect, lastSelectedRowIdx int, maxRows int, preparedQueryParams []interface{}) ([]string, [][]any, []gocql.TypeInfo, int, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var selectSeq []int
	var err error

	// IMPORTANT!
	// Essential, but sometimes overlooked Cassandra feature:
	// When querying Cassandra using the token() function in the WHERE clause,
	// the results are returned ordered by the hash value of the partition key,
	// not the partition key itself. This is guaranteed for standard partitioners like Murmur3Partitioner.

	orderByTokenFields := findTokenPartitionFieldsInLexems(cmd.WhereExpLexems, cmd.TableName, t.columnDefs)
	if len(orderByTokenFields) > 0 {
		// This method requires building keys for each row and sorting them
		combinedOrderByFields := []*OrderByField{}
		for _, tokenFieldName := range orderByTokenFields {
			combinedOrderByFields = append(combinedOrderByFields, &OrderByField{tokenFieldName, ClusteringOrderAsc, ClusteringOrderCaseSensitive})
		}
		combinedOrderByFields = append(combinedOrderByFields, cmd.OrderByFields...)

		selectSeq, err = t.getRowSequenceByKey(combinedOrderByFields)
		if err != nil {
			return nil, nil, nil, -1, fmt.Errorf("cannot get token-based row sequence: %s", err.Error())
		}
	} else {
		// This method uses already sorted data
		selectSeq, err = t.getRowSequenceFromColumnDefAndSelectOrderBy(cmd.OrderByFields)
		if err != nil {
			return nil, nil, nil, -1, fmt.Errorf("cannot get field-based row sequence: %s", err.Error())
		}
	}

	/*
		orderByTokenFieldIdx := findTokenPartitionFieldInLexems(cmd.WhereExpLexems, cmd.TableName, t.columnDefs)
		if orderByTokenFieldIdx >= 0 {
			// This method requires building keys for each row and sorting them
			selectSeq, err = t.getRowSequenceFromColumnDefAndPartitionFieldToken(orderByTokenFieldIdx)
			if err != nil {
				return nil, nil, nil, -1, fmt.Errorf("cannot get token-based row sequence: %s", err.Error())
			}
		} else {
			// This method uses already sorted data
			selectSeq, err = t.getRowSequenceFromColumnDefAndSelectOrderBy(cmd.OrderByFields)
			if err != nil {
				return nil, nil, nil, -1, fmt.Errorf("cannot get field-based row sequence: %s", err.Error())
			}
		}
	*/

	// Unwrap asterisks. It would be nice to do that in the parser, but too bad we have no idea about table field defs when parsing.
	newColumnExpNames := []string{}
	newColumnExpAsts := []ast.Expr{}
	for colIdx, columnExpLexems := range cmd.SelectExpLexems {
		createdNames, createdAsts, err := replaceAsteriskInColumnName(cmd.TableName, t.columnDefs, columnExpLexems)
		if err != nil {
			return nil, nil, nil, -1, err
		}
		if len(createdNames) > 0 {
			newColumnExpNames = append(newColumnExpNames, createdNames...)
			newColumnExpAsts = append(newColumnExpAsts, createdAsts...)
			continue
		}
		createdName, createdAst, err := populateFieldsUnderCount(cmd.TableName, t.columnDefs, columnExpLexems)
		if err != nil {
			return nil, nil, nil, -1, err
		}
		if createdName != "" {
			newColumnExpNames = append(newColumnExpNames, createdName)
			newColumnExpAsts = append(newColumnExpAsts, createdAst)
			continue
		}
		newColumnExpNames = append(newColumnExpNames, cmd.SelectExpNames[colIdx])
		newColumnExpAsts = append(newColumnExpAsts, cmd.SelectExpAsts[colIdx])
	}

	// WARNING: after this, cmd.SelectExpLexems can be shorter than cmd.SelectExpNames/cmd.SelectExpAsts because we unwrapped asterisks
	cmd.SelectExpNames = newColumnExpNames
	cmd.SelectExpAsts = newColumnExpAsts

	typeInfos := make([]gocql.TypeInfo, len(cmd.SelectExpNames))

	var isAgg bool
	aggCtxs := make([]*eval.EvalCtx, len(cmd.SelectExpAsts))
	for internalRowIdx, resultExp := range cmd.SelectExpAsts {
		aggEnabled, aggFuncType, aggFuncArgs := eval.DetectRootAggFunc(resultExp)
		if aggEnabled == eval.AggFuncEnabled {
			aggCtxs[internalRowIdx], err = eval.NewAggEvalCtx(aggFuncType, aggFuncArgs, GocqlmemEvalFunctions, GocqlmemEvalConstants, nil)
			if err != nil {
				return nil, nil, nil, -1, err
			}
			isAgg = true
		} else {
			aggCtxs[internalRowIdx] = eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, nil)
		}
	}

	// Ignore paging for agg selects
	if isAgg {
		lastSelectedRowIdx = -1
		maxRows = -1
	}

	resultRows := [][]any{}
	valMap := eval.VarValuesMap{}
	valMap[""] = map[string]any{}
	valMap[cmd.TableName] = map[string]any{}
	// In Apache Cassandra, if you use an aggregate function (like SUM, AVG, COUNT, MAX, MIN) and select other non-aggregated columns in the same query,
	// Cassandra returns the first row it encounters for those non-aggregated columns. So, use isFirstHitAlreadyPassed.
	var isFirstHitAlreadyPassed bool
	isWithinRequestedPage := (lastSelectedRowIdx == -1)
	newLastSelectedRowIdx := -1
	selectedRowCount := 0
	for _, i := range selectSeq {
		if !isWithinRequestedPage {
			if i == lastSelectedRowIdx {
				// Next iteratio will hit the first row row we have to return
				isWithinRequestedPage = true
			}
			continue
		}
		newLastSelectedRowIdx = i
		resultRow := []any{}
		clear(valMap[""])
		clear(valMap[cmd.TableName])
		for colIdx, colDef := range t.columnDefs {
			valMap[""][colDef.name] = t.columnValues[colIdx][i]
			valMap[cmd.TableName][colDef.name] = t.columnValues[colIdx][i]
		}

		// Add prepared params to the value map that is used for where and for select values, see below
		if err = addPreparedQueryParamsToMap(valMap, preparedQueryParams); err != nil {
			return nil, nil, nil, -1, fmt.Errorf("cannot apply prepared params: %s", err.Error())
		}

		isInclude := true
		var ok bool
		if cmd.WhereExpAst != nil {
			eCtx := eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, valMap)
			isIncludeAny, err := eCtx.Eval(cmd.WhereExpAst)
			if err != nil {
				return nil, nil, nil, -1, fmt.Errorf("cannot evaluate where expression: %s", err.Error())
			}

			isInclude, ok = isIncludeAny.(bool)
			if !ok {
				return nil, nil, nil, -1, fmt.Errorf("where expressions return %T, expected bool", isIncludeAny)
			}
		}

		if isInclude {
			for resultColIdx, selectExpAst := range cmd.SelectExpAsts {
				var val any
				var err error
				if isAgg && (aggCtxs[resultColIdx].IsAggFuncEnabled() || !isFirstHitAlreadyPassed) || !isAgg {
					aggCtxs[resultColIdx].SetVars(valMap)
					val, err = aggCtxs[resultColIdx].Eval(selectExpAst)
					if err != nil {
						return nil, nil, nil, -1, err
					}
				}
				if !isAgg {
					resultRow = append(resultRow, val)
				}

				if typeInfos[resultColIdx] == nil {
					if tableColIdx, ok := t.columnDefMap[cmd.SelectExpNames[resultColIdx]]; ok {
						// A pure table column was selected, return its type
						typeInfos[resultColIdx] = newScalarType(t.columnDefs[tableColIdx].columnType)
					} else {
						// An expression used, return our best guess
						if val != nil {
							typ, err := guessInternalValueType(val)
							if err != nil {
								return nil, nil, nil, -1, fmt.Errorf("cannot guess type of returned column %d: %s", resultColIdx, err.Error())
							}
							typeInfos[resultColIdx] = newScalarType(typ)
						}
					}
				}
			}
			isFirstHitAlreadyPassed = true
			if !isAgg {
				resultRows = append(resultRows, resultRow)
			}

			selectedRowCount++
			if selectedRowCount == maxRows {
				break
			}
		}
	}

	if isAgg {
		resultRow := []any{}
		for resultColIdx := range cmd.SelectExpAsts {
			resultRow = append(resultRow, aggCtxs[resultColIdx].GetValue())
		}
		resultRows = append(resultRows, resultRow)
	}

	return cmd.SelectExpNames, resultRows, typeInfos, newLastSelectedRowIdx, nil
}

func getInsertedPriKeyColumnNameFromEql(tableName string, columnDefMap map[string]int, exp ast.Expr) (string, error) {
	switch typedExp := exp.(type) {
	case *ast.Ident:
		if _, ok := columnDefMap[typedExp.Name]; ok {
			return typedExp.Name, nil
		}
		// It's an ident, but not a column name, maybe it's a still valid right-side ident like col1 == TRUE
		return "", nil

	case *ast.SelectorExpr:
		switch tableIdent := typedExp.X.(type) {
		case *ast.Ident:
			if tableIdent.Name != tableName {
				// It's a selector ident, but not a column in this table, maybe it's a still valid right-side ident like col1 == some_namespace.some_selector
				return "", nil
			}
			return typedExp.Sel.Name, nil
		default:
			return "", fmt.Errorf("expected %s.col_name == ..., got invalid table name of type %T", tableName, tableIdent)
		}
	default:
		// It's a generic expression, maybe it's a still valid right-side ident like col1 == some_func(a*b)
		return "", nil
	}
}

func getInsertedPriKeyColumnValuePairFromEql(tableName string, columnDefs []*columnDef, columnDefMap map[string]int, eqlExp ast.BinaryExpr) (string, any, error) {
	var colName string
	var err error
	var colValExp ast.Expr

	// Try left side for column name
	colName, err = getInsertedPriKeyColumnNameFromEql(tableName, columnDefMap, eqlExp.X)
	if err != nil {
		return "", nil, err
	}
	if colName != "" {
		colValExp = eqlExp.Y
	} else {
		// Try right side for column name
		colName, err = getInsertedPriKeyColumnNameFromEql(tableName, columnDefMap, eqlExp.Y)
		if err != nil {
			return "", nil, err
		}
		if colName != "" {
			colValExp = eqlExp.X
		} else {
			return "", nil, fmt.Errorf("cannot find column ident in the expected col1 == ... , got %T == %T", eqlExp.X, eqlExp.Y)
		}
	}

	// Column value exp can be something like round(2.3), it does not have to be a literal
	eCtx := eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, nil)
	colValAny, err := eCtx.Eval(colValExp)
	if err != nil {
		return "", nil, fmt.Errorf("cannot evaluate column %s value: %s", colName, err.Error())
	}

	if colValAny == nil {
		return colName, nil, nil
	}

	internalColVal, err := sanitizeToInternalKnownType(colValAny, columnDefs[columnDefMap[colName]].columnType)
	if err != nil {
		return "", nil, fmt.Errorf("cannot cast column %s value (%v): %s", colName, colValAny, err.Error())
	}

	return colName, internalColVal, nil
}

func harvestInsertedPriKeyValuesFromAstExp(tableName string, columnDefs []*columnDef, columnDefMap map[string]int, exp ast.Expr, colValueMap map[string]any) error {
	switch typedExp := exp.(type) {
	// TODO: besides "partition_key = value AND clustering_column = value", we need to support:
	// partition_key = value AND clustering_column IN (value1, value2)
	// TOKEN(user_id) > TOKEN(100) AND TOKEN(user_id) < TOKEN(200)
	// support casting: user_id = CAST('101' AS uuid);
	// More rules:
	// You cannot use regular (non-indexed) columns in the WHERE clause unless you use ALLOW FILTERING
	// You cannot use > or < on a partition key without the TOKEN function.
	// No LIKE operator
	// Mandatory Primary Key: If you omit the WHERE clause, the update will fail, or if you use a partially defined primary key, it will throw an error
	case *ast.BinaryExpr:
		switch typedExp.Op {
		case token.LAND:
			if err := harvestInsertedPriKeyValuesFromAstExp(tableName, columnDefs, columnDefMap, typedExp.X, colValueMap); err != nil {
				return fmt.Errorf("error harvesting left side of AND: %s", err.Error())
			}
			if err := harvestInsertedPriKeyValuesFromAstExp(tableName, columnDefs, columnDefMap, typedExp.Y, colValueMap); err != nil {
				return fmt.Errorf("error harvesting right side of AND: %s", err.Error())
			}
		case token.EQL:
			colName, colVal, err := getInsertedPriKeyColumnValuePairFromEql(tableName, columnDefs, columnDefMap, *typedExp)
			if err != nil {
				return fmt.Errorf("cannot get column name value pair: %s", err.Error())
			}
			colValueMap[colName] = colVal
		default:
			return fmt.Errorf("cannot harvest, expected top-level AND or ==, got op %d", typedExp.Op)
		}
	default:
		return fmt.Errorf("cannot harvest, expected top-level AND or ==, got exp %T", typedExp)
	}
	return nil
}

// Convert "WHERE col1 == 'a' and col2 == 100` into col1:'a',col2:100
func getInsertedPriKeyValuesFromWhereClause(tableName string, columnDefs []*columnDef, columnDefMap map[string]int, whereExpAst ast.Expr) (map[string]any, error) {
	// 1. Detect all colX = exp fragments
	// 2. Ensure they are linked with AND
	// 3. Ensure the combined col1==... AND COL2==... is at the top of ast
	// 4. For each colX==... evaluate the exp and add it to the result map
	colValueMap := map[string]any{}
	if err := harvestInsertedPriKeyValuesFromAstExp(tableName, columnDefs, columnDefMap, whereExpAst, colValueMap); err != nil {
		return nil, fmt.Errorf("cannot obtain primary key values from WHERE expression: %s", err.Error())
	}
	return colValueMap, nil
}

func calcValuesToUpdate(cmd *CommandUpdate, columnDefs []*columnDef, columnDefMap map[string]int, valMap eval.VarValuesMap) (map[string]any, error) {
	var err error
	updatedNonKeyColValues := map[string]any{}
	for i, colSetExp := range cmd.ColumnSetExpressions {
		eCtx := eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, valMap)
		updatedNonKeyColValues[colSetExp.Name], err = eCtx.Eval(cmd.ColumnSetExpAsts[i])
		if err != nil {
			return nil, fmt.Errorf("cannot calculate updated column %d: %s", i, err.Error())
		}
		updatedNonKeyColValues[colSetExp.Name], err = sanitizeToInternalKnownType(updatedNonKeyColValues[colSetExp.Name], columnDefs[columnDefMap[colSetExp.Name]].columnType)
		if err != nil {
			return nil, fmt.Errorf("cannot cast updated column %d: %s", i, err.Error())
		}
	}
	return updatedNonKeyColValues, nil
}

func (t *tableStore) execUpdate(cmd *CommandUpdate, preparedQueryParams []interface{}) (bool, []gocql.ColumnInfo, [][]any, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, tableColDef := range t.columnDefs {
		if tableColDef.primaryKey == PrimaryKeyPartition || tableColDef.primaryKey == PrimaryKeyClustering {
			for _, colSetExp := range cmd.ColumnSetExpressions {
				if colSetExp.Name == tableColDef.name {
					return false, nil, nil, fmt.Errorf("partition/clustering column key %s cannot be modified in the update command", tableColDef.name)
				}
			}
		}
	}

	var err error

	valMap := eval.VarValuesMap{}
	valMap[""] = map[string]any{}
	valMap[cmd.TableName] = map[string]any{}
	var isAlreadyExists bool
	for i := range len(t.columnValues[0]) {
		clear(valMap[""])
		clear(valMap[cmd.TableName])
		for colIdx, colDef := range t.columnDefs {
			valMap[""][colDef.name] = t.columnValues[colIdx][i]
			valMap[cmd.TableName][colDef.name] = t.columnValues[colIdx][i]
		}

		// Add prepared params to the value map that is used for where and for update values, see below
		if err = addPreparedQueryParamsToMap(valMap, preparedQueryParams); err != nil {
			return false, nil, nil, fmt.Errorf("cannot apply prepared params: %s", err.Error())
		}

		isUpdate := true

		if cmd.WhereExpAst != nil {
			eCtx := eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, valMap)
			isUpdateAnyFromWhere, err := eCtx.Eval(cmd.WhereExpAst)
			if err != nil {
				return false, nil, nil, err
			}

			var ok bool
			isUpdate, ok = isUpdateAnyFromWhere.(bool)
			if !ok {
				return false, nil, nil, fmt.Errorf("where expressions return %T, expected bool", isUpdateAnyFromWhere)
			}

			if isUpdate {
				isAlreadyExists = true
			}
		}

		if isUpdate && cmd.IfExpAst != nil {
			eCtx := eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, valMap)
			isUpdateAnyFromIf, err := eCtx.Eval(cmd.IfExpAst)
			if err != nil {
				return false, nil, nil, err
			}

			isUpdateFromIf, ok := isUpdateAnyFromIf.(bool)
			if !ok {
				return false, nil, nil, fmt.Errorf("where expressions return %T, expected bool", isUpdateAnyFromIf)
			}

			if !isUpdateFromIf {
				// Last minute IF <condition> says we should not update
				isUpdate = false
			}
		}

		if isUpdate {
			for _, colSetExp := range cmd.ColumnSetExpressions {

				// We cannot calculate values in advance: b = b + 1 expressions are allowed, so do it here
				updatedNonKeyColValues, err := calcValuesToUpdate(cmd, t.columnDefs, t.columnDefMap, valMap)
				if err != nil {
					return false, nil, nil, fmt.Errorf("cannot calculate update value: %s", err.Error())

				}

				colDefIdx, ok := t.columnDefMap[colSetExp.Name]
				if !ok {
					return false, nil, nil, fmt.Errorf("cannot update column %s, it is not in the table definition", colSetExp.Name)
				}
				t.columnValues[colDefIdx][i] = updatedNonKeyColValues[colSetExp.Name]
			}
		}
	}

	if isAlreadyExists {
		// Should we return old content of the updated raw? I do not think so.
		return true, nil, nil, nil
	}

	if cmd.IfExists {
		return false, nil, nil, nil
	}

	// UPSERT

	insertCmd := CommandInsert{
		CtxKeyspace: cmd.CtxKeyspace,
		TableName:   cmd.TableName,
		ColumnNames: make([]string, 0),
		//ColumnValues: make([]*Lexem, 0),
		IfNotExists: false, // We know it does not exist, this is why update became upsert
	}

	// Primary key columns must be set, we have to convert "WHERE col1 = 'a' and col2 = 100` into col1:'a',col2:100
	// Cassandra supports more options (see TODO in harvestInsertedPriKeyValuesFromAstExp), but that's a lot to implement at the moment
	allInsertedColValues, err := getInsertedPriKeyValuesFromWhereClause(cmd.TableName, t.columnDefs, t.columnDefMap, cmd.WhereExpAst)
	if err != nil {
		return false, nil, nil, err
	}

	// Prepare all values, we do not need any existing column data here, but take care of NULL "count" fields
	clear(valMap[""])
	clear(valMap[cmd.TableName])
	for _, colDef := range t.columnDefs {
		if colDef.columnType == gocql.TypeCounter {
			valMap[""][colDef.name] = int64(0)
			valMap[cmd.TableName][colDef.name] = int64(0)
		}
	}

	insertedNonKeyColValues, err := calcValuesToUpdate(cmd, t.columnDefs, t.columnDefMap, valMap)
	if err != nil {
		return false, nil, nil, fmt.Errorf("cannot calculate insert value: %s", err.Error())
	}

	// Add non-primary columns to the map
	for colName, val := range insertedNonKeyColValues {
		allInsertedColValues[colName] = val
	}

	insertedColCount := 0
	for insertedColName, insertedColValue := range allInsertedColValues {
		insertCmd.ColumnNames = append(insertCmd.ColumnNames, insertedColName)
		insertCmd.ColumnValues = append(insertCmd.ColumnValues, insertedColValue)
		insertedColCount++
	}
	return t.execInternalUpsert(&insertCmd)
}

func (t *tableStore) execDelete(cmd *CommandDelete, preparedQueryParams []interface{}) (bool, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	var isApplied bool
	valMap := eval.VarValuesMap{}
	valMap[""] = map[string]any{}
	valMap[cmd.TableName] = map[string]any{}
	rowsToDelete := []int{}
	for i := range len(t.columnValues[0]) {
		clear(valMap[""])
		clear(valMap[cmd.TableName])
		for colIdx, colDef := range t.columnDefs {
			valMap[""][colDef.name] = t.columnValues[colIdx][i]
			valMap[cmd.TableName][colDef.name] = t.columnValues[colIdx][i]
		}

		// Add prepared params to the value map that is used for where and for select values, see below
		if err := addPreparedQueryParamsToMap(valMap, preparedQueryParams); err != nil {
			return false, fmt.Errorf("cannot apply prepared params: %s", err.Error())
		}

		isInclude := true
		var ok bool
		if cmd.WhereExpAst != nil {
			eCtx := eval.NewPlainEvalCtx(GocqlmemEvalFunctions, GocqlmemEvalConstants, valMap)
			isIncludeAny, err := eCtx.Eval(cmd.WhereExpAst)
			if err != nil {
				return false, fmt.Errorf("cannot evaluate where expression: %s", err.Error())
			}

			isInclude, ok = isIncludeAny.(bool)
			if !ok {
				return false, fmt.Errorf("where expressions return %T, expected bool", isIncludeAny)
			}
		}

		if isInclude {
			if len(cmd.ColumnsToDelete) == 0 {
				isApplied = true
				rowsToDelete = append(rowsToDelete, i)
			} else {
				for _, colNameToDelete := range cmd.ColumnsToDelete {
					colIdxToDelete, ok := t.columnDefMap[colNameToDelete]
					if !ok {
						return false, fmt.Errorf("cannot find column to delete: %s", colNameToDelete)
					}
					if t.columnDefs[colIdxToDelete].primaryKey != PrimaryKeyNone {
						return false, fmt.Errorf("cannot delete key column value: %s", colNameToDelete)
					}
					isApplied = true
					t.columnValues[colIdxToDelete][i] = nil
				}
			}
		}
	}

	if len(rowsToDelete) > 0 {
		for colIdx := range len(t.columnValues) {
			for i := len(rowsToDelete) - 1; i >= 0; i-- {
				t.columnValues[colIdx] = slices.Delete(t.columnValues[colIdx], i, i+1)
			}
		}
	}

	return isApplied, nil
}
