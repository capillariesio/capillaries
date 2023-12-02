package sc

import (
	"fmt"
	"go/ast"
	"go/parser"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
)

type FieldRef struct {
	TableName string
	FieldName string
	FieldType TableFieldType
}

func (fr *FieldRef) GetAliasHash() string {
	return fmt.Sprintf("%s.%s", fr.TableName, fr.FieldName)
}

type FieldRefs []FieldRef

func (fieldRefs *FieldRefs) HasFieldsWithTableAlias(tableAlias string) bool {
	for i := 0; i < len(*fieldRefs); i++ {
		if tableAlias == (*fieldRefs)[i].TableName {
			return true
		}
	}
	return false
}

func JoinFieldRefs(srcFieldRefs ...*FieldRefs) *FieldRefs {
	hashes := map[string](*FieldRef){}
	for i := 0; i < len(srcFieldRefs); i++ {
		if srcFieldRefs[i] != nil {
			for j := 0; j < len(*srcFieldRefs[i]); j++ {
				hash := fmt.Sprintf("%s.%s", (*srcFieldRefs[i])[j].TableName, (*srcFieldRefs[i])[j].FieldName)
				if _, ok := hashes[hash]; !ok {
					hashes[hash] = &(*srcFieldRefs[i])[j]
				}
			}
		}
	}

	fieldRefs := make(FieldRefs, len(hashes))
	fieldRefCount := 0
	for _, fieldRef := range hashes {
		fieldRefs[fieldRefCount] = *fieldRef
		fieldRefCount++
	}
	return &fieldRefs
}

func RowidFieldRef(tableName string) FieldRef {
	return FieldRef{
		TableName: tableName,
		FieldName: "rowid",
		FieldType: FieldTypeInt}
}

func RowidTokenFieldRef() FieldRef {
	return FieldRef{
		TableName: "db_system",
		FieldName: "token(rowid)",
		FieldType: FieldTypeInt}
}

// func RunBatchRowidTokenFieldRef() FieldRef {
// 	return FieldRef{
// 		TableName: "db_system",
// 		FieldName: "token(run_id,batch_idx,rowid)",
// 		FieldType: FieldTypeInt}
// }

func KeyTokenFieldRef() FieldRef {
	return FieldRef{
		TableName: "db_system",
		FieldName: "token(key)",
		FieldType: FieldTypeInt}
}

//	func RunBatchKeyTokenFieldRef() FieldRef {
//		return FieldRef{
//			TableName: "db_system",
//			FieldName: "token(run_id,batch_idx,key)",
//			FieldType: FieldTypeInt}
//	}
func IdxKeyFieldRef() FieldRef {
	return FieldRef{
		TableName: "db_system",
		FieldName: "key",
		FieldType: FieldTypeString}
}

func (fieldRefs *FieldRefs) contributeUnresolved(tableName string, fieldName string) {
	// Check if it's already there
	for i := 0; i < len(*fieldRefs); i++ {
		if (*fieldRefs)[i].TableName == tableName &&
			(*fieldRefs)[i].FieldName == fieldName {
			// Already there
			return
		}
	}

	*fieldRefs = append(*fieldRefs, FieldRef{TableName: tableName, FieldName: fieldName, FieldType: FieldTypeUnknown})
}

func (fieldRefs *FieldRefs) Append(otherFieldRefs FieldRefs) {
	fieldRefs.AppendWithFilter(otherFieldRefs, "")
}

func (fieldRefs *FieldRefs) AppendWithFilter(otherFieldRefs FieldRefs, tableFilter string) {
	fieldRefMap := map[string]FieldRef{}

	// Existing to map
	for i := 0; i < len(*fieldRefs); i++ {
		fieldRefMap[(*fieldRefs)[i].GetAliasHash()] = (*fieldRefs)[i]
	}

	// New to map
	for i := 0; i < len(otherFieldRefs); i++ {
		if len(tableFilter) == 0 || tableFilter == (otherFieldRefs)[i].TableName {
			fieldRefMap[(otherFieldRefs)[i].GetAliasHash()] = (otherFieldRefs)[i]
		}
	}

	*fieldRefs = make([]FieldRef, len(fieldRefMap))
	refIdx := 0
	for fieldRefHash := range fieldRefMap {
		(*fieldRefs)[refIdx] = fieldRefMap[fieldRefHash]
		refIdx++
	}
}

func evalExpressionWithFieldRefsAndCheckType(exp ast.Expr, fieldRefs FieldRefs, expectedType TableFieldType) error {
	if exp == nil {
		// Nothing to evaluate
		return nil
	}
	varValuesMap := eval.VarValuesMap{}
	for i := 0; i < len(fieldRefs); i++ {
		tName := fieldRefs[i].TableName
		fName := fieldRefs[i].FieldName
		fType := fieldRefs[i].FieldType
		if _, ok := varValuesMap[tName]; !ok {
			varValuesMap[tName] = map[string]any{}
		}
		switch fType {
		case FieldTypeInt:
			varValuesMap[tName][fName] = int64(0)
		case FieldTypeFloat:
			varValuesMap[tName][fName] = float64(0.0)
		case FieldTypeBool:
			varValuesMap[tName][fName] = false
		case FieldTypeString:
			varValuesMap[tName][fName] = "12345.67" // There may be a float() call out there
		case FieldTypeDateTime:
			varValuesMap[tName][fName] = time.Now()
		case FieldTypeDecimal2:
			varValuesMap[tName][fName] = decimal.NewFromFloat(2.34)
		default:
			return fmt.Errorf("evalExpressionWithFieldRefsAndCheckType unsupported field type %s", fieldRefs[i].FieldType)

		}
	}

	aggFuncEnabled, aggFuncType, aggFuncArgs := eval.DetectRootAggFunc(exp)
	eCtx, err := eval.NewPlainEvalCtxWithVarsAndInitializedAgg(aggFuncEnabled, &varValuesMap, aggFuncType, aggFuncArgs)
	if err != nil {
		return err
	}

	result, err := eCtx.Eval(exp)
	if err != nil {
		return err
	}

	return CheckValueType(result, expectedType)
}

func (fieldRefs *FieldRefs) FindByFieldName(fieldName string) (*FieldRef, bool) {
	for i := 0; i < len(*fieldRefs); i++ {
		if fieldName == (*fieldRefs)[i].FieldName {
			return &(*fieldRefs)[i], true
		}
	}
	return nil, false
}

func checkAllowed(fieldRefsToCheck *FieldRefs, prohibitedFieldRefs *FieldRefs, allowedFieldRefs *FieldRefs) error {
	if fieldRefsToCheck == nil {
		return nil
	}

	// Harvest allowed
	allowedHashes := map[string](*FieldRef){}
	if allowedFieldRefs != nil {
		for i := 0; i < len(*allowedFieldRefs); i++ {
			hash := fmt.Sprintf("%s.%s", (*allowedFieldRefs)[i].TableName, (*allowedFieldRefs)[i].FieldName)
			if _, ok := allowedHashes[hash]; !ok {
				allowedHashes[hash] = &(*allowedFieldRefs)[i]
			}
		}
	}

	// Harvest prohibited
	prohibitedHashes := map[string](*FieldRef){}
	if prohibitedFieldRefs != nil {
		for i := 0; i < len(*prohibitedFieldRefs); i++ {
			hash := fmt.Sprintf("%s.%s", (*prohibitedFieldRefs)[i].TableName, (*prohibitedFieldRefs)[i].FieldName)
			if _, ok := prohibitedHashes[hash]; !ok {
				prohibitedHashes[hash] = &(*prohibitedFieldRefs)[i]
			}
		}
	}

	errors := make([]string, 0, 2)

	for i := 0; i < len(*fieldRefsToCheck); i++ {
		if len((*fieldRefsToCheck)[i].TableName) == 0 || len((*fieldRefsToCheck)[i].FieldName) == 0 {
			errors = append(errors, fmt.Sprintf("dev error, empty FieldRef %s.%s",
				(*fieldRefsToCheck)[i].TableName, (*fieldRefsToCheck)[i].FieldName))
		}
		hash := fmt.Sprintf("%s.%s", (*fieldRefsToCheck)[i].TableName, (*fieldRefsToCheck)[i].FieldName)
		if _, ok := prohibitedHashes[hash]; ok {
			errors = append(errors, fmt.Sprintf("prohibited field %s.%s", (*fieldRefsToCheck)[i].TableName, (*fieldRefsToCheck)[i].FieldName))
		} else if _, ok := allowedHashes[hash]; !ok {
			errors = append(errors, fmt.Sprintf("unknown field %s.%s", (*fieldRefsToCheck)[i].TableName, (*fieldRefsToCheck)[i].FieldName))
		} else {
			// Update check field type, we will use it later for test eval
			(*fieldRefsToCheck)[i].FieldType = allowedHashes[hash].FieldType
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	} else {
		return nil
	}

}

type FieldRefParserFlag uint32

func (f FieldRefParserFlag) HasFlag(flag FieldRefParserFlag) bool { return f&flag != 0 }

// Not used for now, maybe later
// func (f *FieldRefParserFlag) AddFlag(flag FieldRefParserFlag)     { *f |= flag }
// func (f *FieldRefParserFlag) ClearFlag(flag FieldRefParserFlag)   { *f &= ^flag }
// func (f *FieldRefParserFlag) ToggleFlag(flag FieldRefParserFlag)  { *f ^= flag }

const (
	FieldRefStrict             FieldRefParserFlag = 0
	FieldRefAllowUnknownIdents FieldRefParserFlag = 1 << iota
	FieldRefAllowWhateverFeatureYouAreAddingHere
)

func harvestFieldRefsFromParsedExpression(exp ast.Expr, usedFields *FieldRefs, parserFlags FieldRefParserFlag) error {
	switch assertedExp := exp.(type) {
	case *ast.BinaryExpr:
		if err := harvestFieldRefsFromParsedExpression(assertedExp.X, usedFields, parserFlags); err != nil {
			return err
		}
		return harvestFieldRefsFromParsedExpression(assertedExp.Y, usedFields, parserFlags)

	case *ast.UnaryExpr:
		return harvestFieldRefsFromParsedExpression(assertedExp.X, usedFields, parserFlags)

	case *ast.CallExpr:
		for _, v := range assertedExp.Args {
			if err := harvestFieldRefsFromParsedExpression(v, usedFields, parserFlags); err != nil {
				return err
			}
		}

	case *ast.SelectorExpr:
		switch assertedExpIdent := assertedExp.X.(type) {
		case *ast.Ident:
			_, ok := eval.GolangConstants[fmt.Sprintf("%s.%s", assertedExpIdent.Name, assertedExp.Sel.Name)]
			if !ok {
				usedFields.contributeUnresolved(assertedExpIdent.Name, assertedExp.Sel.Name)
			}
		default:
			return fmt.Errorf("selectors starting with non-ident are not allowed, found '%v'; aliases to use: readers - '%s', creators - '%s', custom processors - '%s', lookups - '%s'",
				assertedExp.X, ReaderAlias, CreatorAlias, CustomProcessorAlias, LookupAlias)
		}

	case *ast.Ident:
		// Keep in mind we may use this parser for Python expressions. Allow unknown constructs for those cases.
		if !parserFlags.HasFlag(FieldRefAllowUnknownIdents) {
			if assertedExp.Name != "true" && assertedExp.Name != "false" {
				return fmt.Errorf("plain (non-selector) identifiers are not allowed, expected field qualifiers (tableor_lkp_alias.field_name), found '%s'; for file readers, use '%s' alias; for file creators, use '%s' alias",
					assertedExp.Name, ReaderAlias, CreatorAlias)
			}
		}
	}

	return nil
}

func ParseRawGolangExpressionStringAndHarvestFieldRefs(strExp string, usedFields *FieldRefs) (ast.Expr, error) {
	if len(strings.TrimSpace(strExp)) == 0 {
		return nil, nil
	}

	expCondition, err := parser.ParseExpr(strExp)
	if err != nil {
		return nil, fmt.Errorf("strict parsing error: [%s]", err.Error())
	}

	if usedFields != nil {
		if err := harvestFieldRefsFromParsedExpression(expCondition, usedFields, FieldRefStrict); err != nil {
			return nil, err
		}
	}

	return expCondition, nil
}

func ParseRawRelaxedGolangExpressionStringAndHarvestFieldRefs(strExp string, usedFields *FieldRefs, parserFlags FieldRefParserFlag) (ast.Expr, error) {
	if len(strings.TrimSpace(strExp)) == 0 {
		return nil, nil
	}

	expCondition, err := parser.ParseExpr(strExp)
	if err != nil {
		return nil, fmt.Errorf("relaxed parsing error: [%s]", err.Error())
	}

	if usedFields != nil {
		if err := harvestFieldRefsFromParsedExpression(expCondition, usedFields, parserFlags); err != nil {
			return nil, err
		}
	}

	return expCondition, nil
}
