package proc

import (
	"fmt"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/shopspring/decimal"
	"gopkg.in/inf.v0"
)

type Rowset struct {
	Fields                []sc.FieldRef
	FieldsByFullAliasName map[string]int
	FieldsByFieldName     map[string]int
	Rows                  []*[]interface{}
	RowCount              int
}

func NewRowsetFromFieldRefs(fieldRefsList ...sc.FieldRefs) *Rowset {
	rs := Rowset{}
	for i := 0; i < len(fieldRefsList); i++ {
		rs.AppendFieldRefs(&fieldRefsList[i])
	}
	return &rs
}

func (rs *Rowset) ToString() string {
	var b strings.Builder
	for _, fr := range rs.Fields {
		b.WriteString(fmt.Sprintf("%30s", fr.GetAliasHash()))
	}
	b.WriteString("\n")
	for rowIdx := 0; rowIdx < rs.RowCount; rowIdx++ {
		vals := rs.Rows[rowIdx]
		for _, val := range *vals {
			switch typedVal := val.(type) {
			case *int64:
				b.WriteString(fmt.Sprintf("%30d", *typedVal))
			case *float64:
				b.WriteString(fmt.Sprintf("%30f", *typedVal))
			case *string:
				b.WriteString(fmt.Sprintf("\"%30s\"", *typedVal))
			case *bool:
				if *typedVal {
					return "                          TRUE"
				} else {
					return "                         FALSE"
				}
			case *decimal.Decimal:
				b.WriteString(fmt.Sprintf("%30s", (*typedVal).String()))
			case *time.Time:
				b.WriteString(fmt.Sprintf("%30s", (*typedVal).Format("\"2006-01-02T15:04:05.000-0700\"")))
			default:
				b.WriteString("bla")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (rs *Rowset) ArrangeByRowid(rowids []int64) error {
	if len(rowids) < rs.RowCount {
		return fmt.Errorf("invalid rowid array length")
	}

	rowidColIdx := rs.FieldsByFieldName["rowid"]

	// Build a map for quicker access
	rowMap := map[int64]int{}
	for rowIdx := 0; rowIdx < rs.RowCount; rowIdx++ {
		rowid := *((*rs.Rows[rowIdx])[rowidColIdx]).(*int64)
		rowMap[rowid] = rowIdx
	}

	for i := 0; i < rs.RowCount; i++ {
		// rowids[i] must be at i-th position in rs.Rows
		if rowMap[rowids[i]] != i {
			// Swap
			tailIdx := rowMap[rowids[i]]
			tailRowPtr := rs.Rows[tailIdx]
			headRowid := *((*rs.Rows[i])[rowidColIdx]).(*int64)

			// Move rs.Rows[i] to the tail of rs.Rows
			rs.Rows[tailIdx] = rs.Rows[i]
			rowMap[headRowid] = tailIdx

			// Move tail row to the i-th position
			rs.Rows[i] = tailRowPtr
			rowMap[rowids[i]] = i // As it should be
		}
	}

	return nil
}

func (rs *Rowset) GetFieldNames() *[]string {
	fieldNames := make([]string, len(rs.Fields))
	for colIdx := 0; colIdx < len(rs.Fields); colIdx++ {
		fieldNames[colIdx] = rs.Fields[colIdx].FieldName
	}
	return &fieldNames
}
func (rs *Rowset) AppendFieldRefs(fieldRefs *sc.FieldRefs) {
	rs.AppendFieldRefsWithFilter(fieldRefs, "")
}

func (rs *Rowset) AppendFieldRefsWithFilter(fieldRefs *sc.FieldRefs, tableFilter string) {
	if rs.Fields == nil {
		rs.Fields = make([]sc.FieldRef, 0)
	}
	if rs.FieldsByFullAliasName == nil {
		rs.FieldsByFullAliasName = map[string]int{}
	}
	if rs.FieldsByFieldName == nil {
		rs.FieldsByFieldName = map[string]int{}
	}

	for i := 0; i < len(*fieldRefs); i++ {
		if len(tableFilter) > 0 && (*fieldRefs)[i].TableName != tableFilter {
			continue
		}
		key := (*fieldRefs)[i].GetAliasHash()
		if _, ok := rs.FieldsByFullAliasName[key]; !ok {
			rs.Fields = append(rs.Fields, (*fieldRefs)[i])
			rs.FieldsByFullAliasName[key] = len(rs.Fields) - 1
			rs.FieldsByFieldName[(*fieldRefs)[i].FieldName] = len(rs.Fields) - 1
		}
	}
}

// func (rs *Rowset) InitRowsWithTableCreatorDefaults(capacity int, writerFieldDefs map[string]*sc.WriteTableFieldDef) error {
// 	if rs.Rows == nil || len(rs.Rows) != capacity {
// 		rs.Rows = make([](*[]interface{}), capacity)
// 	}
// 	for rowIdx := 0; rowIdx < capacity; rowIdx++ {
// 		newRow := make([]interface{}, len(rs.Fields))
// 		rs.Rows[rowIdx] = &newRow
// 		for colIdx := 0; colIdx < len(rs.Fields); colIdx++ {
// 			fieldName := rs.Fields[colIdx].FieldName
// 			var defaultValueString string
// 			if len(writerFieldDefs) > 0 {
// 				writerFieldDef, ok := writerFieldDefs[fieldName]
// 				if ok {
// 					defaultValueString = strings.TrimSpace(writerFieldDef.DefaultValue)
// 				}
// 			}
// 			var err error
// 			switch rs.Fields[colIdx].FieldType {
// 			case sc.FieldTypeInt:
// 				v := sc.DefaultInt
// 				if len(defaultValueString) > 0 {
// 					v, err = strconv.ParseInt(defaultValueString, 10, 64)
// 					if err != nil {
// 						return fmt.Errorf("cannot read int64 field %s from default value string '%s': %s", fieldName, defaultValueString, err.Error())
// 					}
// 				}
// 				(*rs.Rows[rowIdx])[colIdx] = &v
// 			case sc.FieldTypeFloat:
// 				v := sc.DefaultFloat
// 				if len(defaultValueString) > 0 {
// 					v, err = strconv.ParseFloat(defaultValueString, 64)
// 					if err != nil {
// 						return fmt.Errorf("cannot read float64 field %s from default value string '%s': %s", fieldName, defaultValueString, err.Error())
// 					}
// 				}
// 				(*rs.Rows[rowIdx])[colIdx] = &v
// 			case sc.FieldTypeString:
// 				v := sc.DefaultString
// 				if len(defaultValueString) > 0 {
// 					v = defaultValueString
// 				}
// 				(*rs.Rows[rowIdx])[colIdx] = &v
// 			case sc.FieldTypeDecimal2:
// 				// Set it to Cassandra-accepted value, not decimal.Decimal: https://github.com/gocql/gocql/issues/1578
// 				v := inf.NewDec(0, 0)
// 				if len(defaultValueString) > 0 {
// 					f, err := strconv.ParseFloat(defaultValueString, 64)
// 					if err != nil {
// 						return fmt.Errorf("cannot read decimal2 field %s from default value string '%s': %s", fieldName, defaultValueString, err.Error())
// 					}
// 					scaled := int64(math.Round(f * 100))
// 					v = inf.NewDec(scaled, 2)
// 				}
// 				(*rs.Rows[rowIdx])[colIdx] = v
// 			case sc.FieldTypeBool:
// 				v := sc.DefaultBool
// 				if len(defaultValueString) > 0 {
// 					v, err = strconv.ParseBool(defaultValueString)
// 					if err != nil {
// 						return fmt.Errorf("cannot read bool field %s, from default value string '%s', allowed values are true,false,T,F,0,1: %s", fieldName, defaultValueString, err.Error())
// 					}
// 				}
// 				(*rs.Rows[rowIdx])[colIdx] = &v
// 			case sc.FieldTypeDateTime:
// 				v := sc.DefaultDateTime()
// 				if len(defaultValueString) > 0 {
// 					v, err = time.Parse(sc.CassandraDatetimeFormat, defaultValueString)
// 					if err != nil {
// 						return fmt.Errorf("cannot read time field %s from default value string '%s': %s", fieldName, defaultValueString, err.Error())
// 					}
// 				}
// 				(*rs.Rows[rowIdx])[colIdx] = &v
// 			default:
// 				return fmt.Errorf("InitRows unsupported field type %s, field %s.%s", rs.Fields[colIdx].FieldType, rs.Fields[colIdx].TableName, fieldName)
// 			}
// 		}
// 	}
// 	return nil

// }

func (rs *Rowset) InitRows(capacity int) error {
	if rs.Rows == nil || len(rs.Rows) != capacity {
		rs.Rows = make([](*[]interface{}), capacity)
	}
	for rowIdx := 0; rowIdx < capacity; rowIdx++ {
		newRow := make([]interface{}, len(rs.Fields))
		rs.Rows[rowIdx] = &newRow
		for colIdx := 0; colIdx < len(rs.Fields); colIdx++ {
			switch rs.Fields[colIdx].FieldType {
			case sc.FieldTypeInt:
				v := int64(0)
				(*rs.Rows[rowIdx])[colIdx] = &v
			case sc.FieldTypeFloat:
				v := float64(0.0)
				(*rs.Rows[rowIdx])[colIdx] = &v
			case sc.FieldTypeString:
				v := ""
				(*rs.Rows[rowIdx])[colIdx] = &v
			case sc.FieldTypeDecimal2:
				// Set it to Cassandra-accepted value, not decimal.Decimal: https://github.com/gocql/gocql/issues/1578
				(*rs.Rows[rowIdx])[colIdx] = inf.NewDec(0, 0)
			case sc.FieldTypeBool:
				v := false
				(*rs.Rows[rowIdx])[colIdx] = &v
			case sc.FieldTypeDateTime:
				v := time.Date(1901, 1, 1, 0, 0, 0, 0, time.UTC)
				(*rs.Rows[rowIdx])[colIdx] = &v
			default:
				return fmt.Errorf("InitRows unsupported field type %s, field %s.%s", rs.Fields[colIdx].FieldType, rs.Fields[colIdx].TableName, rs.Fields[colIdx].FieldName)
			}
		}
	}
	return nil
}
func (rs *Rowset) ExportToVars(rowIdx int, vars *eval.VarValuesMap) error {
	return rs.ExportToVarsWithAlias(rowIdx, vars, "")
}

func (rs *Rowset) GetTableRecord(rowIdx int) (map[string]interface{}, error) {
	tableRecord := map[string]interface{}{}
	for colIdx := 0; colIdx < len(rs.Fields); colIdx++ {
		fName := rs.Fields[colIdx].FieldName
		valuePtr := (*rs.Rows[rowIdx])[rs.FieldsByFieldName[fName]]
		switch assertedValuePtr := valuePtr.(type) {
		case *int64:
			tableRecord[fName] = *assertedValuePtr
		case *string:
			tableRecord[fName] = *assertedValuePtr
		case *time.Time:
			tableRecord[fName] = *assertedValuePtr
		case *bool:
			tableRecord[fName] = *assertedValuePtr
		case *decimal.Decimal:
			tableRecord[fName] = *assertedValuePtr
		case *float64:
			tableRecord[fName] = *assertedValuePtr
		case *inf.Dec:
			decVal, err := decimal.NewFromString((*(valuePtr.(*inf.Dec))).String())
			if err != nil {
				return nil, fmt.Errorf("GetTableRecord cannot convert inf.Dec [%v]to decimal.Decimal", *(valuePtr.(*inf.Dec)))
			}
			tableRecord[fName] = decVal
		default:
			return nil, fmt.Errorf("GetTableRecord unsupported field type %T", valuePtr)
		}
	}
	return tableRecord, nil
}

func (rs *Rowset) ExportToVarsWithAlias(rowIdx int, vars *eval.VarValuesMap, useTableAlias string) error {
	for colIdx := 0; colIdx < len(rs.Fields); colIdx++ {
		tName := &rs.Fields[colIdx].TableName
		if len(useTableAlias) > 0 {
			tName = &useTableAlias
		}
		fName := &rs.Fields[colIdx].FieldName
		_, ok := (*vars)[*tName]
		if !ok {
			(*vars)[*tName] = map[string]interface{}{}
		}
		valuePtr := (*rs.Rows[rowIdx])[colIdx]
		switch assertedValuePtr := valuePtr.(type) {
		case *int64:
			(*vars)[*tName][*fName] = *assertedValuePtr
		case *string:
			(*vars)[*tName][*fName] = *assertedValuePtr
		case *time.Time:
			(*vars)[*tName][*fName] = *assertedValuePtr
		case *bool:
			(*vars)[*tName][*fName] = *assertedValuePtr
		case *decimal.Decimal:
			(*vars)[*tName][*fName] = *assertedValuePtr
		case *float64:
			(*vars)[*tName][*fName] = *assertedValuePtr
		case *inf.Dec:
			decVal, err := decimal.NewFromString((*(valuePtr.(*inf.Dec))).String())
			if err != nil {
				return fmt.Errorf("ExportToVars cannot convert inf.Dec [%v]to decimal.Decimal", *(valuePtr.(*inf.Dec)))
			}
			(*vars)[*tName][*fName] = decVal
		default:
			return fmt.Errorf("ExportToVars unsupported field type %T", valuePtr)
		}
	}
	return nil
}
