package sc

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

const (
	DefaultStringComponentLen int64 = 64
	MinStringComponentLen     int64 = 16
	MaxStringComponentLen     int64 = 1024
)

type IdxSortOrder string

const (
	IdxSortAsc     IdxSortOrder = "asc"
	IdxSortDesc    IdxSortOrder = "desc"
	IdxSortUnknown IdxSortOrder = "unknown"
)

type IdxCaseSensitivity string

const (
	IdxCaseSensitive          IdxCaseSensitivity = "case_sensitive"
	IdxIgnoreCase             IdxCaseSensitivity = "ignore_case"
	IdxCaseSensitivityUnknown IdxCaseSensitivity = "case_sensitivity_unknown"
)

type IdxComponentDef struct {
	FieldName       string
	CaseSensitivity IdxCaseSensitivity
	SortOrder       IdxSortOrder
	StringLen       int64          // For string fields only, default 64
	FieldType       TableFieldType // Populated from tgt_table def
}

type IdxUniqueness string

const (
	IdxUnique            IdxUniqueness = "unique"
	IdxNonUnique         IdxUniqueness = "non_unique"
	IdxUniquenessUnknown IdxUniqueness = "unknown"
)

type IdxDef struct {
	Uniqueness IdxUniqueness
	Components []IdxComponentDef
}

type IdxDefMap map[string]*IdxDef

type IndexRef struct {
	TableName string
	IdxName   string
}

func (idxDef *IdxDef) getComponentFieldRefs(tableName string) FieldRefs {
	fieldRefs := make([]FieldRef, len(idxDef.Components))
	for i := 0; i < len(idxDef.Components); i++ {
		fieldRefs[i] = FieldRef{
			TableName: tableName,
			FieldName: idxDef.Components[i].FieldName,
			FieldType: idxDef.Components[i].FieldType}
	}
	return fieldRefs
}

func (idxDef *IdxDef) parseComponentExpr(fldExp *ast.Expr, fieldRefs *FieldRefs) error {
	// Initialize index component with defaults and append it to idx def
	idxCompDef := IdxComponentDef{
		FieldName:       FieldNameUnknown,
		CaseSensitivity: IdxCaseSensitivityUnknown,
		SortOrder:       IdxSortUnknown,
		StringLen:       DefaultStringComponentLen, // Users can override it, see below
		FieldType:       FieldTypeUnknown}

	switch typedFldExp := (*fldExp).(type) {
	case *ast.CallExpr:
		identExp, ok := typedFldExp.Fun.(*ast.Ident)
		if !ok {
			return fmt.Errorf("cannot parse order component func expression, field %s is not an ident", identExp.Name)
		}
		fieldRef, ok := fieldRefs.FindByFieldName(identExp.Name)
		if !ok {
			return fmt.Errorf("cannot parse order component func expression, field %s unknown", identExp.Name)
		}

		// Defaults
		idxCompDef.FieldType = (*fieldRef).FieldType
		idxCompDef.FieldName = identExp.Name

		// Parse args: asc/desc, case_sensitive/ignore_case, number-string length
		for _, modifierExp := range typedFldExp.Args {
			switch modifierExpType := modifierExp.(type) {
			case *ast.Ident:
				switch modifierExpType.Name {
				case string(IdxCaseSensitive):
					idxCompDef.CaseSensitivity = IdxCaseSensitive
				case string(IdxIgnoreCase):
					idxCompDef.CaseSensitivity = IdxIgnoreCase
				case string(IdxSortAsc):
					idxCompDef.SortOrder = IdxSortAsc
				case string(IdxSortDesc):
					idxCompDef.SortOrder = IdxSortDesc
				default:
					return fmt.Errorf(
						"unknown modifier %s for field %s, expected %s,%s,%s,%s",
						modifierExpType.Name, identExp.Name, IdxIgnoreCase, IdxCaseSensitive, IdxSortAsc, IdxSortDesc)
				}
			case *ast.BasicLit:
				switch modifierExpType.Kind {
				case token.INT:
					if idxCompDef.FieldType != FieldTypeString {
						return fmt.Errorf("invalid expression %v in %s, component length modifier is valid only for string fields, but %s has type %s",
							modifierExpType, identExp.Name, idxCompDef.FieldName, idxCompDef.FieldType)
					}
					idxCompDef.StringLen, _ = strconv.ParseInt(modifierExpType.Value, 10, 64)
					if idxCompDef.StringLen < MinStringComponentLen {
						idxCompDef.StringLen = MinStringComponentLen
					} else if idxCompDef.StringLen > MaxStringComponentLen {
						return fmt.Errorf("invalid expression %v in %s, component length modifier for string fields cannot exceed %d",
							modifierExpType, identExp.Name, MaxStringComponentLen)
					}
				default:
					return fmt.Errorf("invalid expression %v in %s, expected an integer for string component length", modifierExpType, identExp.Name)
				}

			default:
				return fmt.Errorf(
					"invalid expression %v, expected a modifier for field %s: expected %s,%s,%s,%s or an integer",
					modifierExpType, identExp.Name, IdxIgnoreCase, IdxCaseSensitive, IdxSortAsc, IdxSortDesc)
			}

			// Check some rules
			if idxCompDef.FieldType != FieldTypeString && idxCompDef.CaseSensitivity != IdxCaseSensitivityUnknown {
				return fmt.Errorf(
					"index component for field %s of type %s cannot have case sensitivity modifier %s, remove it from index component definition",
					identExp.Name, idxCompDef.FieldType, idxCompDef.CaseSensitivity)
			}
		}

	case *ast.Ident:
		// This is a component def without modifiers (not filed1(...), just field1), so just apply defaults
		fieldRef, ok := fieldRefs.FindByFieldName(typedFldExp.Name)
		if !ok {
			return fmt.Errorf("cannot parse order component ident expression, field %s unknown", typedFldExp.Name)
		}

		// Defaults
		idxCompDef.FieldType = (*fieldRef).FieldType
		idxCompDef.FieldName = typedFldExp.Name

	default:
		return fmt.Errorf(
			"invalid expression in index component definition, expected 'field([modifiers])' or 'field' where 'field' is one of the fields of the table created by this node")
	}

	// Apply defaults if no modifiers supplied: string -> case sensitive, ordered idx -> sort asc
	if idxCompDef.FieldType == FieldTypeString && idxCompDef.CaseSensitivity == IdxCaseSensitivityUnknown {
		idxCompDef.CaseSensitivity = IdxCaseSensitive
	}

	if idxCompDef.SortOrder == IdxSortUnknown {
		idxCompDef.SortOrder = IdxSortAsc
	}

	// All good - add it to the list of idx components
	idxDef.Components = append(idxDef.Components, idxCompDef)

	return nil
}

func (idxDefMap *IdxDefMap) parseRawIndexDefMap(rawIdxDefMap map[string]string, fieldRefs *FieldRefs) error {
	errors := make([]string, 0, 10)

	for idxName, rawIdxDef := range rawIdxDefMap {

		expIdxDef, err := parser.ParseExpr(rawIdxDef)
		if err != nil {
			return fmt.Errorf("cannot parse order def '%s': %v", rawIdxDef, err)
		}
		switch typedExp := expIdxDef.(type) {
		case *ast.CallExpr:
			identExp, ok := typedExp.Fun.(*ast.Ident)
			if !ok {
				return fmt.Errorf("cannot parse call exp %v, expected ident", typedExp.Fun)
			}

			// Init idx def, defaults here if needed
			idxDef := IdxDef{Uniqueness: IdxUniquenessUnknown}

			switch identExp.Name {
			case string(IdxUnique):
				idxDef.Uniqueness = IdxUnique
			case string(IdxNonUnique):
				idxDef.Uniqueness = IdxNonUnique
			default:
				return fmt.Errorf(
					"cannot parse index def [%s]: expected top level unique()) or non_unique() definition, found %s",
					rawIdxDef, identExp.Name)
			}

			// Walk through args - idx field components
			for _, fldExp := range typedExp.Args {
				err := idxDef.parseComponentExpr(&fldExp, fieldRefs)
				if err != nil {
					errors = append(errors, fmt.Sprintf("index %s: [%s]", rawIdxDef, err.Error()))
				}
			}

			// All good - add it to the idx map of the table def
			(*idxDefMap)[idxName] = &idxDef

		default:
			return fmt.Errorf(
				"cannot parse index def [%s]: expected top level unique()) or non_unique() definition, found unknown expression",
				rawIdxDef)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cannot parse order definitions: [%s]", strings.Join(errors, "; "))
	}

	return nil
}
