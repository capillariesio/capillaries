package custom

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/xfer"
)

const ProcessorTagAndDenormalizeName string = "tag_and_denormalize"

type TagAndDenormalizeProcessorDef struct {
	TagFieldName         string            `json:"tag_field_name"`
	RawTagCriteria       map[string]string `json:"tag_criteria"`
	RawTagCriteriaUri    string            `json:"tag_criteria_uri"`
	ParsedTagCriteria    map[string]ast.Expr
	UsedInCriteriaFields sc.FieldRefs
}

func (procDef *TagAndDenormalizeProcessorDef) GetFieldRefs() *sc.FieldRefs {
	return &sc.FieldRefs{
		{
			TableName: sc.CustomProcessorAlias,
			FieldName: procDef.TagFieldName,
			FieldType: sc.FieldTypeString}}
}

func (procDef *TagAndDenormalizeProcessorDef) GetUsedInTargetExpressionsFields() *sc.FieldRefs {
	return &procDef.UsedInCriteriaFields
}

func (procDef *TagAndDenormalizeProcessorDef) Deserialize(raw json.RawMessage, customProcSettings json.RawMessage, caPath string, privateKeys map[string]string) error {
	var err error
	if err = json.Unmarshal(raw, procDef); err != nil {
		return fmt.Errorf("cannot unmarshal tag_and_denormalize processor def: %s", err.Error())
	}

	errors := make([]string, 0)
	procDef.ParsedTagCriteria = map[string]ast.Expr{}

	if len(procDef.RawTagCriteriaUri) > 0 {
		if len(procDef.RawTagCriteria) > 0 {
			return fmt.Errorf("cannot unmarshal both tag_criteria and tag_criteria_url - pick one")
		}

		criteriaBytes, err := xfer.GetFileBytes(procDef.RawTagCriteriaUri, caPath, privateKeys)
		if err != nil {
			return fmt.Errorf("cannot get criteria file [%s]: %s", procDef.RawTagCriteriaUri, err.Error())
		}

		if criteriaBytes == nil || len(criteriaBytes) == 0 {
			return fmt.Errorf("criteria file [%s] is empty", procDef.RawTagCriteriaUri)
		}

		if criteriaBytes != nil {
			if err := json.Unmarshal(criteriaBytes, &procDef.RawTagCriteria); err != nil {
				return fmt.Errorf("cannot unmarshal tag criteria file [%s]: [%s]", procDef.RawTagCriteriaUri, err.Error())
			}
		}
	} else if len(procDef.RawTagCriteria) == 0 {
		return fmt.Errorf("cannot unmarshal with tag_criteria and tag_criteria_url missing")
	}

	for tag, rawExp := range procDef.RawTagCriteria {
		if procDef.ParsedTagCriteria[tag], err = sc.ParseRawGolangExpressionStringAndHarvestFieldRefs(rawExp, &procDef.UsedInCriteriaFields); err != nil {
			errors = append(errors, fmt.Sprintf("cannot parse tag criteria expression [%s]: [%s]", rawExp, err.Error()))
		}
	}

	// Later on, checkFieldUsageInCustomProcessor() will verify all fields from procDef.UsedInCriteriaFields are valid reader fields

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	} else {
		return nil
	}
}

const tagAndDenormalizeFlushBufferSize int = 1000

func (procDef *TagAndDenormalizeProcessorDef) tagAndDenormalize(rsIn *proc.Rowset, flushVarsArray func(varsArray []*eval.VarValuesMap, varsArrayCount int) error) error {
	varsArray := make([]*eval.VarValuesMap, tagAndDenormalizeFlushBufferSize)
	varsArrayCount := 0

	for rowIdx := 0; rowIdx < rsIn.RowCount; rowIdx++ {
		vars := eval.VarValuesMap{}
		if err := rsIn.ExportToVars(rowIdx, &vars); err != nil {
			return err
		}

		for tag, tagCriteria := range procDef.ParsedTagCriteria {
			eCtx := eval.NewPlainEvalCtxWithVars(eval.AggFuncDisabled, &vars)
			valVolatile, err := eCtx.Eval(tagCriteria)
			if err != nil {
				return fmt.Errorf("cannot evaluate expression for tag %s criteria: [%s]", tag, err.Error())
			}
			valBool, ok := valVolatile.(bool)
			if !ok {
				return fmt.Errorf("tag %s criteria returned type %T, expected bool", tag, valVolatile)
			}

			if !valBool {
				// This tag criteria were not met, skip it
				continue
			}

			// Add new tag field to the output

			varsArray[varsArrayCount] = &eval.VarValuesMap{}
			// Write tag
			(*varsArray[varsArrayCount])[sc.CustomProcessorAlias] = map[string]interface{}{procDef.TagFieldName: tag}
			// Write r values
			(*varsArray[varsArrayCount])[sc.ReaderAlias] = map[string]interface{}{}
			for fieldName, fieldVal := range vars[sc.ReaderAlias] {
				(*varsArray[varsArrayCount])[sc.ReaderAlias][fieldName] = fieldVal
			}
			varsArrayCount++

			if varsArrayCount == len(varsArray) {
				if err = flushVarsArray(varsArray, varsArrayCount); err != nil {
					return fmt.Errorf("error flushing vars array of size %d: %s", varsArrayCount, err.Error())
				}
				varsArray = make([]*eval.VarValuesMap, tagAndDenormalizeFlushBufferSize)
				varsArrayCount = 0
			}
		}
	}

	if varsArrayCount > 0 {
		if err := flushVarsArray(varsArray, varsArrayCount); err != nil {
			return fmt.Errorf("error flushing leftovers vars array of size %d: %s", varsArrayCount, err.Error())
		}
	}

	return nil
}
