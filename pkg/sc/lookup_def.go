package sc

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval"
)

type LookupJoinType string

const (
	LookupJoinInner LookupJoinType = "inner"
	LookupJoinLeft  LookupJoinType = "left"
)

type LookupDef struct {
	IndexName                string         `json:"index_name"`
	RawJoinOn                string         `json:"join_on"`
	IsGroup                  bool           `json:"group"`
	RawFilter                string         `json:"filter"`
	LookupJoin               LookupJoinType `json:"join_type"`
	IdxReadBatchSize         int            `json:"idx_read_batch_size"`
	RightLookupReadBatchSize int            `json:"right_lookup_read_batch_size"`

	LeftTableFields    FieldRefs        // In the same order as lookup idx - important
	TableCreator       *TableCreatorDef // Populated when walking through al nodes
	UsedInFilterFields FieldRefs
	Filter             ast.Expr
}

const (
	defaultIdxBatchSize         int = 3000
	maxIdxBatchSize             int = 5000
	defaultRightLookupBatchSize int = 3000
	maxRightLookupReadBatchSize int = 5000
)

func (lkpDef *LookupDef) CheckPagedBatchSize() error {
	// Default gocql iterator page size is 5000, do not exceed it.
	// Actually, getting close to it (4000) causes problems on small servers
	if lkpDef.IdxReadBatchSize <= 0 {
		lkpDef.IdxReadBatchSize = defaultIdxBatchSize
	} else if lkpDef.IdxReadBatchSize > maxIdxBatchSize {
		return fmt.Errorf("cannot use idx_read_batch_size %d, expected <= %d, default %d, ", lkpDef.IdxReadBatchSize, maxIdxBatchSize, defaultIdxBatchSize)
	}
	if lkpDef.RightLookupReadBatchSize <= 0 {
		lkpDef.RightLookupReadBatchSize = defaultRightLookupBatchSize
	} else if lkpDef.RightLookupReadBatchSize > maxRightLookupReadBatchSize {
		return fmt.Errorf("cannot use right_lookup_read_batch_size %d, expected <= %d, default %d, ", lkpDef.RightLookupReadBatchSize, maxRightLookupReadBatchSize, defaultRightLookupBatchSize)
	}
	return nil
}

func (lkpDef *LookupDef) UsesFilter() bool {
	return len(strings.TrimSpace(lkpDef.RawFilter)) > 0
}

func (lkpDef *LookupDef) ValidateJoinType() error {
	if lkpDef.LookupJoin != LookupJoinLeft && lkpDef.LookupJoin != LookupJoinInner {
		return fmt.Errorf("invalid join type, expected inner or left, %s is not supported", lkpDef.LookupJoin)
	}
	return nil
}

func (lkpDef *LookupDef) ParseFilter() error {
	if !lkpDef.UsesFilter() {
		return nil
	}
	var err error
	lkpDef.Filter, err = ParseRawGolangExpressionStringAndHarvestFieldRefs(lkpDef.RawFilter, &lkpDef.UsedInFilterFields)
	if err != nil {
		return fmt.Errorf("cannot parse lookup filter condition [%s]: %s", lkpDef.RawFilter, err.Error())
	}
	return nil
}

func (lkpDef *LookupDef) resolveLeftTableFields(srcName string, srcFieldRefs *FieldRefs) error {
	fieldExpressions := strings.Split(lkpDef.RawJoinOn, ",")
	lkpDef.LeftTableFields = make(FieldRefs, len(fieldExpressions))
	for fieldIdx := 0; fieldIdx < len(fieldExpressions); fieldIdx++ {
		fieldNameParts := strings.Split(strings.TrimSpace(fieldExpressions[fieldIdx]), ".")
		if len(fieldNameParts) != 2 {
			return fmt.Errorf("expected a comma-separated list of <table_name>.<field_name>, got [%s]", lkpDef.RawJoinOn)
		}
		tName := strings.TrimSpace(fieldNameParts[0])
		fName := strings.TrimSpace(fieldNameParts[1])
		if tName != srcName {
			return fmt.Errorf("source table name [%s] unknown, expected [%s]", tName, srcName)
		}
		srcFieldRef, ok := srcFieldRefs.FindByFieldName(fName)
		if !ok {
			return fmt.Errorf("source [%s] does not produce field [%s]", tName, fName)
		}
		if srcFieldRef.FieldType == FieldTypeUnknown {
			return fmt.Errorf("source field [%s.%s] has unknown type", tName, fName)
		}
		lkpDef.LeftTableFields[fieldIdx] = *srcFieldRef
	}

	// Verify lookup idx has this field and the type matches
	idxFieldRefs := lkpDef.TableCreator.Indexes[lkpDef.IndexName].getComponentFieldRefs(lkpDef.TableCreator.Name)
	if len(idxFieldRefs) != len(lkpDef.LeftTableFields) {
		return fmt.Errorf("lookup joins on %d fields, while referenced index %s uses %d fields, these lengths need to be the same", len(lkpDef.LeftTableFields), lkpDef.IndexName, len(idxFieldRefs))
	}

	for fieldIdx := 0; fieldIdx < len(lkpDef.LeftTableFields); fieldIdx++ {
		if lkpDef.LeftTableFields[fieldIdx].FieldType != idxFieldRefs[fieldIdx].FieldType {
			return fmt.Errorf("left-side field %s has type %s, while index field %s has type %s",
				lkpDef.LeftTableFields[fieldIdx].FieldName,
				lkpDef.LeftTableFields[fieldIdx].FieldType,
				idxFieldRefs[fieldIdx].FieldName,
				idxFieldRefs[fieldIdx].FieldType)
		}
	}

	return nil
}

func (lkpDef *LookupDef) CheckFilterCondition(varsFromLookup eval.VarValuesMap) (bool, error) {
	if !lkpDef.UsesFilter() {
		return true, nil
	}
	eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, &varsFromLookup)
	valVolatile, err := eCtx.Eval(lkpDef.Filter)
	if err != nil {
		return false, fmt.Errorf("cannot evaluate expression: [%s]", err.Error())
	}
	valBool, ok := valVolatile.(bool)
	if !ok {
		return false, fmt.Errorf("cannot evaluate lookup filter condition expression, expected bool, got %v(%T) instead", valVolatile, valVolatile)
	}

	return valBool, nil
}
