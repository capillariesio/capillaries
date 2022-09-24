package sc

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval"
	"gopkg.in/inf.v0"
)

const ProhibitedTableNameRegex = "^idx|^wf|^system"
const AllowedTableNameRegex = "[A-Za-z0-9_]+"
const AllowedIdxNameRegex = "^idx[A-Za-z0-9_]+"

type TableUpdaterDef struct {
	Fields map[string]*WriteTableFieldDef `json:"fields"`
}

type TableCreatorDef struct {
	Name                          string `json:"name"`
	RawHaving                     string `json:"having"`
	Having                        ast.Expr
	UsedInHavingFields            FieldRefs
	UsedInTargetExpressionsFields FieldRefs
	Fields                        map[string]*WriteTableFieldDef `json:"fields"`
	RawIndexes                    map[string]string              `json:"indexes"`
	Indexes                       IdxDefMap
}

// func (fieldDef *WriteTableFieldDef) CheckValueType(val interface{}) error {
// 	switch assertedValue := val.(type) {
// 	case int64:
// 		if fieldDef.Type != FieldTypeInt {
// 			return fmt.Errorf("expected type %s, but got int64 (%d)", fieldDef.Type, assertedValue)
// 		}
// 	case float64:
// 		if fieldDef.Type != FieldTypeFloat {
// 			return fmt.Errorf("expected type %s, but got float64 (%f)", fieldDef.Type, assertedValue)
// 		}
// 	case string:
// 		if fieldDef.Type != FieldTypeString {
// 			return fmt.Errorf("expected type %s, but got string (%s)", fieldDef.Type, assertedValue)
// 		}
// 	case bool:
// 		if fieldDef.Type != FieldTypeBool {
// 			return fmt.Errorf("expected type %s, but got bool (%v)", fieldDef.Type, assertedValue)
// 		}
// 	case time.Time:
// 		if fieldDef.Type != FieldTypeDateTime {
// 			return fmt.Errorf("expected type %s, but got datetime (%s)", fieldDef.Type, assertedValue.String())
// 		}
// 	case decimal.Decimal:
// 		if fieldDef.Type != FieldTypeDecimal2 {
// 			return fmt.Errorf("expected type %s, but got decimal (%s)", fieldDef.Type, assertedValue.String())
// 		}
// 	default:
// 		return fmt.Errorf("expected type %s, but got unexpected type %T(%v)", fieldDef.Type, assertedValue, assertedValue)
// 	}
// 	return nil
// }

func (tcDef *TableCreatorDef) GetFieldRefs() *FieldRefs {
	return tcDef.GetFieldRefsWithAlias("")
}

func (tcDef *TableCreatorDef) GetFieldRefsWithAlias(useTableAlias string) *FieldRefs {
	fieldRefs := make(FieldRefs, len(tcDef.Fields))
	i := 0
	for fieldName, fieldDef := range tcDef.Fields {
		tName := tcDef.Name
		if len(useTableAlias) > 0 {
			tName = useTableAlias
		}
		fieldRefs[i] = FieldRef{
			TableName: tName,
			FieldName: fieldName,
			FieldType: fieldDef.Type}
		i += 1
	}
	return &fieldRefs
}

func (tcDef *TableCreatorDef) Deserialize(rawWriter json.RawMessage) error {
	var err error
	if err = json.Unmarshal(rawWriter, tcDef); err != nil {
		return fmt.Errorf("cannot unmarshal table creator: %s", err.Error())
	}

	re := regexp.MustCompile(ProhibitedTableNameRegex)
	invalidNamePieceFound := re.FindString(tcDef.Name)
	if len(invalidNamePieceFound) > 0 {
		return fmt.Errorf("invalid table name [%s]: prohibited regex is [%s]", tcDef.Name, ProhibitedTableNameRegex)
	}

	re = regexp.MustCompile(AllowedTableNameRegex)
	invalidNamePieceFound = re.FindString(tcDef.Name)
	if len(invalidNamePieceFound) != len(tcDef.Name) {
		return fmt.Errorf("invalid table name [%s]: allowed regex is [%s]", tcDef.Name, AllowedTableNameRegex)
	}

	// Having
	tcDef.Having, err = ParseRawGolangExpressionStringAndHarvestFieldRefs(tcDef.RawHaving, &tcDef.UsedInHavingFields)
	if err != nil {
		return fmt.Errorf("cannot parse table creator 'having' condition [%s]: [%s]", tcDef.RawHaving, err.Error())
	}

	// Fields
	for _, fieldDef := range tcDef.Fields {
		if fieldDef.ParsedExpression, err = ParseRawGolangExpressionStringAndHarvestFieldRefs(fieldDef.RawExpression, &fieldDef.UsedFields); err != nil {
			return fmt.Errorf("cannot parse field expression [%s]: [%s]", fieldDef.RawExpression, err.Error())
		}
		if !IsValidFieldType(fieldDef.Type) {
			return fmt.Errorf("invalid field type [%s]", fieldDef.Type)
		}
	}

	tcDef.UsedInTargetExpressionsFields = GetFieldRefsUsedInAllTargetExpressions(tcDef.Fields)

	// Indexes
	tcDef.Indexes = IdxDefMap{}
	if err := tcDef.Indexes.parseRawIndexDefMap(tcDef.RawIndexes, tcDef.GetFieldRefs()); err != nil {
		return err
	}

	re = regexp.MustCompile(AllowedIdxNameRegex)
	for idxName, _ := range tcDef.Indexes {
		invalidNamePieceFound := re.FindString(idxName)
		if len(invalidNamePieceFound) != len(idxName) {
			return fmt.Errorf("invalid index name [%s]: allowed regex is [%s]", idxName, AllowedIdxNameRegex)
		}

	}

	return nil
}

func (creatorDef *TableCreatorDef) GetFieldDefaultReadyForDb(fieldName string) (interface{}, error) {
	writerFieldDef, ok := creatorDef.Fields[fieldName]
	if !ok {
		return nil, fmt.Errorf("unknown field %s", fieldName)
	}
	defaultValueString := strings.TrimSpace(writerFieldDef.DefaultValue)

	var err error
	switch writerFieldDef.Type {
	case FieldTypeInt:
		v := DefaultInt
		if len(defaultValueString) > 0 {
			v, err = strconv.ParseInt(defaultValueString, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot read int64 field %s from default value string '%s': %s", fieldName, defaultValueString, err.Error())
			}
		}
		return v, nil
	case FieldTypeFloat:
		v := DefaultFloat
		if len(defaultValueString) > 0 {
			v, err = strconv.ParseFloat(defaultValueString, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot read float64 field %s from default value string '%s': %s", fieldName, defaultValueString, err.Error())
			}
		}
		return v, nil
	case FieldTypeString:
		v := DefaultString
		if len(defaultValueString) > 0 {
			v = defaultValueString
		}
		return v, nil
	case FieldTypeDecimal2:
		// Set it to Cassandra-accepted value, not decimal.Decimal: https://github.com/gocql/gocql/issues/1578
		v := inf.NewDec(0, 0)
		if len(defaultValueString) > 0 {
			f, err := strconv.ParseFloat(defaultValueString, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot read decimal2 field %s from default value string '%s': %s", fieldName, defaultValueString, err.Error())
			}
			scaled := int64(math.Round(f * 100))
			v = inf.NewDec(scaled, 2)
		}
		return v, nil
	case FieldTypeBool:
		v := DefaultBool
		if len(defaultValueString) > 0 {
			v, err = strconv.ParseBool(defaultValueString)
			if err != nil {
				return nil, fmt.Errorf("cannot read bool field %s, from default value string '%s', allowed values are true,false,T,F,0,1: %s", fieldName, defaultValueString, err.Error())
			}
		}
		return v, nil
	case FieldTypeDateTime:
		v := DefaultDateTime()
		if len(defaultValueString) > 0 {
			v, err = time.Parse(CassandraDatetimeFormat, defaultValueString)
			if err != nil {
				return nil, fmt.Errorf("cannot read time field %s from default value string '%s': %s", fieldName, defaultValueString, err.Error())
			}
		}
		return v, nil
	default:
		return nil, fmt.Errorf("GetFieldDefault unsupported field type %s, field %s", writerFieldDef.Type, fieldName)
	}
}

func CalculateFieldValue(fieldName string, fieldDef *WriteTableFieldDef, srcVars eval.VarValuesMap, canUseAggFunc bool) (interface{}, error) {
	eCtx := eval.NewPlainEvalCtxWithVars(canUseAggFunc && eval.IsRootAggFunc(fieldDef.ParsedExpression), &srcVars)
	valVolatile, err := eCtx.Eval(fieldDef.ParsedExpression)
	if err != nil {
		return nil, fmt.Errorf("cannot evaluate expression for field %s: [%s]", fieldName, err.Error())
	} else {
		if err := CheckValueType(valVolatile, fieldDef.Type); err != nil {
			return nil, fmt.Errorf("invalid field %s type: [%s]", fieldName, err.Error())
		} else {
			return valVolatile, nil
		}
	}
}

func (creatorDef *TableCreatorDef) CalculateTableRecordFromSrcVars(canUseAggFunc bool, srcVars eval.VarValuesMap) (map[string]interface{}, error) {
	errors := make([]string, 0, 2)

	tableRecord := map[string]interface{}{}

	for fieldName, fieldDef := range creatorDef.Fields {
		var err error
		tableRecord[fieldName], err = CalculateFieldValue(fieldName, fieldDef, srcVars, canUseAggFunc)
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf(strings.Join(errors, "; "))
	} else {
		return tableRecord, nil
	}
}

func (creatorDef *TableCreatorDef) CheckTableRecordHavingCondition(tableRecord map[string]interface{}) (bool, error) {
	if creatorDef.Having == nil {
		// No Having condition specified
		return true, nil
	}
	vars := eval.VarValuesMap{}
	vars[CreatorAlias] = map[string]interface{}{}
	for fieldName, fieldValue := range tableRecord {
		vars[CreatorAlias][fieldName] = fieldValue
	}

	eCtx := eval.NewPlainEvalCtxWithVars(false, &vars)
	valVolatile, err := eCtx.Eval(creatorDef.Having)
	if err != nil {
		return false, fmt.Errorf("cannot evaluate 'having' expression: [%s]", err.Error())
	}
	valBool, ok := valVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot get bool when evaluating having expression, got %v(%T) instead", valVolatile, valVolatile)
	}

	return valBool, nil
}
